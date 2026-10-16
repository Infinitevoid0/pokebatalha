package dominio

import (
	"fmt"
	"slices"
)

// os 18 tipos, com os nomes da PokeAPI.
var TiposExistentes = []string{
	"normal", "fire", "water", "electric", "grass", "ice",
	"fighting", "poison", "ground", "flying", "psychic", "bug",
	"rock", "ghost", "dragon", "dark", "steel", "fairy",
}

// como um tipo atacante se sai contra os outros.
type RelacaoTipo struct {
	Tipo       string   `json:"tipo"`
	DobroPara  []string `json:"dobro_para"`
	MetadePara []string `json:"metade_para"`
	NenhumPara []string `json:"nenhum_para"`
}

// confere se é um dos 18 tipos.
func ValidarTipo(tipo string) error {
	if !slices.Contains(TiposExistentes, tipo) {
		return fmt.Errorf("%w: %q", ErrTipoInvalido, tipo)
	}
	return nil
}

// dano de um tipo contra um defensor (ex: fire contra grass/steel = 4x).
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
