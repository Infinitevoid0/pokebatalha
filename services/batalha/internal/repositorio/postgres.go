// acesso ao banco de batalhas.
package repositorio

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pokebatalha/services/batalha/internal/dominio"
)

//go:embed schema.sql
var schema string

type Postgres struct {
	pool *pgxpool.Pool
}

func NovoPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

// cria as tabelas se não existirem.
func (r *Postgres) Migrar(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("criar tabelas de batalha: %w", err)
	}
	return nil
}

// insere e preenche ID e versão.
func (r *Postgres) Criar(ctx context.Context, b *dominio.Batalha) error {
	estado, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("serializar batalha: %w", err)
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO batalhas (jogador_id, status, estado) VALUES ($1, $2, $3)
		RETURNING id, versao`, b.JogadorID, b.Status, estado,
	).Scan(&b.ID, &b.Versao)
	if err != nil {
		return fmt.Errorf("inserir batalha: %w", err)
	}
	return nil
}

// devolve ErrBatalhaNaoEncontrada se o ID não existir.
func (r *Postgres) Buscar(ctx context.Context, id int64) (*dominio.Batalha, error) {
	var (
		estado []byte
		versao int
	)
	err := r.pool.QueryRow(ctx, `SELECT estado, versao FROM batalhas WHERE id = $1`, id).Scan(&estado, &versao)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: id %d", dominio.ErrBatalhaNaoEncontrada, id)
	}
	if err != nil {
		return nil, fmt.Errorf("buscar batalha: %w", err)
	}

	var b dominio.Batalha
	if err := json.Unmarshal(estado, &b); err != nil {
		return nil, fmt.Errorf("ler estado da batalha %d: %w", id, err)
	}
	b.ID, b.Versao = id, versao
	return &b, nil
}

// grava só se ninguém gravou antes (concorrência otimista pela coluna versao).
func (r *Postgres) Atualizar(ctx context.Context, b *dominio.Batalha) error {
	estado, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("serializar batalha: %w", err)
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE batalhas SET estado = $1, status = $2, versao = versao + 1, atualizado_em = now()
		WHERE id = $3 AND versao = $4`, estado, b.Status, b.ID, b.Versao)
	if err != nil {
		return fmt.Errorf("atualizar batalha: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return dominio.ErrConflitoDeVersao
	}
	b.Versao++
	return nil
}
