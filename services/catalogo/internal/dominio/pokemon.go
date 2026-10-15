// Package dominio contém os tipos e regras do catálogo, sem dependência
// de banco, HTTP ou gRPC.
package dominio

import (
	"errors"
	"fmt"
	"slices"
)

// MaiorIDNacional é o último número da Pokédex nacional (até a 9ª geração).
const MaiorIDNacional = 1025

// Erros de negócio: o pedido do cliente é que está errado ou não existe.
var (
	ErrIDInvalido           = errors.New("id de pokémon inválido")
	ErrTierInvalido         = errors.New("tier inválido")
	ErrTipoInvalido         = errors.New("tipo inválido")
	ErrPokemonNaoEncontrado = errors.New("pokémon não encontrado")
)

// Erros de infraestrutura: algo fora do nosso controle falhou.
var (
	ErrServicoExternoIndisponivel = errors.New("serviço externo indisponível")
	// ErrCatalogoAquecendo: ainda não há Pokémon suficientes no cache para
	// sortear um tier (o catálogo está baixando a Pokédex da PokeAPI).
	ErrCatalogoAquecendo = errors.New("catálogo ainda está carregando a Pokédex")
	// ErrForaDoCache indica que o registro ainda não está no banco local
	// (não é erro para o cliente: o serviço busca na PokeAPI).
	ErrForaDoCache = errors.New("registro não está no cache")
)

// Tier é a raridade do Pokémon no gacha, calculada pelo índice de poder.
type Tier string

const (
	TierComum    Tier = "comum"
	TierRaro     Tier = "raro"
	TierEpico    Tier = "epico"
	TierLendario Tier = "lendario"
)

// Tiers lista os tiers do mais comum ao mais raro.
var Tiers = []Tier{TierComum, TierRaro, TierEpico, TierLendario}

// ParseTier valida o texto recebido do cliente.
func ParseTier(s string) (Tier, error) {
	t := Tier(s)
	if !slices.Contains(Tiers, t) {
		return "", fmt.Errorf("%w: %q (use comum, raro, epico ou lendario)", ErrTierInvalido, s)
	}
	return t, nil
}

// Status são os atributos base do Pokémon.
type Status struct {
	HP             int `json:"hp"`
	Ataque         int `json:"ataque"`
	Defesa         int `json:"defesa"`
	AtaqueEspecial int `json:"ataque_especial"`
	DefesaEspecial int `json:"defesa_especial"`
	Velocidade     int `json:"velocidade"`
}

// Pokemon é a forma como o catálogo representa um Pokémon do jogo.
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
