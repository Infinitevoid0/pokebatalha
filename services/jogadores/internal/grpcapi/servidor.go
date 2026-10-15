// Package grpcapi expõe o serviço de jogadores para a batalha via gRPC.
package grpcapi

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jogadoresv1 "pokebatalha/gen/jogadores/v1"
	"pokebatalha/services/jogadores/internal/dominio"
)

// Servico é o que o servidor gRPC precisa da camada de negócio.
type Servico interface {
	ValidarTime(ctx context.Context, id int64, pokemonIDs []int) error
	CreditarMoedas(ctx context.Context, id int64, quantidade int) (int, error)
}

// Servidor implementa jogadoresv1.JogadoresServer.
type Servidor struct {
	jogadoresv1.UnimplementedJogadoresServer
	svc Servico
	log *slog.Logger
}

// Novo cria o servidor gRPC.
func Novo(svc Servico, log *slog.Logger) *Servidor {
	return &Servidor{svc: svc, log: log}
}

func (s *Servidor) ValidarTime(ctx context.Context, req *jogadoresv1.ValidarTimeRequest) (*jogadoresv1.ValidarTimeResponse, error) {
	ids := make([]int, 0, len(req.GetPokemonIds()))
	for _, id := range req.GetPokemonIds() {
		ids = append(ids, int(id))
	}
	if err := s.svc.ValidarTime(ctx, req.GetJogadorId(), ids); err != nil {
		return nil, s.paraStatus(err)
	}
	return &jogadoresv1.ValidarTimeResponse{}, nil
}

func (s *Servidor) CreditarMoedas(ctx context.Context, req *jogadoresv1.CreditarMoedasRequest) (*jogadoresv1.CreditarMoedasResponse, error) {
	saldo, err := s.svc.CreditarMoedas(ctx, req.GetJogadorId(), int(req.GetQuantidade()))
	if err != nil {
		return nil, s.paraStatus(err)
	}
	return &jogadoresv1.CreditarMoedasResponse{Saldo: int32(saldo)}, nil
}

func (s *Servidor) paraStatus(err error) error {
	switch {
	case errors.Is(err, dominio.ErrTimeInvalido), errors.Is(err, dominio.ErrQuantidadeInvalida):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, dominio.ErrJogadorNaoEncontrado):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dominio.ErrPokemonNaoPertence):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		s.log.Error("erro interno no gRPC", "erro", err)
		return status.Error(codes.Internal, "erro interno do serviço de jogadores")
	}
}
