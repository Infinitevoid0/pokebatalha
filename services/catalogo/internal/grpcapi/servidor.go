// Package grpcapi expõe o catálogo para os outros microsserviços via gRPC.
package grpcapi

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogov1 "pokebatalha/gen/catalogo/v1"
	"pokebatalha/services/catalogo/internal/dominio"
)

// Servico é o que o servidor gRPC precisa da camada de negócio.
type Servico interface {
	BuscarPorID(ctx context.Context, id int) (dominio.Pokemon, error)
	BuscarVarios(ctx context.Context, ids []int) ([]dominio.Pokemon, error)
	Sortear(ctx context.Context, tier dominio.Tier, poderReferencia int) (dominio.Pokemon, error)
	TabelaDeTipos(ctx context.Context) ([]dominio.RelacaoTipo, error)
}

// Servidor implementa catalogov1.CatalogoServer.
type Servidor struct {
	catalogov1.UnimplementedCatalogoServer
	svc Servico
	log *slog.Logger
}

// Novo cria o servidor gRPC.
func Novo(svc Servico, log *slog.Logger) *Servidor {
	return &Servidor{svc: svc, log: log}
}

func (s *Servidor) BuscarPokemon(ctx context.Context, req *catalogov1.BuscarPokemonRequest) (*catalogov1.Pokemon, error) {
	p, err := s.svc.BuscarPorID(ctx, int(req.GetId()))
	if err != nil {
		return nil, s.paraStatus(err)
	}
	return ParaProto(p), nil
}

func (s *Servidor) BuscarPokemons(ctx context.Context, req *catalogov1.BuscarPokemonsRequest) (*catalogov1.BuscarPokemonsResponse, error) {
	ids := make([]int, 0, len(req.GetIds()))
	for _, id := range req.GetIds() {
		ids = append(ids, int(id))
	}
	lista, err := s.svc.BuscarVarios(ctx, ids)
	if err != nil {
		return nil, s.paraStatus(err)
	}
	resp := &catalogov1.BuscarPokemonsResponse{}
	for _, p := range lista {
		resp.Pokemons = append(resp.Pokemons, ParaProto(p))
	}
	return resp, nil
}

func (s *Servidor) SortearPokemon(ctx context.Context, req *catalogov1.SortearPokemonRequest) (*catalogov1.Pokemon, error) {
	tier, err := dominio.ParseTier(req.GetTier())
	if err != nil {
		return nil, s.paraStatus(err)
	}
	p, err := s.svc.Sortear(ctx, tier, int(req.GetPoderReferencia()))
	if err != nil {
		return nil, s.paraStatus(err)
	}
	return ParaProto(p), nil
}

func (s *Servidor) TabelaDeTipos(ctx context.Context, _ *catalogov1.TabelaDeTiposRequest) (*catalogov1.TabelaDeTiposResponse, error) {
	tabela, err := s.svc.TabelaDeTipos(ctx)
	if err != nil {
		return nil, s.paraStatus(err)
	}
	resp := &catalogov1.TabelaDeTiposResponse{}
	for _, t := range tabela {
		resp.Relacoes = append(resp.Relacoes, &catalogov1.RelacaoTipo{
			Tipo:       t.Tipo,
			DobroPara:  t.DobroPara,
			MetadePara: t.MetadePara,
			NenhumPara: t.NenhumPara,
		})
	}
	return resp, nil
}

// paraStatus traduz erros do domínio para códigos gRPC, mantendo a mesma
// separação negócio x infraestrutura usada no HTTP.
func (s *Servidor) paraStatus(err error) error {
	switch {
	case errors.Is(err, dominio.ErrIDInvalido),
		errors.Is(err, dominio.ErrTierInvalido),
		errors.Is(err, dominio.ErrTipoInvalido):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, dominio.ErrPokemonNaoEncontrado):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dominio.ErrServicoExternoIndisponivel),
		errors.Is(err, dominio.ErrCatalogoAquecendo):
		return status.Error(codes.Unavailable, err.Error())
	default:
		s.log.Error("erro interno no gRPC", "erro", err)
		return status.Error(codes.Internal, "erro interno do catálogo")
	}
}

// ParaProto converte o Pokémon do domínio para a mensagem gRPC.
func ParaProto(p dominio.Pokemon) *catalogov1.Pokemon {
	return &catalogov1.Pokemon{
		Id:             int32(p.ID),
		Nome:           p.Nome,
		Tipos:          p.Tipos,
		Habilidades:    p.Habilidades,
		AprendeFakeOut: p.AprendeFakeOut,
		Tier:           string(p.Tier),
		Status: &catalogov1.Status{
			Hp:             int32(p.Status.HP),
			Ataque:         int32(p.Status.Ataque),
			Defesa:         int32(p.Status.Defesa),
			AtaqueEspecial: int32(p.Status.AtaqueEspecial),
			DefesaEspecial: int32(p.Status.DefesaEspecial),
			Velocidade:     int32(p.Status.Velocidade),
		},
		Poder: &catalogov1.Poder{
			Total:            int32(p.Poder.Total),
			Stats:            int32(p.Poder.Stats),
			TipagemOfensiva:  int32(p.Poder.Ofensivo),
			TipagemDefensiva: int32(p.Poder.Defensivo),
			Habilidade:       int32(p.Poder.Habilidade),
			FakeOut:          int32(p.Poder.FakeOut),
			HabilidadeAtiva:  p.Poder.HabilidadeAtiva,
		},
	}
}
