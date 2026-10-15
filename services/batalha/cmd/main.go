// Comando batalha: microsserviço que roda as partidas por turnos.
// Expõe REST para o jogador e consome catálogo e jogadores via gRPC.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pokebatalha/pkg/banco"
	"pokebatalha/pkg/execucao"
	"pokebatalha/pkg/observabilidade"
	"pokebatalha/pkg/sorteio"
	"pokebatalha/services/batalha/internal/clientes"
	"pokebatalha/services/batalha/internal/config"
	"pokebatalha/services/batalha/internal/httpapi"
	"pokebatalha/services/batalha/internal/repositorio"
	"pokebatalha/services/batalha/internal/servico"
)

func main() {
	log := observabilidade.NovoLogger("batalha")
	if err := rodar(log); err != nil {
		log.Error("batalha encerrada com erro", "erro", err)
		os.Exit(1)
	}
}

func rodar(log *slog.Logger) error {
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	pool, err := banco.Conectar(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return err
	}
	defer pool.Close()

	repo := repositorio.NovoPostgres(pool)
	if err := repo.Migrar(ctx); err != nil {
		return err
	}

	cat, connCat, err := clientes.ConectarCatalogo(cfg.CatalogoGRPCAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer connCat.Close()

	jog, connJog, err := clientes.ConectarJogadores(cfg.JogadoresGRPCAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer connJog.Close()

	svc := servico.Novo(repo, cat, jog, sorteio.Aleatorio{}, cfg.RecompensaVitoria)

	httpSrv := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           observabilidade.MiddlewareHTTP(log, httpapi.Novo(svc, log).Rotas()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// A batalha não expõe gRPC: ninguém depende dela.
	return execucao.Rodar(ctx, log, httpSrv, nil, "")
}
