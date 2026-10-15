// Package repositorio isola o acesso ao banco do catálogo.
package repositorio

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pokebatalha/services/catalogo/internal/dominio"
)

//go:embed schema.sql
var schema string

// Postgres guarda o cache de Pokémon e tipos.
type Postgres struct {
	pool *pgxpool.Pool
}

// NovoPostgres cria o repositório sobre um pool já conectado.
func NovoPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// Migrar cria as tabelas, se ainda não existirem.
func (r *Postgres) Migrar(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("criar tabelas do catálogo: %w", err)
	}
	return nil
}

const colunasPokemon = `id, nome, tipos, habilidades, aprende_fake_out,
	hp, ataque, defesa, ataque_especial, defesa_especial, velocidade`

// BuscarPokemonPorID devolve dominio.ErrForaDoCache se o ID não estiver salvo.
func (r *Postgres) BuscarPokemonPorID(ctx context.Context, id int) (dominio.Pokemon, error) {
	linha := r.pool.QueryRow(ctx, `SELECT `+colunasPokemon+` FROM pokemons WHERE id = $1`, id)
	return escanearPokemon(linha)
}

// BuscarPokemonPorNome devolve dominio.ErrForaDoCache se o nome não estiver salvo.
func (r *Postgres) BuscarPokemonPorNome(ctx context.Context, nome string) (dominio.Pokemon, error) {
	linha := r.pool.QueryRow(ctx, `SELECT `+colunasPokemon+` FROM pokemons WHERE nome = $1`, nome)
	return escanearPokemon(linha)
}

func escanearPokemon(linha pgx.Row) (dominio.Pokemon, error) {
	var p dominio.Pokemon
	err := linha.Scan(&p.ID, &p.Nome, &p.Tipos, &p.Habilidades, &p.AprendeFakeOut,
		&p.Status.HP, &p.Status.Ataque, &p.Status.Defesa,
		&p.Status.AtaqueEspecial, &p.Status.DefesaEspecial, &p.Status.Velocidade)
	if errors.Is(err, pgx.ErrNoRows) {
		return dominio.Pokemon{}, dominio.ErrForaDoCache
	}
	if err != nil {
		return dominio.Pokemon{}, fmt.Errorf("ler pokémon do banco: %w", err)
	}
	return p, nil
}

// ListarPokemons devolve todo o cache (no máximo ~1025 linhas pequenas).
func (r *Postgres) ListarPokemons(ctx context.Context) ([]dominio.Pokemon, error) {
	linhas, err := r.pool.Query(ctx, `SELECT `+colunasPokemon+` FROM pokemons ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listar pokémon do cache: %w", err)
	}
	defer linhas.Close()

	var todos []dominio.Pokemon
	for linhas.Next() {
		p, err := escanearPokemon(linhas)
		if err != nil {
			return nil, err
		}
		todos = append(todos, p)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("percorrer cache de pokémon: %w", err)
	}
	return todos, nil
}

// SalvarPokemon insere ou atualiza o Pokémon no cache.
func (r *Postgres) SalvarPokemon(ctx context.Context, p dominio.Pokemon) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pokemons (`+colunasPokemon+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			nome = EXCLUDED.nome, tipos = EXCLUDED.tipos, habilidades = EXCLUDED.habilidades,
			aprende_fake_out = EXCLUDED.aprende_fake_out, hp = EXCLUDED.hp,
			ataque = EXCLUDED.ataque, defesa = EXCLUDED.defesa,
			ataque_especial = EXCLUDED.ataque_especial, defesa_especial = EXCLUDED.defesa_especial,
			velocidade = EXCLUDED.velocidade, atualizado_em = now()`,
		p.ID, p.Nome, p.Tipos, p.Habilidades, p.AprendeFakeOut,
		p.Status.HP, p.Status.Ataque, p.Status.Defesa,
		p.Status.AtaqueEspecial, p.Status.DefesaEspecial, p.Status.Velocidade)
	if err != nil {
		return fmt.Errorf("salvar pokémon %d: %w", p.ID, err)
	}
	return nil
}

// BuscarTipo devolve dominio.ErrForaDoCache se o tipo não estiver salvo.
func (r *Postgres) BuscarTipo(ctx context.Context, nome string) (dominio.RelacaoTipo, error) {
	t := dominio.RelacaoTipo{Tipo: nome}
	err := r.pool.QueryRow(ctx,
		`SELECT dobro_para, metade_para, nenhum_para FROM tipos WHERE nome = $1`, nome,
	).Scan(&t.DobroPara, &t.MetadePara, &t.NenhumPara)
	if errors.Is(err, pgx.ErrNoRows) {
		return dominio.RelacaoTipo{}, dominio.ErrForaDoCache
	}
	if err != nil {
		return dominio.RelacaoTipo{}, fmt.Errorf("ler tipo do banco: %w", err)
	}
	return t, nil
}

// SalvarTipo insere ou atualiza as relações de um tipo.
func (r *Postgres) SalvarTipo(ctx context.Context, t dominio.RelacaoTipo) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tipos (nome, dobro_para, metade_para, nenhum_para)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (nome) DO UPDATE SET
			dobro_para = EXCLUDED.dobro_para, metade_para = EXCLUDED.metade_para,
			nenhum_para = EXCLUDED.nenhum_para, atualizado_em = now()`,
		t.Tipo, t.DobroPara, t.MetadePara, t.NenhumPara)
	if err != nil {
		return fmt.Errorf("salvar tipo %s: %w", t.Tipo, err)
	}
	return nil
}
