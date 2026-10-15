package dominio

import (
	"fmt"
	"math"
	"slices"
)

// Acao é o que o jogador escolhe fazer no turno.
type Acao string

const (
	AcaoFisico  Acao = "fisico"   // poder 50, sem tipo: sempre dano neutro
	AcaoTipo    Acao = "tipo"     // poder 70, do 1º tipo: pode dar 0x a 4x
	AcaoTipo2   Acao = "tipo2"    // poder 70, do 2º tipo (só para quem tem dois)
	AcaoFakeOut Acao = "fake_out" // poder 40, Normal, prioridade +3, faz o alvo recuar
	AcaoProtect Acao = "protect"  // prioridade +4, bloqueia o golpe do rival; não repete seguido
	AcaoTrocar  Acao = "trocar"   // prioridade +6, troca pelo Pokémon do banco indicado em Alvo
)

var acoesValidas = []Acao{AcaoFisico, AcaoTipo, AcaoTipo2, AcaoFakeOut, AcaoProtect, AcaoTrocar}

// Escolha é a ação do turno; Alvo só vale para "trocar" (índice no time).
type Escolha struct {
	Acao Acao
	Alvo int
}

// ParseAcao valida o texto vindo do cliente.
func ParseAcao(s string) (Acao, error) {
	a := Acao(s)
	if !slices.Contains(acoesValidas, a) {
		return "", fmt.Errorf("%w: %q (use fisico, tipo, tipo2, fake_out, protect ou trocar)", ErrAcaoInvalida, s)
	}
	return a, nil
}

// golpesDisponiveis lista os golpes de ataque do lutador (sem Protect e troca).
func golpesDisponiveis(l *Lutador) []Acao {
	golpes := []Acao{AcaoFisico, AcaoTipo}
	if len(l.Tipos) > 1 {
		golpes = append(golpes, AcaoTipo2)
	}
	return golpes
}

type golpe struct {
	nome      string
	poder     int
	tipo      string // "" = sem tipo
	fazRecuar bool
	fisico    bool // afetado por Fur Coat
}

func golpeDe(a Acao, atacante *Lutador) golpe {
	tipoN := func(i int) string {
		if len(atacante.Tipos) > i {
			return atacante.Tipos[i]
		}
		return "normal"
	}
	switch a {
	case AcaoTipo:
		t := tipoN(0)
		return golpe{nome: "Ataque de " + NomeTipo(t), poder: 70, tipo: t}
	case AcaoTipo2:
		t := tipoN(1)
		return golpe{nome: "Ataque de " + NomeTipo(t), poder: 70, tipo: t}
	case AcaoFakeOut:
		return golpe{nome: "Fake Out", poder: 40, tipo: "normal", fazRecuar: true, fisico: true}
	default:
		return golpe{nome: "Ataque Físico", poder: 50, fisico: true}
	}
}

// efetividade combina a tabela de tipos com as imunidades por habilidade.
// imunePor é o nome da habilidade que anulou o golpe (se houver).
func efetividade(g golpe, alvo *Lutador, t TabelaTipos) (mult float64, imunePor string) {
	if g.tipo == "" {
		return 1, ""
	}
	if imunidadePorHabilidade[alvo.Habilidade] == g.tipo {
		return 0, alvo.Habilidade
	}
	return t.Multiplicador(g.tipo, alvo.Tipos), ""
}

// modificadores são os outros multiplicadores de dano: clima e habilidades.
func modificadores(g golpe, atk, alvo *Lutador, clima Clima) float64 {
	m := 1.0

	switch {
	case clima == ClimaSol && g.tipo == "fire", clima == ClimaChuva && g.tipo == "water":
		m *= 1.5
	case clima == ClimaSol && g.tipo == "water", clima == ClimaChuva && g.tipo == "fire":
		m *= 0.5
	}

	switch atk.Habilidade {
	case "adaptability":
		if g.tipo != "" && slices.Contains(atk.Tipos, g.tipo) {
			m *= 1.33
		}
	case "technician":
		if g.poder <= 60 {
			m *= 1.5
		}
	}

	switch alvo.Habilidade {
	case "multiscale", "shadow-shield":
		if alvo.HP == alvo.HPMax {
			m *= 0.5
		}
	case "thick-fat":
		if g.tipo == "fire" || g.tipo == "ice" {
			m *= 0.5
		}
	case "fur-coat":
		if g.fisico {
			m *= 0.5
		}
	}
	return m
}

func ataqueEfetivo(atk, alvo *Lutador, clima Clima) float64 {
	a := float64(atk.Ataque) * multiplicadorEstagio(atk.EstagioAtaque)
	switch atk.Habilidade {
	case "huge-power", "pure-power":
		a *= 2
	case "hadron-engine":
		a *= 1.33
	case "orichalcum-pulse":
		if clima == ClimaSol {
			a *= 1.33
		}
	}
	if atk.atributoImpulsionado() == "ataque" {
		a *= 1.3
	}
	if alvo.Habilidade == "tablets-of-ruin" || alvo.Habilidade == "vessel-of-ruin" {
		a *= 0.75
	}
	return a
}

func defesaEfetiva(alvo, atk *Lutador) float64 {
	d := float64(alvo.Defesa)
	if alvo.atributoImpulsionado() == "defesa" {
		d *= 1.3
	}
	if atk.Habilidade == "sword-of-ruin" || atk.Habilidade == "beads-of-ruin" {
		d *= 0.75
	}
	return d
}

func velocidadeEfetiva(l *Lutador) float64 {
	v := float64(l.Velocidade) * multiplicadorEstagio(l.EstagioVelocidade)
	if l.atributoImpulsionado() == "velocidade" {
		v *= 1.5
	}
	return v
}

type resultadoDano struct {
	dano     int
	mult     float64
	imunePor string
	sturdy   bool
}

// calcularDano usa a fórmula dos jogos no nível 50 (simplificada):
//
//	base = (22 * poder * ataque / defesa) / 50 + 2
//	dano = base * efetividade * modificadores * aleatório(0.85 a 1.0)
func calcularDano(g golpe, atk, alvo *Lutador, clima Clima, t TabelaTipos, s Sorteador) resultadoDano {
	mult, imunePor := efetividade(g, alvo, t)
	if mult == 0 {
		return resultadoDano{mult: 0, imunePor: imunePor}
	}

	base := math.Floor(22*float64(g.poder)*ataqueEfetivo(atk, alvo, clima)/defesaEfetiva(alvo, atk))/50 + 2
	aleatorio := 0.85 + 0.15*s.Float64()
	dano := max(int(base*mult*modificadores(g, atk, alvo, clima)*aleatorio), 1)

	r := resultadoDano{dano: dano, mult: mult}
	if alvo.Habilidade == "sturdy" && alvo.HP == alvo.HPMax && dano >= alvo.HP {
		r.dano = alvo.HP - 1
		r.sturdy = true
	}
	return r
}
