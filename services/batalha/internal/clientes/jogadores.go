package clientes

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jogadoresv1 "pokebatalha/gen/jogadores/v1"
	"pokebatalha/services/batalha/internal/dominio"
)

// jogadores via gRPC.
type Jogadores struct {
	api     jogadoresv1.JogadoresClient
	timeout time.Duration
}

// quem chama fecha a conexão.
func ConectarJogadores(endereco string, timeout time.Duration) (*Jogadores, *grpc.ClientConn, error) {
	conn, err := conectar(endereco)
	if err != nil {
		return nil, nil, err
	}
	return &Jogadores{api: jogadoresv1.NewJogadoresClient(conn), timeout: timeout}, conn, nil
}

// confere se o jogador tem os Pokémon do time.
func (j *Jogadores) ValidarTime(ctx context.Context, jogadorID int64, pokemonIDs []int) error {
	ctx, cancelar := context.WithTimeout(ctx, j.timeout)
	defer cancelar()

	ids := make([]int32, 0, len(pokemonIDs))
	for _, id := range pokemonIDs {
		ids = append(ids, int32(id))
	}
	_, err := j.api.ValidarTime(ctx, &jogadoresv1.ValidarTimeRequest{JogadorId: jogadorID, PokemonIds: ids})
	if err != nil {
		return j.traduzir(err)
	}
	return nil
}

// paga a recompensa da vitória.
func (j *Jogadores) CreditarMoedas(ctx context.Context, jogadorID int64, quantidade int) (int, error) {
	ctx, cancelar := context.WithTimeout(ctx, j.timeout)
	defer cancelar()

	resp, err := j.api.CreditarMoedas(ctx, &jogadoresv1.CreditarMoedasRequest{
		JogadorId: jogadorID, Quantidade: int32(quantidade),
	})
	if err != nil {
		return 0, j.traduzir(err)
	}
	return int(resp.GetSaldo()), nil
}

func (j *Jogadores) traduzir(err error) error {
	if e := erroDeInfra("jogadores", err); e != nil {
		return e
	}
	switch status.Code(err) {
	case codes.NotFound:
		return novoErroRemoto(dominio.ErrJogadorNaoEncontrado, err)
	case codes.FailedPrecondition:
		return novoErroRemoto(dominio.ErrPokemonNaoPertence, err)
	case codes.InvalidArgument:
		return novoErroRemoto(dominio.ErrTimeInvalido, err)
	}
	return fmt.Errorf("jogadores respondeu erro inesperado: %w", err)
}
