// Package clientes contém os clients gRPC que a batalha usa para falar
// com os microsserviços de catálogo e de jogadores.
package clientes

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	catalogov1 "pokebatalha/gen/catalogo/v1"
	"pokebatalha/services/batalha/internal/dominio"
)

func conectar(endereco string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(endereco, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("conectar em %s: %w", endereco, err)
	}
	return conn, nil
}

// erroDeInfra devolve um erro de infraestrutura se o código gRPC indicar
// serviço fora do ar ou timeout; caso contrário devolve nil.
func erroDeInfra(servico string, err error) error {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%w: %s: %v", dominio.ErrServicoIndisponivel, servico, err)
	}
	return nil
}

// erroRemoto preserva a mensagem que o outro serviço mandou, mas responde a
// errors.Is com o erro de domínio local (sem repetir o texto na mensagem).
type erroRemoto struct {
	sentinela error
	mensagem  string
}

func (e *erroRemoto) Error() string { return e.mensagem }
func (e *erroRemoto) Unwrap() error { return e.sentinela }

func novoErroRemoto(sentinela, err error) error {
	return &erroRemoto{sentinela: sentinela, mensagem: status.Convert(err).Message()}
}

// Catalogo implementa servico.Catalogo sobre gRPC.
type Catalogo struct {
	api     catalogov1.CatalogoClient
	timeout time.Duration
}

// ConectarCatalogo abre a conexão. Quem chama deve fechar a *grpc.ClientConn.
func ConectarCatalogo(endereco string, timeout time.Duration) (*Catalogo, *grpc.ClientConn, error) {
	conn, err := conectar(endereco)
	if err != nil {
		return nil, nil, err
	}
	return &Catalogo{api: catalogov1.NewCatalogoClient(conn), timeout: timeout}, conn, nil
}

// BuscarPokemon pede os dados de um Pokémon pelo ID.
func (c *Catalogo) BuscarPokemon(ctx context.Context, id int) (dominio.Especie, error) {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	p, err := c.api.BuscarPokemon(ctx, &catalogov1.BuscarPokemonRequest{Id: int32(id)})
	if err != nil {
		return dominio.Especie{}, c.traduzir(err)
	}
	return paraEspecie(p), nil
}

// SortearPokemon pede um Pokémon aleatório do tier com poder próximo do
// informado (para montar o time da CPU equilibrado com o do jogador).
func (c *Catalogo) SortearPokemon(ctx context.Context, tier string, poder int) (dominio.Especie, error) {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	p, err := c.api.SortearPokemon(ctx, &catalogov1.SortearPokemonRequest{Tier: tier, PoderReferencia: int32(poder)})
	if err != nil {
		return dominio.Especie{}, c.traduzir(err)
	}
	return paraEspecie(p), nil
}

// TabelaDeTipos pede as relações de dano dos 18 tipos.
func (c *Catalogo) TabelaDeTipos(ctx context.Context) (dominio.Tabela, error) {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	resp, err := c.api.TabelaDeTipos(ctx, &catalogov1.TabelaDeTiposRequest{})
	if err != nil {
		return nil, c.traduzir(err)
	}
	relacoes := make([]dominio.RelacaoTipo, 0, len(resp.GetRelacoes()))
	for _, r := range resp.GetRelacoes() {
		relacoes = append(relacoes, dominio.RelacaoTipo{
			Tipo:       r.GetTipo(),
			DobroPara:  r.GetDobroPara(),
			MetadePara: r.GetMetadePara(),
			NenhumPara: r.GetNenhumPara(),
		})
	}
	return dominio.NovaTabela(relacoes), nil
}

func (c *Catalogo) traduzir(err error) error {
	if e := erroDeInfra("catálogo", err); e != nil {
		return e
	}
	if status.Code(err) == codes.NotFound {
		return novoErroRemoto(dominio.ErrPokemonNaoEncontrado, err)
	}
	return fmt.Errorf("catálogo respondeu erro inesperado: %w", err)
}

func paraEspecie(p *catalogov1.Pokemon) dominio.Especie {
	s := p.GetStatus()
	return dominio.Especie{
		PokemonID:       int(p.GetId()),
		Nome:            p.GetNome(),
		Tier:            p.GetTier(),
		Poder:           int(p.GetPoder().GetTotal()),
		HabilidadeAtiva: p.GetPoder().GetHabilidadeAtiva(),
		Tipos:           p.GetTipos(),
		Habilidades:     p.GetHabilidades(),
		AprendeFakeOut:  p.GetAprendeFakeOut(),
		HP:              int(s.GetHp()),
		Ataque:          int(s.GetAtaque()),
		Defesa:          int(s.GetDefesa()),
		AtaqueEspecial:  int(s.GetAtaqueEspecial()),
		DefesaEspecial:  int(s.GetDefesaEspecial()),
		Velocidade:      int(s.GetVelocidade()),
	}
}
