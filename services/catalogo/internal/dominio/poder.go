package dominio

// índice de poder = soma de quatro pesos na mesma escala:
//
//	stats      HP + maior ataque + média das defesas + velocidade
//	tipagem    cobertura ofensiva + resistência defensiva
//	habilidade peso da melhor habilidade implementada
//	fake out   bônus fixo para quem aprende o golpe
//
// cortes calibrados nos 1025 Pokémon: ~42% comum, ~27% raro, ~20% épico, ~10% lendário.
const (
	CorteRaro     = 300
	CorteEpico    = 370
	CorteLendario = 430

	pesoOfensivo  = 120 // cobertura média 1.33x vale +40
	pesoDefensivo = 120 // receber 0.8x em média vale +24
	PesoFakeOut   = 20
)

// peso das habilidades implementadas; as outras valem 0.
var PesosHabilidade = map[string]int{
	"huge-power": 70, "pure-power": 70,
	"intimidate": 45, "orichalcum-pulse": 45,
	"drought": 40, "drizzle": 40, "speed-boost": 40, "fur-coat": 40,
	"hadron-engine": 35, "sword-of-ruin": 35, "beads-of-ruin": 35,
	"intrepid-sword": 35, "multiscale": 35, "shadow-shield": 35,
	"tablets-of-ruin": 30, "vessel-of-ruin": 30,
	"protosynthesis": 30, "quark-drive": 30, "adaptability": 30,
	"moxie": 25, "levitate": 25,
	"defiant": 20, "competitive": 20, "thick-fat": 20,
	"technician": 15, "flash-fire": 15, "water-absorb": 15, "storm-drain": 15,
	"volt-absorb": 15, "lightning-rod": 15, "motor-drive": 15, "sap-sipper": 15, "sturdy": 15,
	"clear-body": 10, "white-smoke": 10, "full-metal-body": 10, "inner-focus": 10,
}

// índice de força com a conta aberta.
type Poder struct {
	Total           int    `json:"total"`
	Stats           int    `json:"stats"`
	Ofensivo        int    `json:"tipagem_ofensiva"`
	Defensivo       int    `json:"tipagem_defensiva"`
	Habilidade      int    `json:"habilidade"`
	FakeOut         int    `json:"fake_out"`
	HabilidadeAtiva string `json:"habilidade_ativa,omitempty"`
}

// efetividade de um tipo atacante contra o defensor.
type Tabela map[string]RelacaoTipo

func (t Tabela) multiplicador(ataque string, defensor []string) float64 {
	rel, ok := t[ataque]
	if !ok {
		return 1
	}
	return rel.Multiplicador(defensor)
}

// habilidade de maior peso (empate: a primeira); "" se nenhuma.
func HabilidadeAtiva(habilidades []string) string {
	melhor, pesoMelhor := "", 0
	for _, h := range habilidades {
		if p := PesosHabilidade[h]; p > pesoMelhor {
			melhor, pesoMelhor = h, p
		}
	}
	return melhor
}

// monta o índice de poder do Pokémon.
func CalcularPoder(p Pokemon, t Tabela) Poder {
	s := p.Status
	stats := s.HP + max(s.Ataque, s.AtaqueEspecial) + (s.Defesa+s.DefesaEspecial)/2 + s.Velocidade

	hab := HabilidadeAtiva(p.Habilidades)
	poder := Poder{
		Stats:           stats,
		Ofensivo:        tipagemOfensiva(p.Tipos, t),
		Defensivo:       tipagemDefensiva(p.Tipos, t),
		Habilidade:      PesosHabilidade[hab],
		HabilidadeAtiva: hab,
	}
	if p.AprendeFakeOut {
		poder.FakeOut = PesoFakeOut
	}
	poder.Total = poder.Stats + poder.Ofensivo + poder.Defensivo + poder.Habilidade + poder.FakeOut
	return poder
}

// melhor multiplicador contra cada um dos 18 tipos; média > 1 = boa cobertura.
func tipagemOfensiva(tipos []string, t Tabela) int {
	if len(tipos) == 0 {
		return 0
	}
	soma := 0.0
	for _, defensor := range TiposExistentes {
		melhor := 0.0
		for _, atk := range tipos {
			melhor = max(melhor, t.multiplicador(atk, []string{defensor}))
		}
		soma += melhor
	}
	media := soma / float64(len(TiposExistentes))
	return arredondar((media - 1) * pesoOfensivo)
}

// dano médio recebido dos 18 tipos; abaixo de 1 ganha pontos, acima perde.
func tipagemDefensiva(tipos []string, t Tabela) int {
	if len(tipos) == 0 {
		return 0
	}
	soma := 0.0
	for _, atk := range TiposExistentes {
		soma += t.multiplicador(atk, tipos)
	}
	media := soma / float64(len(TiposExistentes))
	return arredondar((1 - media) * pesoDefensivo)
}

func arredondar(x float64) int {
	if x < 0 {
		return -int(-x + 0.5)
	}
	return int(x + 0.5)
}

// índice -> tier do gacha.
func TierPorPoder(total int) Tier {
	switch {
	case total >= CorteLendario:
		return TierLendario
	case total >= CorteEpico:
		return TierEpico
	case total >= CorteRaro:
		return TierRaro
	default:
		return TierComum
	}
}

// fórmula exposta pela API.
type Regras struct {
	Formula         string         `json:"formula"`
	Cortes          map[Tier]int   `json:"cortes"`
	PesoFakeOut     int            `json:"peso_fake_out"`
	PesosHabilidade map[string]int `json:"pesos_habilidade"`
}

// fórmula e cortes em vigor.
func RegrasDoPoder() Regras {
	return Regras{
		Formula: "poder = stats (HP + maior ataque + média das defesas + velocidade) " +
			"+ tipagem ofensiva + tipagem defensiva + peso da habilidade + bônus de Fake Out",
		Cortes: map[Tier]int{
			TierComum: 0, TierRaro: CorteRaro, TierEpico: CorteEpico, TierLendario: CorteLendario,
		},
		PesoFakeOut:     PesoFakeOut,
		PesosHabilidade: PesosHabilidade,
	}
}
