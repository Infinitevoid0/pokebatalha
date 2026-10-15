package dominio

// Sorteador permite trocar a aleatoriedade por algo previsível nos testes.
type Sorteador interface {
	IntN(n int) int
}

// ChanceTier é a probabilidade (em %) de um pacote sair em cada tier.
type ChanceTier struct {
	Tier        string `json:"tier"`
	Porcentagem int    `json:"porcentagem"`
}

// Chances do gacha, do mais raro ao mais comum. A soma precisa dar 100.
var Chances = []ChanceTier{
	{Tier: "lendario", Porcentagem: 2},
	{Tier: "epico", Porcentagem: 8},
	{Tier: "raro", Porcentagem: 20},
	{Tier: "comum", Porcentagem: 70},
}

// Pity: depois de tantos pacotes sem, o próximo é garantido.
const (
	PityEpico    = 10 // épico ou melhor
	PityLendario = 50
)

// Pity guarda há quantos pacotes o jogador não tira cada raridade.
type Pity struct {
	SemEpico    int `json:"pacotes_sem_epico"`
	SemLendario int `json:"pacotes_sem_lendario"`
}

// InfoPity é o pity como o jogador vê: quanto falta para cada garantia.
type InfoPity struct {
	Pity
	EpicoGarantidoEm    int `json:"epico_garantido_em"`
	LendarioGarantidoEm int `json:"lendario_garantido_em"`
}

// Info calcula quantos pacotes faltam para cada garantia.
func (p Pity) Info() InfoPity {
	return InfoPity{
		Pity:                p,
		EpicoGarantidoEm:    PityEpico - p.SemEpico,
		LendarioGarantidoEm: PityLendario - p.SemLendario,
	}
}

// SortearTier rola um número de 0 a 99 e vê em qual faixa ele cai.
// Ex: 0-1 lendário, 2-9 épico, 10-29 raro, 30-99 comum.
func SortearTier(s Sorteador) string {
	n := s.IntN(100)
	acumulado := 0
	for _, c := range Chances {
		acumulado += c.Porcentagem
		if n < acumulado {
			return c.Tier
		}
	}
	return "comum"
}

// ResultadoSorteio é o tier sorteado e como fica o pity depois dele.
type ResultadoSorteio struct {
	Tier      string
	Garantido bool // saiu por causa do pity
	Pity      Pity
}

// SortearComPity aplica as garantias antes do sorteio normal.
// No pity de épico, o resultado é épico ou lendário na mesma proporção das
// chances normais (2 para 8), para o lendário continuar possível.
func SortearComPity(s Sorteador, atual Pity) ResultadoSorteio {
	proximo := Pity{SemEpico: atual.SemEpico + 1, SemLendario: atual.SemLendario + 1}

	var r ResultadoSorteio
	switch {
	case proximo.SemLendario >= PityLendario:
		r = ResultadoSorteio{Tier: "lendario", Garantido: true}
	case proximo.SemEpico >= PityEpico:
		lend, epic := chance("lendario"), chance("epico")
		tier := "epico"
		if s.IntN(lend+epic) < lend {
			tier = "lendario"
		}
		r = ResultadoSorteio{Tier: tier, Garantido: true}
	default:
		r = ResultadoSorteio{Tier: SortearTier(s)}
	}

	switch r.Tier {
	case "lendario":
		proximo = Pity{} // lendário também conta como "épico ou melhor"
	case "epico":
		proximo.SemEpico = 0
	}
	r.Pity = proximo
	return r
}

func chance(tier string) int {
	for _, c := range Chances {
		if c.Tier == tier {
			return c.Porcentagem
		}
	}
	return 0
}
