package servico

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"testing"

	"pokebatalha/services/catalogo/internal/dominio"
)

// repoMemoria é um cache em memória para os testes.
type repoMemoria struct {
	porID  map[int]dominio.Pokemon
	tipos  map[string]dominio.RelacaoTipo
	falhar error // se definido, a leitura de pokémon falha com ele
}

func novoRepo() *repoMemoria {
	return &repoMemoria{porID: map[int]dominio.Pokemon{}, tipos: map[string]dominio.RelacaoTipo{}}
}

func (r *repoMemoria) BuscarPokemonPorID(_ context.Context, id int) (dominio.Pokemon, error) {
	if r.falhar != nil {
		return dominio.Pokemon{}, r.falhar
	}
	p, ok := r.porID[id]
	if !ok {
		return dominio.Pokemon{}, dominio.ErrForaDoCache
	}
	return p, nil
}

func (r *repoMemoria) BuscarPokemonPorNome(_ context.Context, nome string) (dominio.Pokemon, error) {
	for _, p := range r.porID {
		if p.Nome == nome {
			return p, nil
		}
	}
	return dominio.Pokemon{}, dominio.ErrForaDoCache
}

func (r *repoMemoria) ListarPokemons(context.Context) ([]dominio.Pokemon, error) {
	var todos []dominio.Pokemon
	for _, p := range r.porID {
		todos = append(todos, p)
	}
	return todos, nil
}

func (r *repoMemoria) SalvarPokemon(_ context.Context, p dominio.Pokemon) error {
	r.porID[p.ID] = p
	return nil
}

func (r *repoMemoria) BuscarTipo(_ context.Context, nome string) (dominio.RelacaoTipo, error) {
	t, ok := r.tipos[nome]
	if !ok {
		return dominio.RelacaoTipo{}, dominio.ErrForaDoCache
	}
	return t, nil
}

func (r *repoMemoria) SalvarTipo(_ context.Context, t dominio.RelacaoTipo) error {
	r.tipos[t.Tipo] = t
	return nil
}

// fonteFalsa simula a PokeAPI e conta quantas vezes foi chamada.
type fonteFalsa struct {
	pokemons map[string]dominio.Pokemon // chave: id em texto
	erro     error
	chamadas int
}

func (f *fonteFalsa) BuscarPokemon(_ context.Context, chave string) (dominio.Pokemon, error) {
	f.chamadas++
	if f.erro != nil {
		return dominio.Pokemon{}, f.erro
	}
	p, ok := f.pokemons[chave]
	if !ok {
		return dominio.Pokemon{}, fmt.Errorf("%w: %s", dominio.ErrPokemonNaoEncontrado, chave)
	}
	return p, nil
}

func (f *fonteFalsa) BuscarTipo(_ context.Context, nome string) (dominio.RelacaoTipo, error) {
	if f.erro != nil {
		return dominio.RelacaoTipo{}, f.erro
	}
	// Tabela neutra: todo ataque é 1x, então o poder vira stats + habilidade.
	return dominio.RelacaoTipo{Tipo: nome}, nil
}

// sorteadorFixo devolve os valores na ordem dada (e 0 quando acabam).
type sorteadorFixo struct{ valores []int }

func (s *sorteadorFixo) IntN(n int) int {
	if len(s.valores) == 0 {
		return 0
	}
	v := s.valores[0]
	s.valores = s.valores[1:]
	return v % n
}

// especie cria um Pokémon cujos stats somam exatamente "stats" no índice de poder.
func especie(id int, nome string, stats int) dominio.Pokemon {
	q := stats / 4
	return dominio.Pokemon{ID: id, Nome: nome, Tipos: []string{"normal"},
		Status: dominio.Status{HP: q, Ataque: q, Defesa: q, DefesaEspecial: q, Velocidade: stats - 3*q}}
}

func novoServico(t *testing.T, repo Repositorio, fonte FonteExterna, s Sorteador) *Servico {
	t.Helper()
	return Novo(repo, fonte, s, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func fonteCom(pokemons ...dominio.Pokemon) *fonteFalsa {
	f := &fonteFalsa{pokemons: map[string]dominio.Pokemon{}}
	for _, p := range pokemons {
		f.pokemons[strconv.Itoa(p.ID)] = p
	}
	return f
}

func TestBuscarPorID_UsaCacheDepoisDaPrimeiraVez(t *testing.T) {
	fonte := fonteCom(especie(6, "charizard", 368))
	svc := novoServico(t, novoRepo(), fonte, nil)

	for range 3 {
		if _, err := svc.BuscarPorID(context.Background(), 6); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	if fonte.chamadas != 1 {
		t.Fatalf("PokeAPI deveria ser chamada 1 vez, foi chamada %d", fonte.chamadas)
	}
}

func TestBuscar_CalculaPoderETier(t *testing.T) {
	forte := especie(1, "forte", 440)
	fraco := especie(2, "fraco", 200)
	svc := novoServico(t, novoRepo(), fonteCom(forte, fraco), nil)

	f, _ := svc.BuscarPorID(context.Background(), 1)
	c, _ := svc.BuscarPorID(context.Background(), 2)
	if f.Poder.Total != 440 || f.Tier != dominio.TierLendario {
		t.Errorf("esperava poder 440 lendário, veio %d %s", f.Poder.Total, f.Tier)
	}
	if c.Tier != dominio.TierComum {
		t.Errorf("esperava comum, veio %s", c.Tier)
	}
}

func TestBuscarPorID_Invalido(t *testing.T) {
	svc := novoServico(t, novoRepo(), &fonteFalsa{}, nil)
	if _, err := svc.BuscarPorID(context.Background(), 0); !errors.Is(err, dominio.ErrIDInvalido) {
		t.Fatalf("esperava ErrIDInvalido, veio %v", err)
	}
}

func TestBuscarPorID_PokeAPIForaDoAr(t *testing.T) {
	fonte := &fonteFalsa{erro: fmt.Errorf("%w: timeout", dominio.ErrServicoExternoIndisponivel)}
	svc := novoServico(t, novoRepo(), fonte, nil)

	_, err := svc.BuscarPorID(context.Background(), 25)
	if !errors.Is(err, dominio.ErrServicoExternoIndisponivel) {
		t.Fatalf("esperava erro de infraestrutura, veio %v", err)
	}
	if errors.Is(err, dominio.ErrPokemonNaoEncontrado) {
		t.Fatal("PokeAPI fora do ar não pode virar 'não encontrado'")
	}
}

func TestBuscarPorID_BancoForaDoArNaoViraNaoEncontrado(t *testing.T) {
	repo := novoRepo()
	repo.falhar = errors.New("conexão recusada")
	svc := novoServico(t, repo, fonteCom(), nil)

	_, err := svc.BuscarPorID(context.Background(), 25)
	if err == nil || errors.Is(err, dominio.ErrPokemonNaoEncontrado) {
		t.Fatalf("esperava erro interno, veio %v", err)
	}
}

func TestSortear_RivalDePoderProximo(t *testing.T) {
	repo := novoRepo()
	// Lendários de 430 a 530, de 10 em 10.
	for i := range 11 {
		p := especie(100+i, fmt.Sprintf("lendario-%d", i), 430+10*i)
		repo.porID[p.ID] = p
	}
	svc := novoServico(t, repo, fonteCom(), &sorteadorFixo{})

	for range 20 {
		p, err := svc.Sortear(context.Background(), dominio.TierLendario, 440)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		// Os 5 mais próximos de 440 vão de 430 a 470.
		if p.Poder.Total < 430 || p.Poder.Total > 470 {
			t.Fatalf("rival com poder %d está longe demais de 440", p.Poder.Total)
		}
	}
}

func TestSortear_CacheFrioAmostraIDsAleatorios(t *testing.T) {
	alvo := especie(500, "achado", 450)
	fonte := fonteCom(especie(7, "fraco", 200), alvo)
	// Primeiro sorteio cai no 7 (comum), o segundo no 500 (lendário).
	svc := novoServico(t, novoRepo(), fonte, &sorteadorFixo{valores: []int{6, 499}})

	p, err := svc.Sortear(context.Background(), dominio.TierLendario, 0)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if p.Nome != "achado" {
		t.Fatalf("esperava o lendário 'achado', veio %s", p.Nome)
	}
}

func TestSortear_SemNenhumCandidato(t *testing.T) {
	// Toda a Pokédex é fraca: não existe lendário para sortear.
	fonte := fonteCom()
	for id := 1; id <= dominio.MaiorIDNacional; id++ {
		fonte.pokemons[strconv.Itoa(id)] = especie(id, fmt.Sprintf("p%d", id), 100)
	}
	svc := novoServico(t, novoRepo(), fonte, &sorteadorFixo{})

	_, err := svc.Sortear(context.Background(), dominio.TierLendario, 0)
	if !errors.Is(err, dominio.ErrCatalogoAquecendo) {
		t.Fatalf("esperava ErrCatalogoAquecendo, veio %v", err)
	}
}

func TestEfetividade_ValidaTipos(t *testing.T) {
	svc := novoServico(t, novoRepo(), &fonteFalsa{}, nil)

	if _, err := svc.Efetividade(context.Background(), "fogo", []string{"grass"}); !errors.Is(err, dominio.ErrTipoInvalido) {
		t.Errorf("tipo em português deveria ser inválido, veio %v", err)
	}
	if _, err := svc.Efetividade(context.Background(), "fire", nil); !errors.Is(err, dominio.ErrTipoInvalido) {
		t.Errorf("defensor vazio deveria ser inválido, veio %v", err)
	}
}
