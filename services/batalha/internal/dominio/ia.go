package dominio

import "math"

// A "IA" do rival estima o dano de cada golpe com a mesma fórmula da batalha
// (usando o fator aleatório médio) e decide com base na vantagem do confronto:
// quanto do HP do oponente eu tiro por turno menos quanto ele tira do meu.

const (
	chanceProtegerDoFakeOut = 60 // %: rival usa Protect quando você ameaça Fake Out
	chanceTrocar            = 70 // %: rival troca quando o confronto está ruim
	chanceProtegerDoNocaute = 25 // %: rival se protege quando vai ser nocauteado antes de agir
	limiteConfrontoRuim     = -0.3
	ganhoMinimoParaTrocar   = 0.35
	custoDeTrocar           = 0.15 // quem entra toma o golpe do turno
)

// sorteadorMedio dá sempre o meio do fator aleatório (0.925).
type sorteadorMedio struct{}

func (sorteadorMedio) IntN(int) int     { return 0 }
func (sorteadorMedio) Float64() float64 { return 0.5 }

func danoEsperado(a Acao, atk, alvo *Lutador, clima Clima, t TabelaTipos) float64 {
	r := calcularDano(golpeDe(a, atk), atk, alvo, clima, t, sorteadorMedio{})
	return float64(r.dano)
}

// melhorGolpe devolve o golpe de ataque com maior dano esperado.
func melhorGolpe(atk, alvo *Lutador, clima Clima, t TabelaTipos) (Acao, float64) {
	melhor, maiorDano := AcaoFisico, -1.0
	for _, a := range golpesDisponiveis(atk) {
		if d := danoEsperado(a, atk, alvo, clima, t); d > maiorDano {
			melhor, maiorDano = a, d
		}
	}
	return melhor, maiorDano
}

// vantagem vai de -1 (eu sou atropelado) a +1 (eu atropelo).
func vantagem(eu, op *Lutador, clima Clima, t TabelaTipos) float64 {
	if eu.Desmaiado() || op.Desmaiado() {
		return -1
	}
	_, meu := melhorGolpe(eu, op, clima, t)
	_, dele := melhorGolpe(op, eu, clima, t)
	v := math.Min(meu/float64(op.HP), 1) - math.Min(dele/float64(eu.HP), 1)
	switch ve, vo := velocidadeEfetiva(eu), velocidadeEfetiva(op); {
	case ve > vo:
		v += 0.1
	case ve < vo:
		v -= 0.1
	}
	return v
}

func fakeOutAfeta(alvo *Lutador, t TabelaTipos) bool {
	mult, _ := efetividade(golpeDe(AcaoFakeOut, alvo), alvo, t)
	return mult > 0 && alvo.Habilidade != "inner-focus"
}

// EscolherJogadaCPU decide a ação do rival no turno.
func EscolherJogadaCPU(b *Batalha, t TabelaTipos, s Sorteador) Escolha {
	eu, op := b.CPU.atual(), b.Jogador.atual()

	// 1. Fake Out no primeiro turno em campo, se funcionar no alvo.
	if eu.PodeUsarFakeOut() && fakeOutAfeta(op, t) {
		return Escolha{Acao: AcaoFakeOut}
	}

	// 2. Protect quando o jogador ameaça Fake Out.
	if op.PodeUsarFakeOut() && !eu.UsouProtect && fakeOutAfeta(eu, t) && s.IntN(100) < chanceProtegerDoFakeOut {
		return Escolha{Acao: AcaoProtect}
	}

	// 3. Troca se o confronto está ruim e alguém do banco se sai bem melhor.
	atual := vantagem(eu, op, b.Clima, t)
	if atual < limiteConfrontoRuim {
		melhorIdx, melhor := -1, atual
		for _, i := range b.CPU.TrocasDisponiveis() {
			if v := vantagem(&b.CPU.Time[i], op, b.Clima, t) - custoDeTrocar; v > melhor {
				melhorIdx, melhor = i, v
			}
		}
		if melhorIdx >= 0 && melhor > atual+ganhoMinimoParaTrocar && s.IntN(100) < chanceTrocar {
			return Escolha{Acao: AcaoTrocar, Alvo: melhorIdx}
		}
	}

	// 4. Às vezes se protege quando vai ser nocauteado antes de agir.
	_, danoDele := melhorGolpe(op, eu, b.Clima, t)
	if !eu.UsouProtect && danoDele >= float64(eu.HP) && velocidadeEfetiva(op) > velocidadeEfetiva(eu) &&
		s.IntN(100) < chanceProtegerDoNocaute {
		return Escolha{Acao: AcaoProtect}
	}

	// 5. Senão, o golpe que mais machuca.
	golpe, _ := melhorGolpe(eu, op, b.Clima, t)
	return Escolha{Acao: golpe}
}

// EscolherSubstitutoCPU escolhe, depois de um desmaio, quem do banco tem o
// melhor confronto contra o Pokémon do jogador.
func EscolherSubstitutoCPU(b *Batalha, t TabelaTipos) int {
	op := b.Jogador.atual()
	melhorIdx, melhor := -1, math.Inf(-1)
	for _, i := range b.CPU.TrocasDisponiveis() {
		if v := vantagem(&b.CPU.Time[i], op, b.Clima, t); v > melhor {
			melhorIdx, melhor = i, v
		}
	}
	return melhorIdx
}
