// Package execucao sobe os servidores HTTP e gRPC de um serviço e faz o
// desligamento gracioso quando o processo recebe SIGINT/SIGTERM.
package execucao

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

const tempoDesligamento = 10 * time.Second

// Rodar bloqueia até ctx ser cancelado ou um dos servidores falhar.
// grpcSrv pode ser nil para serviços que só expõem HTTP.
func Rodar(ctx context.Context, log *slog.Logger, httpSrv *http.Server, grpcSrv *grpc.Server, enderecoGRPC string) error {
	erros := make(chan error, 2)

	go func() {
		log.Info("servidor http iniciado", "endereco", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- fmt.Errorf("servidor http: %w", err)
		}
	}()

	if grpcSrv != nil {
		lis, err := net.Listen("tcp", enderecoGRPC)
		if err != nil {
			return fmt.Errorf("escutar gRPC em %s: %w", enderecoGRPC, err)
		}
		go func() {
			log.Info("servidor grpc iniciado", "endereco", enderecoGRPC)
			if err := grpcSrv.Serve(lis); err != nil {
				erros <- fmt.Errorf("servidor grpc: %w", err)
			}
		}()
	}

	var errExecucao error
	select {
	case <-ctx.Done():
		log.Info("sinal de desligamento recebido")
	case errExecucao = <-erros:
		log.Error("servidor falhou", "erro", errExecucao)
	}

	ctxDesl, cancelar := context.WithTimeout(context.Background(), tempoDesligamento)
	defer cancelar()
	if grpcSrv != nil {
		grpcSrv.GracefulStop()
	}
	if err := httpSrv.Shutdown(ctxDesl); err != nil {
		return fmt.Errorf("desligar http: %w", err)
	}
	log.Info("serviço encerrado")
	return errExecucao
}
