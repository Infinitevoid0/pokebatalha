// client gRPC do catálogo.
package catalogo

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	catalogov1 "pokebatalha/gen/catalogo/v1"
	"pokebatalha/services/jogadores/internal/dominio"
)

// máximo de IDs por chamada ao catálogo.
const tamanhoLote = 50

// catálogo via gRPC.
type Cliente struct {
	api     catalogov1.CatalogoClient
	timeout time.Duration
}

// quem chama fecha a conexão.
func Conectar(endereco string, timeout time.Duration) (*Cliente, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(endereco, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("conectar ao catálogo em %s: %w", endereco, err)
	}
	return &Cliente{api: catalogov1.NewCatalogoClient(conn), timeout: timeout}, conn, nil
}

// pede um Pokémon aleatório do tier.
func (c *Cliente) SortearPokemon(ctx context.Context, tier string) (dominio.PokemonSorteado, error) {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	p, err := c.api.SortearPokemon(ctx, &catalogov1.SortearPokemonRequest{Tier: tier})
	if err != nil {
		return dominio.PokemonSorteado{}, traduzirErro(err)
	}
	return dominio.PokemonSorteado{PokemonID: int(p.GetId()), Nome: p.GetNome(), Detalhes: detalhes(p)}, nil
}

// dados atuais de cada Pokémon, por ID.
func (c *Cliente) BuscarDetalhes(ctx context.Context, ids []int) (map[int]dominio.Detalhes, error) {
	resultado := make(map[int]dominio.Detalhes, len(ids))
	for inicio := 0; inicio < len(ids); inicio += tamanhoLote {
		lote := ids[inicio:min(inicio+tamanhoLote, len(ids))]
		req := &catalogov1.BuscarPokemonsRequest{}
		for _, id := range lote {
			req.Ids = append(req.Ids, int32(id))
		}

		ctxLote, cancelar := context.WithTimeout(ctx, c.timeout)
		resp, err := c.api.BuscarPokemons(ctxLote, req)
		cancelar()
		if err != nil {
			return nil, traduzirErro(err)
		}
		for _, p := range resp.GetPokemons() {
			resultado[int(p.GetId())] = detalhes(p)
		}
	}
	return resultado, nil
}

func detalhes(p *catalogov1.Pokemon) dominio.Detalhes {
	s, pd := p.GetStatus(), p.GetPoder()
	return dominio.Detalhes{
		Tier:  p.GetTier(),
		Tipos: p.GetTipos(),
		Poder: dominio.ComposicaoPoder{
			Total:           int(pd.GetTotal()),
			Stats:           int(pd.GetStats()),
			Tipagem:         int(pd.GetTipagemOfensiva() + pd.GetTipagemDefensiva()),
			Habilidade:      int(pd.GetHabilidade()),
			FakeOut:         int(pd.GetFakeOut()),
			HabilidadeAtiva: pd.GetHabilidadeAtiva(),
		},
		Stats: dominio.Stats{
			HP:             int(s.GetHp()),
			Ataque:         int(s.GetAtaque()),
			Defesa:         int(s.GetDefesa()),
			AtaqueEspecial: int(s.GetAtaqueEspecial()),
			DefesaEspecial: int(s.GetDefesaEspecial()),
			Velocidade:     int(s.GetVelocidade()),
		},
	}
}

// status gRPC -> erro do domínio.
func traduzirErro(err error) error {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		// catálogo (ou PokeAPI atrás dele) fora: infraestrutura.
		return fmt.Errorf("%w: catálogo: %v", dominio.ErrServicoIndisponivel, err)
	default:
		return fmt.Errorf("catálogo respondeu erro inesperado: %w", err)
	}
}
