// orquestra a batalha: valida o time, busca dados no catálogo,
// roda o motor e paga a recompensa.
package servico

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"pokebatalha/services/batalha/internal/dominio"
)

// catálogo (implementado em clientes.Catalogo).
type Catalogo interface {
	BuscarPokemon(ctx context.Context, id int) (dominio.Especie, error)
	SortearPokemon(ctx context.Context, tier string, poder int) (dominio.Especie, error)
	TabelaDeTipos(ctx context.Context) (dominio.Tabela, error)
}

// jogadores (implementado em clientes.Jogadores).
type Jogadores interface {
	ValidarTime(ctx context.Context, jogadorID int64, pokemonIDs []int) error
	CreditarMoedas(ctx context.Context, jogadorID int64, quantidade int) (int, error)
}

// banco de batalhas (implementado em repositorio.Postgres).
type Repositorio interface {
	Criar(ctx context.Context, b *dominio.Batalha) error
	Buscar(ctx context.Context, id int64) (*dominio.Batalha, error)
	Atualizar(ctx context.Context, b *dominio.Batalha) error
}

type Servico struct {
	repo       Repositorio
	catalogo   Catalogo
	jogadores  Jogadores
	sorteio    dominio.Sorteador
	recompensa int

	// a tabela de tipos não muda: busca uma vez e guarda em memória.
	mu     sync.Mutex
	tabela dominio.Tabela
}

func Novo(repo Repositorio, cat Catalogo, jog Jogadores, s dominio.Sorteador, recompensa int) *Servico {
	return &Servico{repo: repo, catalogo: cat, jogadores: jog, sorteio: s, recompensa: recompensa}
}

func (s *Servico) tabelaDeTipos(ctx context.Context) (dominio.Tabela, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tabela != nil {
		return s.tabela, nil
	}
	t, err := s.catalogo.TabelaDeTipos(ctx)
	if err != nil {
		return nil, err
	}
	s.tabela = t
	return t, nil
}

// monta o time do jogador e um rival equilibrado: mesmo tier e poder parecido.
func (s *Servico) CriarBatalha(ctx context.Context, jogadorID int64, pokemonIDs []int) (*dominio.Batalha, error) {
	if jogadorID < 1 {
		return nil, fmt.Errorf("%w: jogador_id é obrigatório", dominio.ErrTimeInvalido)
	}
	if len(pokemonIDs) == 0 || len(pokemonIDs) > dominio.TamanhoMaximoTime {
		return nil, fmt.Errorf("%w: escolha de 1 a %d pokémon", dominio.ErrTimeInvalido, dominio.TamanhoMaximoTime)
	}
	if temRepetido(pokemonIDs) {
		return nil, fmt.Errorf("%w: não repita pokémon no time", dominio.ErrTimeInvalido)
	}

	// 1. jogadores confirma que o time é dele (gRPC).
	if err := s.jogadores.ValidarTime(ctx, jogadorID, pokemonIDs); err != nil {
		return nil, err
	}
	// 2. garante a tabela de tipos.
	if _, err := s.tabelaDeTipos(ctx); err != nil {
		return nil, err
	}

	// 3. dados do time e sorteio do rival, tier por tier (gRPC).
	timeJogador := make([]dominio.Lutador, 0, len(pokemonIDs))
	timeCPU := make([]dominio.Lutador, 0, len(pokemonIDs))
	for _, id := range pokemonIDs {
		e, err := s.catalogo.BuscarPokemon(ctx, id)
		if err != nil {
			return nil, err
		}
		timeJogador = append(timeJogador, dominio.NovoLutador(e))

		rival, err := s.catalogo.SortearPokemon(ctx, e.Tier, e.Poder)
		if err != nil {
			return nil, err
		}
		timeCPU = append(timeCPU, dominio.NovoLutador(rival))
	}

	b, err := dominio.NovaBatalha(jogadorID, timeJogador, timeCPU, s.sorteio)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Criar(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// estado atual da batalha.
func (s *Servico) Buscar(ctx context.Context, id int64) (*dominio.Batalha, error) {
	return s.repo.Buscar(ctx, id)
}

// executa um turno (alvo só vale para "trocar").
func (s *Servico) Jogar(ctx context.Context, id int64, acaoTexto string, alvo int) (*dominio.Batalha, error) {
	acao, err := dominio.ParseAcao(acaoTexto)
	if err != nil {
		return nil, err
	}
	b, err := s.repo.Buscar(ctx, id)
	if err != nil {
		return nil, err
	}
	tabela, err := s.tabelaDeTipos(ctx)
	if err != nil {
		return nil, err
	}
	if err := b.ExecutarTurno(dominio.Escolha{Acao: acao, Alvo: alvo}, tabela, s.sorteio); err != nil {
		return nil, err
	}

	// paga antes de salvar: se jogadores cair, o turno não é gravado
	// e o jogador tenta de novo sem perder a recompensa.
	if b.Status == dominio.Vitoria {
		if _, err := s.jogadores.CreditarMoedas(ctx, b.JogadorID, s.recompensa); err != nil {
			return nil, err
		}
		b.Recompensa = s.recompensa
	}

	if err := s.repo.Atualizar(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func temRepetido(ids []int) bool {
	ordenados := slices.Clone(ids)
	slices.Sort(ordenados)
	return len(slices.Compact(ordenados)) != len(ids)
}
