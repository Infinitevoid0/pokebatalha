package dominio

import (
	"fmt"
	"slices"
)

// TiposExistentes são os 18 tipos, com os nomes usados pela PokeAPI.
var TiposExistentes = []string{
	"normal", "fire", "water", "electric", "grass", "ice",
	"fighting", "poison", "ground", "flying", "psychic", "bug",
	"rock", "ghost", "dragon", "dark", "steel", "fairy",
}

// RelacaoTipo descreve como um tipo atacante se sai contra os outros.
type RelacaoTipo struct {
	Tipo       string   `json:"tipo"`
	DobroPara  []string `json:"dobro_para"`
	MetadePara []string `json:"metade_para"`
	NenhumPara []string `json:"nenhum_para"`
}

// ValidarTipo confere se o nome é um dos 18 tipos.
func ValidarTipo(tipo string) error {
	if !slices.Contains(TiposExistentes, tipo) {
		return fmt.Errorf("%w: %q", ErrTipoInvalido, tipo)
	}
	return nil
}

// Multiplicador calcula o dano de um ataque deste tipo contra um defensor
// com um ou dois tipos (ex: fire contra grass/steel = 2 x 2 = 4).
func (r RelacaoTipo) Multiplicador(tiposDefensor []string) float64 {
	m := 1.0
	for _, d := range tiposDefensor {
		switch {
		case slices.Contains(r.NenhumPara, d):
			m *= 0
		case slices.Contains(r.DobroPara, d):
			m *= 2
		case slices.Contains(r.MetadePara, d):
			m *= 0.5
		}
	}
	return m
}
