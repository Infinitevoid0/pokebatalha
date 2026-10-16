// microsserviço de catálogo: PokeAPI com cache no banco, REST e gRPC.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	catalogov1 "pokebatalha/gen/catalogo/v1"
	"pokebatalha/pkg/banco"
	"pokebatalha/pkg/execucao"
	"pokebatalha/pkg/observabilidade"
	"pokebatalha/pkg/sorteio"
	"pokebatalha/services/catalogo/internal/config"
	"pokebatalha/services/catalogo/internal/grpcapi"
	"pokebatalha/services/catalogo/internal/httpapi"
	"pokebatalha/services/catalogo/internal/pokeapi"
	"pokebatalha/services/catalogo/internal/repositorio"
	"pokebatalha/services/catalogo/internal/servico"
)

func main() {
	log := observabilidade.NovoLogger("catalogo")
	if err := rodar(log); err != nil {
		log.Error("catalogo encerrado com erro", "erro", err)
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

	fonte := pokeapi.NovoCliente(cfg.PokeAPIURL, cfg.PokeAPITimeout)
	svc := servico.Novo(repo, fonte, sorteio.Aleatorio{}, log)

	// baixa a Pokédex para o cache em segundo plano (só na primeira vez).
	if cfg.AquecerCache {
		go svc.Aquecer(ctx)
	}

	httpSrv := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           observabilidade.MiddlewareHTTP(log, httpapi.Novo(svc, log).Rotas()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(observabilidade.InterceptadorGRPC(log)))
	catalogov1.RegisterCatalogoServer(grpcSrv, grpcapi.Novo(svc, log))

	return execucao.Rodar(ctx, log, httpSrv, grpcSrv, ":"+cfg.PortaGRPC)
}
