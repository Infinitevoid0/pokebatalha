package dominio

import "testing"

// tabelaReal tem a tabela de tipos completa, para os números baterem com o jogo.
func tabelaReal() Tabela {
	rel := func(tipo string, dobro, metade, nenhum []string) RelacaoTipo {
		return RelacaoTipo{Tipo: tipo, DobroPara: dobro, MetadePara: metade, NenhumPara: nenhum}
	}
	l := func(s ...string) []string { return s }
	t := Tabela{}
	for _, r := range []RelacaoTipo{
		rel("normal", nil, l("rock", "steel"), l("ghost")),
		rel("fire", l("grass", "ice", "bug", "steel"), l("fire", "water", "rock", "dragon"), nil),
		rel("water", l("fire", "ground", "rock"), l("water", "grass", "dragon"), nil),
		rel("electric", l("water", "flying"), l("electric", "grass", "dragon"), l("ground")),
		rel("grass", l("water", "ground", "rock"), l("fire", "grass", "poison", "flying", "bug", "dragon", "steel"), nil),
		rel("ice", l("grass", "ground", "flying", "dragon"), l("fire", "water", "ice", "steel"), nil),
		rel("fighting", l("normal", "ice", "rock", "dark", "steel"), l("poison", "flying", "psychic", "bug", "fairy"), l("ghost")),
		rel("poison", l("grass", "fairy"), l("poison", "ground", "rock", "ghost"), l("steel")),
		rel("ground", l("fire", "electric", "poison", "rock", "steel"), l("grass", "bug"), l("flying")),
		rel("flying", l("grass", "fighting", "bug"), l("electric", "rock", "steel"), nil),
		rel("psychic", l("fighting", "poison"), l("psychic", "steel"), l("dark")),
		rel("bug", l("grass", "psychic", "dark"), l("fire", "fighting", "poison", "flying", "ghost", "steel", "fairy"), nil),
		rel("rock", l("fire", "ice", "flying", "bug"), l("fighting", "ground", "steel"), nil),
		rel("ghost", l("psychic", "ghost"), l("dark"), l("normal")),
		rel("dragon", l("dragon"), l("steel"), l("fairy")),
		rel("dark", l("psychic", "ghost"), l("fighting", "dark", "fairy"), nil),
		rel("steel", l("ice", "rock", "fairy"), l("fire", "water", "electric", "steel"), nil),
		rel("fairy", l("fighting", "dragon", "dark"), l("fire", "poison", "steel"), nil),
	} {
		t[r.Tipo] = r
	}
	return t
}

var (
	incineroar = Pokemon{Nome: "incineroar", Tipos: []string{"fire", "dark"},
		Habilidades: []string{"blaze", "intimidate"}, AprendeFakeOut: true,
		Status: Status{HP: 95, Ataque: 115, Defesa: 90, AtaqueEspecial: 80, DefesaEspecial: 90, Velocidade: 60}}
	charizard = Pokemon{Nome: "charizard", Tipos: []string{"fire", "flying"},
		Habilidades: []string{"blaze", "solar-power"},
		Status:      Status{HP: 78, Ataque: 84, Defesa: 78, AtaqueEspecial: 109, DefesaEspecial: 85, Velocidade: 100}}
)

func TestCalcularPoder_BateComACalibracao(t *testing.T) {
	// Valores conferidos com o script de calibração sobre os CSVs da PokeAPI.
	p := CalcularPoder(incineroar, tabelaReal())
	esperado := Poder{Total: 465, Stats: 360, Ofensivo: 40, Defensivo: 0, Habilidade: 45, FakeOut: 20, HabilidadeAtiva: "intimidate"}
	if p != esperado {
		t.Fatalf("poder do incineroar:\n esperado %+v\n veio     %+v", esperado, p)
	}
	if c := CalcularPoder(charizard, tabelaReal()); c.Total != 395 {
		t.Fatalf("poder do charizard: esperado 395, veio %d (%+v)", c.Total, c)
	}
}

func TestMecanicaPesaMaisQueStatsBrutos(t *testing.T) {
	// O exemplo do jogo: o Charizard tem stats maiores (368 x 360),
	// mas Intimidate, Fake Out e a tipagem colocam o Incineroar um tier acima.
	inc, cha := CalcularPoder(incineroar, tabelaReal()), CalcularPoder(charizard, tabelaReal())
	if cha.Stats <= inc.Stats {
		t.Fatal("o exemplo pressupõe charizard com stats maiores")
	}
	if TierPorPoder(inc.Total) != TierLendario || TierPorPoder(cha.Total) != TierEpico {
		t.Errorf("esperava incineroar lendário e charizard épico, veio %s e %s",
			TierPorPoder(inc.Total), TierPorPoder(cha.Total))
	}
}

func TestTipagemDefensiva_ImunidadesValem(t *testing.T) {
	tabela := tabelaReal()
	// Fantasma/Fada é imune a Normal, Lutador e Dragão; Gelo puro é fraco a muita coisa.
	if fantasmaFada, gelo := tipagemDefensiva([]string{"ghost", "fairy"}, tabela), tipagemDefensiva([]string{"ice"}, tabela); fantasmaFada <= gelo {
		t.Errorf("ghost/fairy (%d) deveria defender melhor que ice (%d)", fantasmaFada, gelo)
	}
}

func TestHabilidadeAtiva_UsaAMaisForte(t *testing.T) {
	if h := HabilidadeAtiva([]string{"intimidate", "moxie"}); h != "intimidate" {
		t.Errorf("gyarados deveria usar intimidate, veio %q", h)
	}
	if h := HabilidadeAtiva([]string{"blaze", "solar-power"}); h != "" {
		t.Errorf("sem habilidade implementada deveria vir vazio, veio %q", h)
	}
}

func TestTierPorPoder_Cortes(t *testing.T) {
	casos := map[int]Tier{
		CorteRaro - 1: TierComum, CorteRaro: TierRaro,
		CorteEpico - 1: TierRaro, CorteEpico: TierEpico,
		CorteLendario - 1: TierEpico, CorteLendario: TierLendario,
	}
	for poder, esperado := range casos {
		if got := TierPorPoder(poder); got != esperado {
			t.Errorf("poder %d: esperava %s, veio %s", poder, esperado, got)
		}
	}
}
