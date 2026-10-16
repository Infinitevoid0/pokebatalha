package clientes

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	jogadoresv1 "pokebatalha/gen/jogadores/v1"
	"pokebatalha/services/batalha/internal/dominio"
)

// responde ValidarTime com o código gRPC configurado.
type servidorFalso struct {
	jogadoresv1.UnimplementedJogadoresServer
	codigo codes.Code
	atraso time.Duration
}

func (s *servidorFalso) ValidarTime(ctx context.Context, _ *jogadoresv1.ValidarTimeRequest) (*jogadoresv1.ValidarTimeResponse, error) {
	if s.atraso > 0 {
		select {
		case <-time.After(s.atraso):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.codigo != codes.OK {
		return nil, status.Error(s.codigo, "falha simulada")
	}
	return &jogadoresv1.ValidarTimeResponse{}, nil
}

// servidor gRPC em memória (bufconn), sem rede.
func novoCliente(t *testing.T, srv *servidorFalso, timeout time.Duration) *Jogadores {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	s := grpc.NewServer()
	jogadoresv1.RegisterJogadoresServer(s, srv)
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("servidor de teste encerrado: %v", err)
		}
	}()
	t.Cleanup(s.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &Jogadores{api: jogadoresv1.NewJogadoresClient(conn), timeout: timeout}
}

func TestJogadores_TraduzCodigosGRPC(t *testing.T) {
	casos := []struct {
		codigo   codes.Code
		esperado error
	}{
		{codes.NotFound, dominio.ErrJogadorNaoEncontrado},
		{codes.FailedPrecondition, dominio.ErrPokemonNaoPertence},
		{codes.InvalidArgument, dominio.ErrTimeInvalido},
		{codes.Unavailable, dominio.ErrServicoIndisponivel},
	}
	for _, c := range casos {
		t.Run(c.codigo.String(), func(t *testing.T) {
			cli := novoCliente(t, &servidorFalso{codigo: c.codigo}, time.Second)
			err := cli.ValidarTime(context.Background(), 1, []int{727})
			if !errors.Is(err, c.esperado) {
				t.Fatalf("esperava %v, veio %v", c.esperado, err)
			}
		})
	}
}

func TestJogadores_TimeoutViraErroDeInfra(t *testing.T) {
	cli := novoCliente(t, &servidorFalso{atraso: time.Second}, 50*time.Millisecond)
	err := cli.ValidarTime(context.Background(), 1, []int{727})
	if !errors.Is(err, dominio.ErrServicoIndisponivel) {
		t.Fatalf("timeout deveria virar ErrServicoIndisponivel, veio %v", err)
	}
}

func TestJogadores_Sucesso(t *testing.T) {
	cli := novoCliente(t, &servidorFalso{}, time.Second)
	if err := cli.ValidarTime(context.Background(), 1, []int{727}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
}
