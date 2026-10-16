// microsserviço de jogadores: contas, moedas, coleção e gacha.
// expõe REST para o jogador e gRPC para a batalha.
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

	jogadoresv1 "pokebatalha/gen/jogadores/v1"
	"pokebatalha/pkg/banco"
	"pokebatalha/pkg/execucao"
	"pokebatalha/pkg/observabilidade"
	"pokebatalha/pkg/sorteio"
	"pokebatalha/services/jogadores/internal/catalogo"
	"pokebatalha/services/jogadores/internal/config"
	"pokebatalha/services/jogadores/internal/grpcapi"
	"pokebatalha/services/jogadores/internal/httpapi"
	"pokebatalha/services/jogadores/internal/repositorio"
	"pokebatalha/services/jogadores/internal/servico"
)

func main() {
	log := observabilidade.NovoLogger("jogadores")
	if err := rodar(log); err != nil {
		log.Error("jogadores encerrado com erro", "erro", err)
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

	cat, conn, err := catalogo.Conectar(cfg.CatalogoGRPCAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	svc := servico.Novo(repo, cat, sorteio.Aleatorio{}, servico.Regras{
		MoedasIniciais: cfg.MoedasIniciais,
		PrecoPacote:    cfg.PrecoPacote,
	})

	httpSrv := &http.Server{
		Addr:              ":" + cfg.PortaHTTP,
		Handler:           observabilidade.MiddlewareHTTP(log, httpapi.Novo(svc, log).Rotas()),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(observabilidade.InterceptadorGRPC(log)))
	jogadoresv1.RegisterJogadoresServer(grpcSrv, grpcapi.Novo(svc, log))

	return execucao.Rodar(ctx, log, httpSrv, grpcSrv, ":"+cfg.PortaGRPC)
}
