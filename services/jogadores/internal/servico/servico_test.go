package servico

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"pokebatalha/services/jogadores/internal/dominio"
)

// tudo em mapas, só para os testes.
type repoMemoria struct {
	jogadores map[int64]*dominio.Jogador
	colecao   map[int64][]dominio.ItemColecao
}

func novoRepo(jogadores ...dominio.Jogador) *repoMemoria {
	r := &repoMemoria{jogadores: map[int64]*dominio.Jogador{}, colecao: map[int64][]dominio.ItemColecao{}}
	for _, j := range jogadores {
		r.jogadores[j.ID] = &j
	}
	return r
}

func (r *repoMemoria) Criar(_ context.Context, nome string, moedas int) (dominio.Jogador, error) {
	j := dominio.Jogador{ID: int64(len(r.jogadores) + 1), Nome: nome, Moedas: moedas}
	r.jogadores[j.ID] = &j
	return j, nil
}

func (r *repoMemoria) BuscarPorID(_ context.Context, id int64) (dominio.Jogador, error) {
	j, ok := r.jogadores[id]
	if !ok {
		return dominio.Jogador{}, dominio.ErrJogadorNaoEncontrado
	}
	return *j, nil
}

func (r *repoMemoria) ListarColecao(_ context.Context, id int64) ([]dominio.ItemColecao, error) {
	return r.colecao[id], nil
}

func (r *repoMemoria) RegistrarCompra(_ context.Context, id int64, custo int, lido, novo dominio.Pity, p dominio.PokemonSorteado) (dominio.ItemColecao, int, error) {
	j := r.jogadores[id]
	if j.Moedas < custo {
		return dominio.ItemColecao{}, 0, dominio.ErrSaldoInsuficiente
	}
	if j.Pity != lido {
		return dominio.ItemColecao{}, 0, dominio.ErrCompraSimultanea
	}
	j.Moedas -= custo
	j.Pity = novo
	it := dominio.ItemColecao{PokemonID: p.PokemonID, Nome: p.Nome, Tier: p.Detalhes.Tier}
	r.colecao[id] = append(r.colecao[id], it)
	return it, j.Moedas, nil
}

func (r *repoMemoria) Creditar(_ context.Context, id int64, qtd int) (int, error) {
	j, ok := r.jogadores[id]
	if !ok {
		return 0, dominio.ErrJogadorNaoEncontrado
	}
	j.Moedas += qtd
	return j.Moedas, nil
}

func (r *repoMemoria) PokemonsFaltando(_ context.Context, id int64, ids []int) ([]int, error) {
	var faltando []int
	for _, pid := range ids {
		possui := slices.ContainsFunc(r.colecao[id], func(it dominio.ItemColecao) bool { return it.PokemonID == pid })
		if !possui {
			faltando = append(faltando, pid)
		}
	}
	return faltando, nil
}

// registra o tier pedido e devolve um Pokémon fixo.
type catalogoFalso struct {
	erro       error
	tierPedido string
	chamadas   int
}

func (c *catalogoFalso) SortearPokemon(_ context.Context, tier string) (dominio.PokemonSorteado, error) {
	c.chamadas++
	c.tierPedido = tier
	if c.erro != nil {
		return dominio.PokemonSorteado{}, c.erro
	}
	return dominio.PokemonSorteado{PokemonID: 727, Nome: "incineroar", Detalhes: dominio.Detalhes{Tier: tier}}, nil
}

func (c *catalogoFalso) BuscarDetalhes(_ context.Context, ids []int) (map[int]dominio.Detalhes, error) {
	if c.erro != nil {
		return nil, c.erro
	}
	m := map[int]dominio.Detalhes{}
	for _, id := range ids {
		m[id] = dominio.Detalhes{Tier: "raro", Poder: dominio.ComposicaoPoder{Total: id}}
	}
	return m, nil
}

type rolagemFixa int

func (r rolagemFixa) IntN(int) int { return int(r) }

var regras = Regras{MoedasIniciais: 100, PrecoPacote: 30}

func TestAbrirPacote_CobraEEntrega(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo", Moedas: 100})
	cat := &catalogoFalso{}
	svc := Novo(repo, cat, rolagemFixa(0), regras) // rolagem 0 = lendário

	res, err := svc.AbrirPacote(context.Background(), 1)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cat.tierPedido != "lendario" {
		t.Errorf("deveria pedir tier lendario ao catálogo, pediu %q", cat.tierPedido)
	}
	if res.Saldo != 70 || res.Pokemon.Nome != "incineroar" {
		t.Errorf("resultado errado: %+v", res)
	}
}

func TestAbrirPacote_SemSaldoNemChamaOCatalogo(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo", Moedas: 10})
	cat := &catalogoFalso{}
	svc := Novo(repo, cat, rolagemFixa(50), regras)

	_, err := svc.AbrirPacote(context.Background(), 1)
	if !errors.Is(err, dominio.ErrSaldoInsuficiente) {
		t.Fatalf("esperava ErrSaldoInsuficiente, veio %v", err)
	}
	if cat.chamadas != 0 {
		t.Fatal("não deveria chamar o catálogo sem saldo")
	}
}

func TestAbrirPacote_CatalogoForaNaoCobra(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo", Moedas: 100})
	cat := &catalogoFalso{erro: fmt.Errorf("%w: catálogo fora", dominio.ErrServicoIndisponivel)}
	svc := Novo(repo, cat, rolagemFixa(50), regras)

	_, err := svc.AbrirPacote(context.Background(), 1)
	if !errors.Is(err, dominio.ErrServicoIndisponivel) {
		t.Fatalf("esperava ErrServicoIndisponivel, veio %v", err)
	}
	if repo.jogadores[1].Moedas != 100 {
		t.Fatal("o jogador não pode perder moedas se o pacote falhou")
	}
}

func TestAbrirPacote_JogadorInexistente(t *testing.T) {
	svc := Novo(novoRepo(), &catalogoFalso{}, rolagemFixa(50), regras)
	if _, err := svc.AbrirPacote(context.Background(), 99); !errors.Is(err, dominio.ErrJogadorNaoEncontrado) {
		t.Fatalf("esperava ErrJogadorNaoEncontrado, veio %v", err)
	}
}

func TestValidarTime(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo", Moedas: 100})
	repo.colecao[1] = []dominio.ItemColecao{{PokemonID: 727}, {PokemonID: 6}}
	svc := Novo(repo, &catalogoFalso{}, rolagemFixa(0), regras)
	ctx := context.Background()

	if err := svc.ValidarTime(ctx, 1, []int{727, 6}); err != nil {
		t.Errorf("time válido foi rejeitado: %v", err)
	}
	if err := svc.ValidarTime(ctx, 1, []int{727, 150}); !errors.Is(err, dominio.ErrPokemonNaoPertence) {
		t.Errorf("esperava ErrPokemonNaoPertence, veio %v", err)
	}
	if err := svc.ValidarTime(ctx, 1, nil); !errors.Is(err, dominio.ErrTimeInvalido) {
		t.Errorf("esperava ErrTimeInvalido, veio %v", err)
	}
	if err := svc.ValidarTime(ctx, 99, []int{727}); !errors.Is(err, dominio.ErrJogadorNaoEncontrado) {
		t.Errorf("esperava ErrJogadorNaoEncontrado, veio %v", err)
	}
}

func TestCreditarMoedas_QuantidadeInvalida(t *testing.T) {
	svc := Novo(novoRepo(dominio.Jogador{ID: 1}), &catalogoFalso{}, rolagemFixa(0), regras)
	if _, err := svc.CreditarMoedas(context.Background(), 1, -5); !errors.Is(err, dominio.ErrQuantidadeInvalida) {
		t.Fatalf("esperava ErrQuantidadeInvalida, veio %v", err)
	}
}

func TestAbrirPacote_PityGaranteEpicoNoDecimo(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo", Moedas: 1000, Pity: dominio.Pity{SemEpico: 9, SemLendario: 9}})
	cat := &catalogoFalso{}
	svc := Novo(repo, cat, rolagemFixa(50), regras) // 50 seria raro; o pity força épico+

	res, err := svc.AbrirPacote(context.Background(), 1)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !res.Garantido || cat.tierPedido != "epico" {
		t.Fatalf("o 10º pacote deveria ser épico garantido (garantido=%v, tier=%s)", res.Garantido, cat.tierPedido)
	}
	if res.Pity.SemEpico != 0 || res.Pity.SemLendario != 10 || res.Pity.EpicoGarantidoEm != dominio.PityEpico {
		t.Errorf("pity depois do épico errado: %+v", res.Pity)
	}
}

func TestColecao_OrdenaPorPoderEUsaTierAtual(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo"})
	repo.colecao[1] = []dominio.ItemColecao{{PokemonID: 6, Tier: "comum"}, {PokemonID: 727, Tier: "lendario"}}
	svc := Novo(repo, &catalogoFalso{}, rolagemFixa(0), regras)

	c, err := svc.Colecao(context.Background(), 1)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if c.Pokemons[0].PokemonID != 727 || c.Pokemons[0].Tier != "raro" || c.Pokemons[0].Detalhes == nil {
		t.Fatalf("esperava o de maior poder primeiro, com tier atualizado: %+v", c.Pokemons[0])
	}
}

func TestColecao_CatalogoForaDevolveSemDetalhes(t *testing.T) {
	repo := novoRepo(dominio.Jogador{ID: 1, Nome: "theo"})
	repo.colecao[1] = []dominio.ItemColecao{{PokemonID: 6}}
	cat := &catalogoFalso{erro: fmt.Errorf("%w: catálogo", dominio.ErrServicoIndisponivel)}
	svc := Novo(repo, cat, rolagemFixa(0), regras)

	c, err := svc.Colecao(context.Background(), 1)
	if err != nil {
		t.Fatalf("catálogo fora não deveria derrubar a coleção: %v", err)
	}
	if !c.DetalhesIndisponiveis || len(c.Pokemons) != 1 {
		t.Fatalf("esperava a coleção sem detalhes: %+v", c)
	}
}
