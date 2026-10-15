// Package banco abre o pool de conexões com o Postgres, tentando de novo
// enquanto o banco ainda está subindo (comum no docker compose).
package banco

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	tentativas      = 10
	esperaEntreElas = 2 * time.Second
)

// Conectar cria o pool e só retorna quando o banco responder ao ping.
func Conectar(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("configurar pool do postgres: %w", err)
	}

	for i := 1; i <= tentativas; i++ {
		ctxPing, cancelar := context.WithTimeout(ctx, 3*time.Second)
		err = pool.Ping(ctxPing)
		cancelar()
		if err == nil {
			return pool, nil
		}
		log.Warn("postgres ainda indisponível, tentando de novo", "tentativa", i, "erro", err)

		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(esperaEntreElas):
		}
	}
	pool.Close()
	return nil, fmt.Errorf("postgres não respondeu após %d tentativas: %w", tentativas, err)
}
