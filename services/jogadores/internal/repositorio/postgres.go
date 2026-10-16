// acesso ao banco de jogadores.
package repositorio

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"pokebatalha/services/jogadores/internal/dominio"
)

//go:embed schema.sql
var schema string

// código do Postgres para chave duplicada.
const codigoViolacaoUnique = "23505"

// jogadores e coleções.
type Postgres struct {
	pool *pgxpool.Pool
}

func NovoPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// cria as tabelas se não existirem.
func (r *Postgres) Migrar(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("criar tabelas de jogadores: %w", err)
	}
	return nil
}

// insere um jogador novo.
func (r *Postgres) Criar(ctx context.Context, nome string, moedas int) (dominio.Jogador, error) {
	j := dominio.Jogador{Nome: nome, Moedas: moedas}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO jogadores (nome, moedas) VALUES ($1, $2) RETURNING id, criado_em`,
		nome, moedas,
	).Scan(&j.ID, &j.CriadoEm)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == codigoViolacaoUnique {
		return dominio.Jogador{}, fmt.Errorf("%w: %q", dominio.ErrNomeEmUso, nome)
	}
	if err != nil {
		return dominio.Jogador{}, fmt.Errorf("inserir jogador: %w", err)
	}
	return j, nil
}

// devolve ErrJogadorNaoEncontrado se o ID não existir.
func (r *Postgres) BuscarPorID(ctx context.Context, id int64) (dominio.Jogador, error) {
	var j dominio.Jogador
	err := r.pool.QueryRow(ctx, `
		SELECT id, nome, moedas, pacotes_sem_epico, pacotes_sem_lendario, criado_em
		FROM jogadores WHERE id = $1`, id,
	).Scan(&j.ID, &j.Nome, &j.Moedas, &j.Pity.SemEpico, &j.Pity.SemLendario, &j.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return dominio.Jogador{}, fmt.Errorf("%w: id %d", dominio.ErrJogadorNaoEncontrado, id)
	}
	if err != nil {
		return dominio.Jogador{}, fmt.Errorf("buscar jogador: %w", err)
	}
	return j, nil
}

// lista os Pokémon do jogador, do mais recente ao mais antigo.
func (r *Postgres) ListarColecao(ctx context.Context, jogadorID int64) ([]dominio.ItemColecao, error) {
	linhas, err := r.pool.Query(ctx, `
		SELECT id, pokemon_id, nome, tier, tipos, adquirido_em
		FROM colecao WHERE jogador_id = $1 ORDER BY adquirido_em DESC, id DESC`, jogadorID)
	if err != nil {
		return nil, fmt.Errorf("listar coleção: %w", err)
	}
	defer linhas.Close()

	itens := []dominio.ItemColecao{}
	for linhas.Next() {
		var it dominio.ItemColecao
		if err := linhas.Scan(&it.ID, &it.PokemonID, &it.Nome, &it.Tier, &it.Tipos, &it.AdquiridoEm); err != nil {
			return nil, fmt.Errorf("ler item da coleção: %w", err)
		}
		itens = append(itens, it)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("percorrer coleção: %w", err)
	}
	return itens, nil
}

// debita, atualiza o pity e entrega na mesma transação (nunca cobra sem entregar).
// o UPDATE só passa se o pity não mudou desde o sorteio:
// dois pacotes simultâneos não usam a mesma garantia.
func (r *Postgres) RegistrarCompra(ctx context.Context, jogadorID int64, custo int, pityLido, pityNovo dominio.Pity, p dominio.PokemonSorteado) (dominio.ItemColecao, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return dominio.ItemColecao{}, 0, fmt.Errorf("abrir transação: %w", err)
	}
	defer tx.Rollback(ctx) // sem efeito se o Commit já tiver acontecido

	var saldo int
	err = tx.QueryRow(ctx, `
		UPDATE jogadores
		SET moedas = moedas - $1, pacotes_sem_epico = $2, pacotes_sem_lendario = $3
		WHERE id = $4 AND moedas >= $1 AND pacotes_sem_epico = $5 AND pacotes_sem_lendario = $6
		RETURNING moedas`,
		custo, pityNovo.SemEpico, pityNovo.SemLendario, jogadorID, pityLido.SemEpico, pityLido.SemLendario,
	).Scan(&saldo)
	if errors.Is(err, pgx.ErrNoRows) {
		// nenhuma linha: jogador não existe, sem saldo, ou o pity mudou.
		j, errBusca := r.BuscarPorID(ctx, jogadorID)
		if errBusca != nil {
			return dominio.ItemColecao{}, 0, errBusca
		}
		if j.Moedas < custo {
			return dominio.ItemColecao{}, 0, dominio.ErrSaldoInsuficiente
		}
		return dominio.ItemColecao{}, 0, dominio.ErrCompraSimultanea
	}
	if err != nil {
		return dominio.ItemColecao{}, 0, fmt.Errorf("debitar moedas: %w", err)
	}

	d := p.Detalhes
	it := dominio.ItemColecao{PokemonID: p.PokemonID, Nome: p.Nome, Tier: d.Tier, Tipos: d.Tipos, Detalhes: &d}
	err = tx.QueryRow(ctx, `
		INSERT INTO colecao (jogador_id, pokemon_id, nome, tier, tipos)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, adquirido_em`,
		jogadorID, p.PokemonID, p.Nome, d.Tier, d.Tipos,
	).Scan(&it.ID, &it.AdquiridoEm)
	if err != nil {
		return dominio.ItemColecao{}, 0, fmt.Errorf("adicionar à coleção: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dominio.ItemColecao{}, 0, fmt.Errorf("confirmar compra: %w", err)
	}
	return it, saldo, nil
}

// soma moedas e devolve o saldo novo.
func (r *Postgres) Creditar(ctx context.Context, jogadorID int64, quantidade int) (int, error) {
	var saldo int
	err := r.pool.QueryRow(ctx,
		`UPDATE jogadores SET moedas = moedas + $1 WHERE id = $2 RETURNING moedas`,
		quantidade, jogadorID,
	).Scan(&saldo)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("%w: id %d", dominio.ErrJogadorNaoEncontrado, jogadorID)
	}
	if err != nil {
		return 0, fmt.Errorf("creditar moedas: %w", err)
	}
	return saldo, nil
}

// quais dos IDs o jogador não possui.
func (r *Postgres) PokemonsFaltando(ctx context.Context, jogadorID int64, pokemonIDs []int) ([]int, error) {
	linhas, err := r.pool.Query(ctx,
		`SELECT DISTINCT pokemon_id FROM colecao WHERE jogador_id = $1 AND pokemon_id = ANY($2)`,
		jogadorID, pokemonIDs)
	if err != nil {
		return nil, fmt.Errorf("conferir coleção: %w", err)
	}
	possuidos, err := pgx.CollectRows(linhas, pgx.RowTo[int])
	if err != nil {
		return nil, fmt.Errorf("ler coleção: %w", err)
	}

	var faltando []int
	for _, id := range pokemonIDs {
		if !slices.Contains(possuidos, id) {
			faltando = append(faltando, id)
		}
	}
	return faltando, nil
}
