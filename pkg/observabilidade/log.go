// Package observabilidade concentra logs estruturados (slog em JSON)
// e os middlewares que registram cada requisição HTTP e chamada gRPC.
package observabilidade

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// NovoLogger cria um logger JSON que marca todas as linhas com o nome do serviço.
func NovoLogger(servico string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With("servico", servico)
}

type respostaComStatus struct {
	http.ResponseWriter
	status int
}

func (r *respostaComStatus) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// MiddlewareHTTP registra método, caminho, status e duração de cada requisição.
func MiddlewareHTTP(log *slog.Logger, prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		rw := &respostaComStatus{ResponseWriter: w, status: http.StatusOK}
		prox.ServeHTTP(rw, r)

		nivel := slog.LevelInfo
		if rw.status >= 500 {
			nivel = slog.LevelError
		}
		log.Log(r.Context(), nivel, "requisição http",
			"metodo", r.Method,
			"caminho", r.URL.Path,
			"status", rw.status,
			"duracao_ms", time.Since(inicio).Milliseconds(),
		)
	})
}

// InterceptadorGRPC registra cada chamada gRPC recebida pelo servidor.
func InterceptadorGRPC(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		inicio := time.Now()
		resp, err := handler(ctx, req)

		codigo := status.Code(err)
		nivel := slog.LevelInfo
		if err != nil {
			nivel = slog.LevelWarn
		}
		log.Log(ctx, nivel, "chamada grpc",
			"metodo", info.FullMethod,
			"codigo", codigo.String(),
			"duracao_ms", time.Since(inicio).Milliseconds(),
		)
		return resp, err
	}
}
