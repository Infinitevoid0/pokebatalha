// regras de jogadores: conta, gacha, validar time e creditar recompensa.
package servico

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"pokebatalha/services/jogadores/internal/dominio"
)

// banco de jogadores (implementado em repositorio.Postgres).
type Repositorio interface {
	Criar(ctx context.Context, nome string, moedas int) (dominio.Jogador, error)
	BuscarPorID(ctx context.Context, id int64) (dominio.Jogador, error)
	ListarColecao(ctx context.Context, jogadorID int64) ([]dominio.ItemColecao, error)
	RegistrarCompra(ctx context.Context, jogadorID int64, custo int, pityLido, pityNovo dominio.Pity, p dominio.PokemonSorteado) (dominio.ItemColecao, int, error)
	Creditar(ctx context.Context, jogadorID int64, quantidade int) (int, error)
	PokemonsFaltando(ctx context.Context, jogadorID int64, pokemonIDs []int) ([]int, error)
}

// catálogo via gRPC (implementado em catalogo.Cliente).
type Catalogo interface {
	SortearPokemon(ctx context.Context, tier string) (dominio.PokemonSorteado, error)
	BuscarDetalhes(ctx context.Context, ids []int) (map[int]dominio.Detalhes, error)
}

// valores de economia, vindos da config.
type Regras struct {
	MoedasIniciais int
	PrecoPacote    int
}

type Servico struct {
	repo     Repositorio
	catalogo Catalogo
	sorteio  dominio.Sorteador
	regras   Regras
}

func Novo(repo Repositorio, cat Catalogo, s dominio.Sorteador, regras Regras) *Servico {
	return &Servico{repo: repo, catalogo: cat, sorteio: s, regras: regras}
}

// cria a conta com o saldo inicial.
func (s *Servico) CriarJogador(ctx context.Context, nome string) (dominio.Jogador, error) {
	nome, err := dominio.NormalizarNome(nome)
	if err != nil {
		return dominio.Jogador{}, err
	}
	return s.repo.Criar(ctx, nome, s.regras.MoedasIniciais)
}

// busca a conta pelo ID.
func (s *Servico) BuscarJogador(ctx context.Context, id int64) (dominio.Jogador, error) {
	return s.repo.BuscarPorID(ctx, id)
}

// coleção com os dados atuais de cada Pokémon.
type Colecao struct {
	Pokemons []dominio.ItemColecao `json:"pokemons"`
	// catálogo fora: a coleção vem sem stats em vez de falhar.
	DetalhesIndisponiveis bool `json:"detalhes_indisponiveis,omitempty"`
}

// lista do mais forte ao mais fraco.
func (s *Servico) Colecao(ctx context.Context, id int64) (Colecao, error) {
	if _, err := s.repo.BuscarPorID(ctx, id); err != nil {
		return Colecao{}, err
	}
	itens, err := s.repo.ListarColecao(ctx, id)
	if err != nil {
		return Colecao{}, err
	}

	var ids []int
	vistos := map[int]bool{}
	for _, it := range itens {
		if !vistos[it.PokemonID] {
			vistos[it.PokemonID] = true
			ids = append(ids, it.PokemonID)
		}
	}
	detalhes, err := s.catalogo.BuscarDetalhes(ctx, ids)
	if errors.Is(err, dominio.ErrServicoIndisponivel) {
		return Colecao{Pokemons: itens, DetalhesIndisponiveis: true}, nil
	}
	if err != nil {
		return Colecao{}, err
	}

	for i := range itens {
		if d, ok := detalhes[itens[i].PokemonID]; ok {
			itens[i].Detalhes = &d
			itens[i].Tier = d.Tier // o tier atual, calculado pelo índice de poder
		}
	}
	sort.SliceStable(itens, func(a, b int) bool { return poderDe(itens[a]) > poderDe(itens[b]) })
	return Colecao{Pokemons: itens}, nil
}

func poderDe(it dominio.ItemColecao) int {
	if it.Detalhes == nil {
		return 0
	}
	return it.Detalhes.Poder.Total
}

// o que o jogador recebe ao abrir um pacote.
type ResultadoPacote struct {
	Pokemon   dominio.ItemColecao `json:"pokemon"`
	Saldo     int                 `json:"saldo"`
	Garantido bool                `json:"garantido_pelo_pity"`
	Pity      dominio.InfoPity    `json:"pity"`
}

// cobra, sorteia o tier e pede ao catálogo um Pokémon dele.
func (s *Servico) AbrirPacote(ctx context.Context, id int64) (ResultadoPacote, error) {
	j, err := s.repo.BuscarPorID(ctx, id)
	if err != nil {
		return ResultadoPacote{}, err
	}
	// checagem antecipada para não chamar o catálogo à toa;
	// a garantia real é o UPDATE condicional do repositório.
	if j.Moedas < s.regras.PrecoPacote {
		return ResultadoPacote{}, fmt.Errorf("%w: pacote custa %d, você tem %d",
			dominio.ErrSaldoInsuficiente, s.regras.PrecoPacote, j.Moedas)
	}

	sorteio := dominio.SortearComPity(s.sorteio, j.Pity)
	p, err := s.catalogo.SortearPokemon(ctx, sorteio.Tier)
	if err != nil {
		return ResultadoPacote{}, err
	}

	item, saldo, err := s.repo.RegistrarCompra(ctx, id, s.regras.PrecoPacote, j.Pity, sorteio.Pity, p)
	if err != nil {
		return ResultadoPacote{}, err
	}
	return ResultadoPacote{Pokemon: item, Saldo: saldo, Garantido: sorteio.Garantido, Pity: sorteio.Pity.Info()}, nil
}

// confere se o jogador tem todos os Pokémon do time.
func (s *Servico) ValidarTime(ctx context.Context, id int64, pokemonIDs []int) error {
	if len(pokemonIDs) == 0 {
		return fmt.Errorf("%w: o time está vazio", dominio.ErrTimeInvalido)
	}
	if _, err := s.repo.BuscarPorID(ctx, id); err != nil {
		return err
	}
	faltando, err := s.repo.PokemonsFaltando(ctx, id, pokemonIDs)
	if err != nil {
		return err
	}
	if len(faltando) > 0 {
		return fmt.Errorf("%w: ids %v", dominio.ErrPokemonNaoPertence, faltando)
	}
	return nil
}

// soma a recompensa ao saldo.
func (s *Servico) CreditarMoedas(ctx context.Context, id int64, quantidade int) (int, error) {
	if quantidade <= 0 {
		return 0, fmt.Errorf("%w: %d", dominio.ErrQuantidadeInvalida, quantidade)
	}
	return s.repo.Creditar(ctx, id, quantidade)
}
