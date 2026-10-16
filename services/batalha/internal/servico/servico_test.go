package servico

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"pokebatalha/services/batalha/internal/dominio"
)

// dano máximo e IA sem decisões aleatórias: turno previsível.
type sorteadorFixo struct{}

func (sorteadorFixo) IntN(n int) int   { return n - 1 }
func (sorteadorFixo) Float64() float64 { return 1 }

// time do jogador forte, rival fraquíssimo.
type catalogoFalso struct {
	tiersSorteados []string
	erro           error
}

func (c *catalogoFalso) BuscarPokemon(_ context.Context, id int) (dominio.Especie, error) {
	if c.erro != nil {
		return dominio.Especie{}, c.erro
	}
	return dominio.Especie{PokemonID: id, Nome: fmt.Sprintf("forte-%d", id), Tier: "lendario",
		Tipos: []string{"normal"}, HP: 200, Ataque: 200, Defesa: 200, Velocidade: 200}, nil
}

func (c *catalogoFalso) SortearPokemon(_ context.Context, tier string, _ int) (dominio.Especie, error) {
	c.tiersSorteados = append(c.tiersSorteados, tier)
	return dominio.Especie{PokemonID: 129, Nome: "magikarp", Tier: tier,
		Tipos: []string{"water"}, HP: 1, Ataque: 1, Defesa: 1, Velocidade: 1}, nil
}

func (c *catalogoFalso) TabelaDeTipos(context.Context) (dominio.Tabela, error) {
	if c.erro != nil {
		return nil, c.erro
	}
	return dominio.NovaTabela(nil), nil
}

type jogadoresFalso struct {
	erroValidar  error
	erroCreditar error
	creditado    int
}

func (j *jogadoresFalso) ValidarTime(context.Context, int64, []int) error { return j.erroValidar }

func (j *jogadoresFalso) CreditarMoedas(_ context.Context, _ int64, qtd int) (int, error) {
	if j.erroCreditar != nil {
		return 0, j.erroCreditar
	}
	j.creditado += qtd
	return j.creditado, nil
}

type repoMemoria struct {
	batalhas     map[int64]dominio.Batalha
	atualizacoes int
}

func novoRepo() *repoMemoria { return &repoMemoria{batalhas: map[int64]dominio.Batalha{}} }

func (r *repoMemoria) Criar(_ context.Context, b *dominio.Batalha) error {
	b.ID = int64(len(r.batalhas) + 1)
	r.batalhas[b.ID] = *b
	return nil
}

func (r *repoMemoria) Buscar(_ context.Context, id int64) (*dominio.Batalha, error) {
	b, ok := r.batalhas[id]
	if !ok {
		return nil, dominio.ErrBatalhaNaoEncontrada
	}
	return &b, nil
}

func (r *repoMemoria) Atualizar(_ context.Context, b *dominio.Batalha) error {
	r.atualizacoes++
	r.batalhas[b.ID] = *b
	return nil
}

func TestCriarBatalha_RivalTemOsMesmosTiers(t *testing.T) {
	cat := &catalogoFalso{}
	svc := Novo(novoRepo(), cat, &jogadoresFalso{}, sorteadorFixo{}, 50)

	b, err := svc.CriarBatalha(context.Background(), 1, []int{727, 987})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(b.CPU.Time) != 2 {
		t.Fatalf("o rival deveria ter 2 pokémon, tem %d", len(b.CPU.Time))
	}
	if cat.tiersSorteados[0] != "lendario" || cat.tiersSorteados[1] != "lendario" {
		t.Errorf("o rival deveria ser sorteado nos tiers do time do jogador: %v", cat.tiersSorteados)
	}
}

func TestCriarBatalha_ValidacoesDeNegocio(t *testing.T) {
	svc := Novo(novoRepo(), &catalogoFalso{}, &jogadoresFalso{}, sorteadorFixo{}, 50)
	ctx := context.Background()

	casos := map[string][]int{
		"time vazio":       {},
		"time grande":      {1, 2, 3, 4},
		"pokémon repetido": {727, 727},
	}
	for nome, time := range casos {
		if _, err := svc.CriarBatalha(ctx, 1, time); !errors.Is(err, dominio.ErrTimeInvalido) {
			t.Errorf("%s: esperava ErrTimeInvalido, veio %v", nome, err)
		}
	}
}

func TestCriarBatalha_PokemonDeOutroJogador(t *testing.T) {
	jog := &jogadoresFalso{erroValidar: fmt.Errorf("%w: ids [150]", dominio.ErrPokemonNaoPertence)}
	svc := Novo(novoRepo(), &catalogoFalso{}, jog, sorteadorFixo{}, 50)

	_, err := svc.CriarBatalha(context.Background(), 1, []int{150})
	if !errors.Is(err, dominio.ErrPokemonNaoPertence) {
		t.Fatalf("esperava ErrPokemonNaoPertence, veio %v", err)
	}
}

func TestCriarBatalha_CatalogoForaDoAr(t *testing.T) {
	cat := &catalogoFalso{erro: fmt.Errorf("%w: catálogo", dominio.ErrServicoIndisponivel)}
	svc := Novo(novoRepo(), cat, &jogadoresFalso{}, sorteadorFixo{}, 50)

	_, err := svc.CriarBatalha(context.Background(), 1, []int{727})
	if !errors.Is(err, dominio.ErrServicoIndisponivel) {
		t.Fatalf("esperava erro de infraestrutura, veio %v", err)
	}
}

func TestJogar_VitoriaPagaRecompensa(t *testing.T) {
	repo, jog := novoRepo(), &jogadoresFalso{}
	svc := Novo(repo, &catalogoFalso{}, jog, sorteadorFixo{}, 50)
	ctx := context.Background()

	b, _ := svc.CriarBatalha(ctx, 1, []int{727})
	b, err := svc.Jogar(ctx, b.ID, "fisico", 0)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if b.Status != dominio.Vitoria || b.Recompensa != 50 || jog.creditado != 50 {
		t.Fatalf("esperava vitória com 50 moedas (status=%s recompensa=%d creditado=%d)",
			b.Status, b.Recompensa, jog.creditado)
	}
}

func TestJogar_SeNaoConseguePagarNaoSalvaOTurno(t *testing.T) {
	repo := novoRepo()
	jog := &jogadoresFalso{}
	svc := Novo(repo, &catalogoFalso{}, jog, sorteadorFixo{}, 50)
	ctx := context.Background()
	b, _ := svc.CriarBatalha(ctx, 1, []int{727})

	jog.erroCreditar = fmt.Errorf("%w: jogadores", dominio.ErrServicoIndisponivel)
	if _, err := svc.Jogar(ctx, b.ID, "fisico", 0); !errors.Is(err, dominio.ErrServicoIndisponivel) {
		t.Fatalf("esperava ErrServicoIndisponivel, veio %v", err)
	}
	if repo.atualizacoes != 0 {
		t.Fatal("o turno não deveria ser salvo sem pagar a recompensa")
	}
}

func TestJogar_AcaoInvalidaEBatalhaInexistente(t *testing.T) {
	svc := Novo(novoRepo(), &catalogoFalso{}, &jogadoresFalso{}, sorteadorFixo{}, 50)
	ctx := context.Background()

	if _, err := svc.Jogar(ctx, 1, "hyper-beam", 0); !errors.Is(err, dominio.ErrAcaoInvalida) {
		t.Errorf("esperava ErrAcaoInvalida, veio %v", err)
	}
	if _, err := svc.Jogar(ctx, 99, "fisico", 0); !errors.Is(err, dominio.ErrBatalhaNaoEncontrada) {
		t.Errorf("esperava ErrBatalhaNaoEncontrada, veio %v", err)
	}
}
