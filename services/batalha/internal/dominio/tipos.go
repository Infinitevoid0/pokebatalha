package dominio

import (
	"slices"
	"strings"
)

// multiplicador de um tipo de ataque contra o defensor.
type TabelaTipos interface {
	Multiplicador(ataque string, tiposDefensor []string) float64
}

// dados de um tipo atacante vindos do catálogo.
type RelacaoTipo struct {
	Tipo       string
	DobroPara  []string
	MetadePara []string
	NenhumPara []string
}

// implementa TabelaTipos com as relações do catálogo.
type Tabela map[string]RelacaoTipo

// indexa as relações pelo tipo atacante.
func NovaTabela(relacoes []RelacaoTipo) Tabela {
	t := make(Tabela, len(relacoes))
	for _, r := range relacoes {
		t[r.Tipo] = r
	}
	return t
}

// 0, 0.25, 0.5, 1, 2 ou 4.
func (t Tabela) Multiplicador(ataque string, tiposDefensor []string) float64 {
	rel, ok := t[ataque]
	if !ok {
		return 1
	}
	m := 1.0
	for _, d := range tiposDefensor {
		switch {
		case slices.Contains(rel.NenhumPara, d):
			m *= 0
		case slices.Contains(rel.DobroPara, d):
			m *= 2
		case slices.Contains(rel.MetadePara, d):
			m *= 0.5
		}
	}
	return m
}

var nomesTipos = map[string]string{
	"normal": "Normal", "fire": "Fogo", "water": "Água", "electric": "Elétrico",
	"grass": "Planta", "ice": "Gelo", "fighting": "Lutador", "poison": "Venenoso",
	"ground": "Terra", "flying": "Voador", "psychic": "Psíquico", "bug": "Inseto",
	"rock": "Pedra", "ghost": "Fantasma", "dragon": "Dragão", "dark": "Sombrio",
	"steel": "Aço", "fairy": "Fada",
}

// tipo da PokeAPI em português.
func NomeTipo(tipo string) string {
	if n, ok := nomesTipos[tipo]; ok {
		return n
	}
	return tipo
}

// "flutter-mane" -> "Flutter Mane".
func Exibir(nome string) string {
	partes := strings.Split(nome, "-")
	for i, p := range partes {
		if p != "" {
			partes[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(partes, " ")
}
