package dominio

// Especie são os dados do Pokémon como vêm do catálogo.
type Especie struct {
	PokemonID       int
	Nome            string
	Tier            string
	Poder           int
	HabilidadeAtiva string // escolhida pelo catálogo; vazia = escolher aqui
	Tipos           []string
	Habilidades     []string
	AprendeFakeOut  bool
	HP              int
	Ataque          int
	Defesa          int
	AtaqueEspecial  int
	DefesaEspecial  int
	Velocidade      int
}

// Lutador é o Pokémon dentro de uma batalha, com HP atual e estágios.
// Tudo é calculado no nível 50, como no VGC.
type Lutador struct {
	PokemonID           int      `json:"pokemon_id"`
	Nome                string   `json:"nome"`
	Tier                string   `json:"tier"`
	Poder               int      `json:"poder"`
	Tipos               []string `json:"tipos"`
	Habilidade          string   `json:"habilidade,omitempty"`
	DescricaoHabilidade string   `json:"descricao_habilidade,omitempty"`
	AprendeFakeOut      bool     `json:"aprende_fake_out"`
	HPMax               int      `json:"hp_max"`
	HP                  int      `json:"hp"`
	Ataque              int      `json:"ataque"`
	Defesa              int      `json:"defesa"`
	Velocidade          int      `json:"velocidade"`
	EstagioAtaque       int      `json:"estagio_ataque"`
	EstagioVelocidade   int      `json:"estagio_velocidade"`
	TurnosEmCampo       int      `json:"turnos_em_campo"`
	UsouProtect         bool     `json:"usou_protect"` // a última ação foi Protect

	// Valem só durante o turno, não são salvos.
	protegido    bool // usou Protect neste turno
	recemChegado bool // entrou por troca neste turno (não conta o turno em campo)
}

const (
	estagioMinimo = -6
	estagioMaximo = 6
)

// NovoLutador converte a espécie em lutador de nível 50.
// Simplificação: o jogo só tem um "ataque" e uma "defesa", então usamos o
// maior entre ataque físico/especial e a média das defesas. Assim atacantes
// especiais (ex: Flutter Mane) não ficam inúteis.
func NovoLutador(e Especie) Lutador {
	ataque := max(e.Ataque, e.AtaqueEspecial)
	defesa := (e.Defesa + e.DefesaEspecial) / 2
	hab := e.HabilidadeAtiva
	if DescricaoHabilidade(hab) == "" {
		hab = EscolherHabilidade(e.Habilidades)
	}

	return Lutador{
		PokemonID:           e.PokemonID,
		Nome:                e.Nome,
		Tier:                e.Tier,
		Poder:               e.Poder,
		Tipos:               e.Tipos,
		Habilidade:          hab,
		DescricaoHabilidade: DescricaoHabilidade(hab),
		AprendeFakeOut:      e.AprendeFakeOut,
		HPMax:               e.HP + 60, // fórmula de HP no nível 50, sem IV/EV
		HP:                  e.HP + 60,
		Ataque:              ataque + 5, // fórmula dos outros atributos no nível 50
		Defesa:              defesa + 5,
		Velocidade:          e.Velocidade + 5,
	}
}

// Desmaiado indica que o HP chegou a zero.
func (l *Lutador) Desmaiado() bool { return l.HP <= 0 }

// PodeUsarFakeOut: só funciona no primeiro turno em campo, como no jogo.
func (l *Lutador) PodeUsarFakeOut() bool { return l.AprendeFakeOut && l.TurnosEmCampo == 0 }

// mudarEstagio aplica delta respeitando os limites -6..+6 e devolve
// quanto realmente mudou (0 se já estava no limite).
func mudarEstagio(estagio *int, delta int) int {
	novo := min(max(*estagio+delta, estagioMinimo), estagioMaximo)
	aplicado := novo - *estagio
	*estagio = novo
	return aplicado
}

// multiplicadorEstagio: +1 = x1.5, +2 = x2, -1 = x0.67, -2 = x0.5...
func multiplicadorEstagio(e int) float64 {
	if e >= 0 {
		return float64(2+e) / 2
	}
	return 2 / float64(2-e)
}

// atributoImpulsionado é o atributo que Protosynthesis/Quark Drive melhora.
func (l *Lutador) atributoImpulsionado() string {
	if l.Habilidade != "protosynthesis" && l.Habilidade != "quark-drive" {
		return ""
	}
	switch {
	case l.Velocidade > l.Ataque && l.Velocidade > l.Defesa:
		return "velocidade"
	case l.Defesa > l.Ataque:
		return "defesa"
	default:
		return "ataque"
	}
}
