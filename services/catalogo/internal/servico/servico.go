// Package servico contém a lógica de negócio do catálogo: cache sobre a
// PokeAPI, índice de poder de cada Pokémon e sorteio por tier.
package servico

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"pokebatalha/services/catalogo/internal/dominio"
)

// Repositorio é o cache local (implementado em repositorio.Postgres).
type Repositorio interface {
	BuscarPokemonPorID(ctx context.Context, id int) (dominio.Pokemon, error)
	BuscarPokemonPorNome(ctx context.Context, nome string) (dominio.Pokemon, error)
	ListarPokemons(ctx context.Context) ([]dominio.Pokemon, error)
	SalvarPokemon(ctx context.Context, p dominio.Pokemon) error
	BuscarTipo(ctx context.Context, nome string) (dominio.RelacaoTipo, error)
	SalvarTipo(ctx context.Context, t dominio.RelacaoTipo) error
}

// FonteExterna é de onde vêm os dados originais (implementada em pokeapi.Cliente).
type FonteExterna interface {
	BuscarPokemon(ctx context.Context, idOuNome string) (dominio.Pokemon, error)
	BuscarTipo(ctx context.Context, nome string) (dominio.RelacaoTipo, error)
}

// Sorteador permite trocar a aleatoriedade por algo previsível nos testes.
type Sorteador interface {
	IntN(n int) int
}

const (
	// Com o cache frio, o sorteio testa IDs aleatórios até achar o tier.
	tentativasAmostragem = 40
	// O rival é escolhido entre os candidatos mais próximos do poder pedido.
	janelaPoder        = 20
	candidatosMinimos  = 5
	maximoBuscaEmLote  = 50
	trabalhadoresAquec = 4
)

// Servico é o caso de uso do catálogo.
type Servico struct {
	repo    Repositorio
	fonte   FonteExterna
	sorteio Sorteador
	log     *slog.Logger

	// A tabela de tipos não muda: carregamos uma vez e guardamos em memória.
	mu     sync.Mutex
	tabela dominio.Tabela
}

// Novo monta o serviço com suas dependências.
func Novo(repo Repositorio, fonte FonteExterna, s Sorteador, log *slog.Logger) *Servico {
	return &Servico{repo: repo, fonte: fonte, sorteio: s, log: log}
}

// BuscarPorID devolve o Pokémon pelo número da PokeAPI.
func (s *Servico) BuscarPorID(ctx context.Context, id int) (dominio.Pokemon, error) {
	if id < 1 {
		return dominio.Pokemon{}, fmt.Errorf("%w: %d", dominio.ErrIDInvalido, id)
	}
	return s.buscar(ctx, strconv.Itoa(id), func() (dominio.Pokemon, error) {
		return s.repo.BuscarPokemonPorID(ctx, id)
	})
}

// BuscarPorNome devolve o Pokémon pelo nome usado na PokeAPI (ex: "incineroar").
func (s *Servico) BuscarPorNome(ctx context.Context, nome string) (dominio.Pokemon, error) {
	nome = strings.ToLower(strings.TrimSpace(nome))
	if nome == "" {
		return dominio.Pokemon{}, fmt.Errorf("%w: nome vazio", dominio.ErrPokemonNaoEncontrado)
	}
	return s.buscar(ctx, nome, func() (dominio.Pokemon, error) {
		return s.repo.BuscarPokemonPorNome(ctx, nome)
	})
}

// BuscarVarios devolve vários Pokémon de uma vez (usado para a coleção).
func (s *Servico) BuscarVarios(ctx context.Context, ids []int) ([]dominio.Pokemon, error) {
	if len(ids) > maximoBuscaEmLote {
		return nil, fmt.Errorf("%w: no máximo %d por vez", dominio.ErrIDInvalido, maximoBuscaEmLote)
	}
	resultado := make([]dominio.Pokemon, 0, len(ids))
	for _, id := range ids {
		p, err := s.BuscarPorID(ctx, id)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, p)
	}
	return resultado, nil
}

// buscar tenta o cache e, se não achar, vai na PokeAPI e guarda o resultado.
func (s *Servico) buscar(ctx context.Context, chave string, doCache func() (dominio.Pokemon, error)) (dominio.Pokemon, error) {
	tabela, err := s.tabelaDeTipos(ctx)
	if err != nil {
		return dominio.Pokemon{}, err
	}

	p, err := doCache()
	if err == nil {
		return comPoder(p, tabela), nil
	}
	if !errors.Is(err, dominio.ErrForaDoCache) {
		return dominio.Pokemon{}, fmt.Errorf("consultar cache: %w", err)
	}

	p, err = s.fonte.BuscarPokemon(ctx, chave)
	if err != nil {
		return dominio.Pokemon{}, err // já vem classificado: negócio ou infraestrutura
	}

	// Falhar ao gravar o cache não impede de responder: só registramos.
	if err := s.repo.SalvarPokemon(ctx, p); err != nil {
		s.log.Warn("não foi possível salvar pokémon no cache", "pokemon", p.Nome, "erro", err)
	}
	return comPoder(p, tabela), nil
}

func comPoder(p dominio.Pokemon, t dominio.Tabela) dominio.Pokemon {
	p.Poder = dominio.CalcularPoder(p, t)
	p.Tier = dominio.TierPorPoder(p.Poder.Total)
	return p
}

// Sortear escolhe um Pokémon aleatório do tier. Se poderReferencia > 0, sorteia
// entre os de poder mais próximo (é assim que o rival fica equilibrado).
func (s *Servico) Sortear(ctx context.Context, tier dominio.Tier, poderReferencia int) (dominio.Pokemon, error) {
	candidatos, err := s.candidatosDoCache(ctx, tier)
	if err != nil {
		return dominio.Pokemon{}, err
	}

	// Cache frio (o aquecimento ainda não terminou): testa IDs aleatórios.
	precisa := 1
	if poderReferencia > 0 {
		precisa = candidatosMinimos
	}
	for i := 0; len(candidatos) < precisa && i < tentativasAmostragem; i++ {
		p, err := s.BuscarPorID(ctx, 1+s.sorteio.IntN(dominio.MaiorIDNacional))
		if err != nil {
			return dominio.Pokemon{}, err
		}
		if p.Tier == tier && !slices.ContainsFunc(candidatos, func(c dominio.Pokemon) bool { return c.ID == p.ID }) {
			candidatos = append(candidatos, p)
		}
	}
	if len(candidatos) == 0 {
		return dominio.Pokemon{}, fmt.Errorf("%w: nenhum pokémon %s no cache ainda", dominio.ErrCatalogoAquecendo, tier)
	}

	if poderReferencia > 0 {
		candidatos = maisProximos(candidatos, poderReferencia)
	}
	return candidatos[s.sorteio.IntN(len(candidatos))], nil
}

func (s *Servico) candidatosDoCache(ctx context.Context, tier dominio.Tier) ([]dominio.Pokemon, error) {
	tabela, err := s.tabelaDeTipos(ctx)
	if err != nil {
		return nil, err
	}
	todos, err := s.repo.ListarPokemons(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar cache: %w", err)
	}
	var candidatos []dominio.Pokemon
	for _, p := range todos {
		if p.ID > dominio.MaiorIDNacional {
			continue // formas alternativas ficam fora do gacha
		}
		if p = comPoder(p, tabela); p.Tier == tier {
			candidatos = append(candidatos, p)
		}
	}
	return candidatos, nil
}

// maisProximos fica com quem está dentro da janela de poder, garantindo
// pelo menos alguns candidatos para o sorteio não ficar previsível.
func maisProximos(candidatos []dominio.Pokemon, alvo int) []dominio.Pokemon {
	distancia := func(p dominio.Pokemon) int { return abs(p.Poder.Total - alvo) }
	slices.SortStableFunc(candidatos, func(a, b dominio.Pokemon) int { return distancia(a) - distancia(b) })

	n := 0
	for n < len(candidatos) && (n < candidatosMinimos || distancia(candidatos[n]) <= janelaPoder) {
		n++
	}
	return candidatos[:n]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TabelaDeTipos devolve as relações de dano dos 18 tipos.
func (s *Servico) TabelaDeTipos(ctx context.Context) ([]dominio.RelacaoTipo, error) {
	tabela, err := s.tabelaDeTipos(ctx)
	if err != nil {
		return nil, err
	}
	lista := make([]dominio.RelacaoTipo, 0, len(dominio.TiposExistentes))
	for _, nome := range dominio.TiposExistentes {
		lista = append(lista, tabela[nome])
	}
	return lista, nil
}

func (s *Servico) tabelaDeTipos(ctx context.Context) (dominio.Tabela, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tabela != nil {
		return s.tabela, nil
	}
	tabela := make(dominio.Tabela, len(dominio.TiposExistentes))
	for _, nome := range dominio.TiposExistentes {
		t, err := s.relacaoTipo(ctx, nome)
		if err != nil {
			return nil, err
		}
		tabela[nome] = t
	}
	s.tabela = tabela
	return tabela, nil
}

func (s *Servico) relacaoTipo(ctx context.Context, nome string) (dominio.RelacaoTipo, error) {
	t, err := s.repo.BuscarTipo(ctx, nome)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, dominio.ErrForaDoCache) {
		return dominio.RelacaoTipo{}, fmt.Errorf("consultar cache de tipos: %w", err)
	}
	t, err = s.fonte.BuscarTipo(ctx, nome)
	if err != nil {
		return dominio.RelacaoTipo{}, err
	}
	if err := s.repo.SalvarTipo(ctx, t); err != nil {
		s.log.Warn("não foi possível salvar tipo no cache", "tipo", nome, "erro", err)
	}
	return t, nil
}

// Efetividade calcula o multiplicador de um ataque contra um defensor
// com um ou dois tipos.
func (s *Servico) Efetividade(ctx context.Context, ataque string, defensor []string) (float64, error) {
	if err := dominio.ValidarTipo(ataque); err != nil {
		return 0, err
	}
	if len(defensor) == 0 || len(defensor) > 2 {
		return 0, fmt.Errorf("%w: o defensor precisa ter 1 ou 2 tipos", dominio.ErrTipoInvalido)
	}
	for _, d := range defensor {
		if err := dominio.ValidarTipo(d); err != nil {
			return 0, err
		}
	}
	tabela, err := s.tabelaDeTipos(ctx)
	if err != nil {
		return 0, err
	}
	return tabela[ataque].Multiplicador(defensor), nil
}

// Regras devolve a fórmula do índice de poder e os cortes de tier.
func (s *Servico) Regras() dominio.Regras {
	return dominio.RegrasDoPoder()
}

// Aquecer baixa para o cache toda a Pokédex que ainda não está nele, para o
// sorteio por tier ficar rápido. Roda em segundo plano ao subir o serviço;
// depois disso a PokeAPI só é consultada de novo se o banco for apagado.
func (s *Servico) Aquecer(ctx context.Context) {
	if _, err := s.tabelaDeTipos(ctx); err != nil {
		s.log.Warn("aquecimento adiado: tabela de tipos indisponível", "erro", err)
		return
	}
	todos, err := s.repo.ListarPokemons(ctx)
	if err != nil {
		s.log.Warn("aquecimento cancelado: cache indisponível", "erro", err)
		return
	}
	jaTem := make(map[int]bool, len(todos))
	for _, p := range todos {
		jaTem[p.ID] = true
	}
	var faltando []int
	for id := 1; id <= dominio.MaiorIDNacional; id++ {
		if !jaTem[id] {
			faltando = append(faltando, id)
		}
	}
	if len(faltando) == 0 {
		s.log.Info("cache da pokédex completo", "pokemons", len(todos))
		return
	}
	s.log.Info("aquecendo cache da pokédex", "faltando", len(faltando))

	ids := make(chan int)
	var (
		wg     sync.WaitGroup
		feitos atomic.Int64
		falhas atomic.Int64
	)
	for range trabalhadoresAquec {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range ids {
				if _, err := s.BuscarPorID(ctx, id); err != nil {
					falhas.Add(1)
					s.log.Warn("falha ao aquecer pokémon", "id", id, "erro", err)
					continue
				}
				if n := feitos.Add(1); n%100 == 0 {
					s.log.Info("aquecimento em andamento", "baixados", n, "de", len(faltando))
				}
			}
		}()
	}
enviar:
	for _, id := range faltando {
		select {
		case ids <- id:
		case <-ctx.Done():
			break enviar
		}
	}
	close(ids)
	wg.Wait()
	s.log.Info("aquecimento concluído", "baixados", feitos.Load(), "falhas", falhas.Load())
}
