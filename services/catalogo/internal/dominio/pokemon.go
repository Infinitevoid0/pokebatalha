// tipos e regras do catálogo, sem banco, HTTP ou gRPC.
package dominio

import (
	"errors"
	"fmt"
	"slices"
)

// último número da Pokédex nacional (9ª geração).
const MaiorIDNacional = 1025

// erros de negócio: o pedido do cliente está errado ou não existe.
var (
	ErrIDInvalido           = errors.New("id de pokémon inválido")
	ErrTierInvalido         = errors.New("tier inválido")
	ErrTipoInvalido         = errors.New("tipo inválido")
	ErrPokemonNaoEncontrado = errors.New("pokémon não encontrado")
)

// erros de infraestrutura: algo fora do nosso controle falhou.
var (
	ErrServicoExternoIndisponivel = errors.New("serviço externo indisponível")
	// cache ainda sem Pokémon do tier (baixando a Pokédex).
	ErrCatalogoAquecendo = errors.New("catálogo ainda está carregando a Pokédex")
	// fora do banco local; não é erro para o cliente, busca na PokeAPI.
	ErrForaDoCache = errors.New("registro não está no cache")
)

// raridade no gacha, pelo índice de poder.
type Tier string

const (
	TierComum    Tier = "comum"
	TierRaro     Tier = "raro"
	TierEpico    Tier = "epico"
	TierLendario Tier = "lendario"
)

// do mais comum ao mais raro.
var Tiers = []Tier{TierComum, TierRaro, TierEpico, TierLendario}

// valida o tier vindo do cliente.
func ParseTier(s string) (Tier, error) {
	t := Tier(s)
	if !slices.Contains(Tiers, t) {
		return "", fmt.Errorf("%w: %q (use comum, raro, epico ou lendario)", ErrTierInvalido, s)
	}
	return t, nil
}

// atributos base.
type Status struct {
	HP             int `json:"hp"`
	Ataque         int `json:"ataque"`
	Defesa         int `json:"defesa"`
	AtaqueEspecial int `json:"ataque_especial"`
	DefesaEspecial int `json:"defesa_especial"`
	Velocidade     int `json:"velocidade"`
}

// um Pokémon como o catálogo representa.
type Pokemon struct {
	ID             int      `json:"id"`
	Nome           string   `json:"nome"`
	Tipos          []string `json:"tipos"`
	Habilidades    []string `json:"habilidades"`
	AprendeFakeOut bool     `json:"aprende_fake_out"`
	Status         Status   `json:"status"`
	Poder          Poder    `json:"poder"`
	Tier           Tier     `json:"tier"`
}
