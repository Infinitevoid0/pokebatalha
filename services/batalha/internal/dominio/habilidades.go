package dominio

import "slices"

// habilidade implementada no jogo.
type InfoHabilidade struct {
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
}

// em ordem de prioridade: com mais de uma, vale a primeira da lista.
var Habilidades = []InfoHabilidade{
	{"intimidate", "Ao entrar, reduz o ataque do oponente em 1 estágio."},
	{"drought", "Ao entrar, deixa o sol forte por 5 turnos (fogo x1.5, água x0.5)."},
	{"drizzle", "Ao entrar, faz chover por 5 turnos (água x1.5, fogo x0.5)."},
	{"orichalcum-pulse", "Ao entrar, deixa o sol forte; com sol, ataque x1.33."},
	{"hadron-engine", "Ataque x1.33."},
	{"protosynthesis", "Energia Booster: o maior atributo ganha x1.3 (velocidade x1.5)."},
	{"quark-drive", "Energia Booster: o maior atributo ganha x1.3 (velocidade x1.5)."},
	{"sword-of-ruin", "Reduz a defesa do oponente em 25%."},
	{"beads-of-ruin", "Reduz a defesa do oponente em 25%."},
	{"tablets-of-ruin", "Reduz o ataque do oponente em 25%."},
	{"vessel-of-ruin", "Reduz o ataque do oponente em 25%."},
	{"huge-power", "Dobra o ataque."},
	{"pure-power", "Dobra o ataque."},
	{"intrepid-sword", "Ao entrar, aumenta o próprio ataque em 1 estágio."},
	{"multiscale", "Com HP cheio, recebe metade do dano."},
	{"shadow-shield", "Com HP cheio, recebe metade do dano."},
	{"defiant", "Se o oponente reduzir seu ataque, ganha +2 de ataque."},
	{"competitive", "Se o oponente reduzir seu ataque, ganha +2 de ataque."},
	{"speed-boost", "No fim de cada turno, aumenta a velocidade em 1 estágio."},
	{"moxie", "Ao nocautear um oponente, aumenta o ataque em 1 estágio."},
	{"adaptability", "Ataques do próprio tipo causam x1.33."},
	{"technician", "Golpes com poder até 60 causam x1.5."},
	{"levitate", "Imune a ataques do tipo Terra."},
	{"flash-fire", "Imune a ataques do tipo Fogo."},
	{"water-absorb", "Imune a ataques do tipo Água."},
	{"storm-drain", "Imune a ataques do tipo Água."},
	{"volt-absorb", "Imune a ataques do tipo Elétrico."},
	{"lightning-rod", "Imune a ataques do tipo Elétrico."},
	{"motor-drive", "Imune a ataques do tipo Elétrico."},
	{"sap-sipper", "Imune a ataques do tipo Planta."},
	{"thick-fat", "Recebe metade do dano de Fogo e Gelo."},
	{"fur-coat", "Recebe metade do dano de Ataque Físico e Fake Out."},
	{"sturdy", "Com HP cheio, sobrevive a qualquer golpe com 1 de HP."},
	{"clear-body", "Imune a Intimidate."},
	{"white-smoke", "Imune a Intimidate."},
	{"full-metal-body", "Imune a Intimidate."},
	{"inner-focus", "Imune a Intimidate e não recua com Fake Out."},
}

// habilidade -> tipo de ataque que ela anula.
var imunidadePorHabilidade = map[string]string{
	"levitate":      "ground",
	"flash-fire":    "fire",
	"water-absorb":  "water",
	"storm-drain":   "water",
	"volt-absorb":   "electric",
	"lightning-rod": "electric",
	"motor-drive":   "electric",
	"sap-sipper":    "grass",
}

var imunesAIntimidate = []string{"clear-body", "white-smoke", "full-metal-body", "inner-focus"}

// a habilidade implementada de maior prioridade; "" se nenhuma.
func EscolherHabilidade(doPokemon []string) string {
	for _, h := range Habilidades {
		if slices.Contains(doPokemon, h.Nome) {
			return h.Nome
		}
	}
	return ""
}

// texto explicativo da habilidade.
func DescricaoHabilidade(nome string) string {
	for _, h := range Habilidades {
		if h.Nome == nome {
			return h.Descricao
		}
	}
	return ""
}
