package dominio

import (
	"errors"
	"strings"
	"testing"
)

// sorteadorFixo: dano sempre máximo (Float64 = 1) e empates para o primeiro.
// Com IntN = 0, toda decisão aleatória da IA "acontece" (ex: Protect).
type sorteadorFixo struct{}

func (sorteadorFixo) IntN(int) int     { return 0 }
func (sorteadorFixo) Float64() float64 { return 1 }

// semSorte: dano máximo, mas a IA nunca toma as decisões aleatórias.
type semSorte struct{}

func (semSorte) IntN(n int) int   { return n - 1 }
func (semSorte) Float64() float64 { return 1 }

// tabelaTeste tem só as relações usadas nos testes.
var tabelaTeste = NovaTabela([]RelacaoTipo{
	{Tipo: "fire", DobroPara: []string{"grass", "steel"}, MetadePara: []string{"water", "fire"}},
	{Tipo: "water", DobroPara: []string{"fire", "ground"}, MetadePara: []string{"grass", "water"}},
	{Tipo: "grass", DobroPara: []string{"water", "ground"}, MetadePara: []string{"fire", "grass"}},
	{Tipo: "normal", NenhumPara: []string{"ghost"}},
	{Tipo: "ground", DobroPara: []string{"fire", "electric"}},
	{Tipo: "ghost", DobroPara: []string{"ghost", "psychic"}},
})

func lutador(nome string, tipos []string, hab string, hp, atk, def, vel int) Lutador {
	return NovoLutador(Especie{
		Nome: nome, Tipos: tipos, Habilidades: []string{hab},
		HP: hp, Ataque: atk, Defesa: def, DefesaEspecial: def, Velocidade: vel,
	})
}

func logContem(b *Batalha, trecho string) bool {
	for _, l := range b.Log {
		if strings.Contains(l, trecho) {
			return true
		}
	}
	return false
}

func TestNovoLutador_StatusNoNivel50(t *testing.T) {
	l := NovoLutador(Especie{Nome: "flutter-mane", HP: 55, Ataque: 55, AtaqueEspecial: 135,
		Defesa: 55, DefesaEspecial: 135, Velocidade: 135, Habilidades: []string{"protosynthesis"}})

	if l.HPMax != 115 {
		t.Errorf("HP no nível 50 deveria ser 55+60=115, veio %d", l.HPMax)
	}
	if l.Ataque != 140 {
		t.Errorf("deveria usar o maior ataque (especial 135)+5 = 140, veio %d", l.Ataque)
	}
	if l.Habilidade != "protosynthesis" {
		t.Errorf("habilidade errada: %q", l.Habilidade)
	}
}

func TestEscolherHabilidade_PrioridadeCompetitiva(t *testing.T) {
	// Torkoal tem white-smoke, drought e shell-armor: no VGC ele é usado pelo Drought.
	if h := EscolherHabilidade([]string{"white-smoke", "drought", "shell-armor"}); h != "drought" {
		t.Errorf("esperava drought, veio %q", h)
	}
	if h := EscolherHabilidade([]string{"blaze"}); h != "" {
		t.Errorf("habilidade não implementada deveria virar vazio, veio %q", h)
	}
}

func TestCalcularDano_SuperEfetivoDobraODano(t *testing.T) {
	atk := lutador("charmander", []string{"fire"}, "", 50, 60, 50, 60)
	planta := lutador("bulbasaur", []string{"grass"}, "", 50, 50, 60, 45)
	normal := lutador("rattata", []string{"normal"}, "", 50, 50, 60, 45)

	g := golpeDe(AcaoTipo, &atk)
	contraPlanta := calcularDano(g, &atk, &planta, SemClima, tabelaTeste, sorteadorFixo{})
	contraNormal := calcularDano(g, &atk, &normal, SemClima, tabelaTeste, sorteadorFixo{})

	if contraPlanta.mult != 2 || contraNormal.mult != 1 {
		t.Fatalf("multiplicadores errados: %v e %v", contraPlanta.mult, contraNormal.mult)
	}
	// Pode variar 1 ponto por causa do arredondamento.
	if diff := contraPlanta.dano - 2*contraNormal.dano; diff < -1 || diff > 1 {
		t.Errorf("super efetivo deveria causar o dobro: %d vs %d", contraPlanta.dano, contraNormal.dano)
	}
}

func TestCalcularDano_ClimaAfetaFogoEAgua(t *testing.T) {
	atk := lutador("charmander", []string{"fire"}, "", 50, 60, 50, 60)
	alvo := lutador("rattata", []string{"normal"}, "", 200, 50, 60, 45)
	g := golpeDe(AcaoTipo, &atk)

	normal := calcularDano(g, &atk, &alvo, SemClima, tabelaTeste, sorteadorFixo{}).dano
	sol := calcularDano(g, &atk, &alvo, ClimaSol, tabelaTeste, sorteadorFixo{}).dano
	chuva := calcularDano(g, &atk, &alvo, ClimaChuva, tabelaTeste, sorteadorFixo{}).dano

	if !(sol > normal && normal > chuva) {
		t.Errorf("fogo deveria ser mais forte no sol e mais fraco na chuva: sol=%d normal=%d chuva=%d", sol, normal, chuva)
	}
}

func TestLevitateAnulaTerra(t *testing.T) {
	atk := lutador("sandshrew", []string{"ground"}, "", 50, 75, 85, 40)
	alvo := lutador("gengar", []string{"ghost"}, "levitate", 60, 65, 75, 110)
	r := calcularDano(golpeDe(AcaoTipo, &atk), &atk, &alvo, SemClima, tabelaTeste, sorteadorFixo{})
	if r.dano != 0 || r.imunePor != "levitate" {
		t.Fatalf("levitate deveria anular terra, veio %+v", r)
	}
}

func TestMultiscaleSoFuncionaComHPCheio(t *testing.T) {
	atk := lutador("rattata", []string{"normal"}, "", 50, 80, 50, 70)
	alvo := lutador("dragonite", []string{"dragon"}, "multiscale", 91, 134, 95, 80)
	g := golpeDe(AcaoFisico, &atk)

	cheio := calcularDano(g, &atk, &alvo, SemClima, tabelaTeste, sorteadorFixo{}).dano
	alvo.HP--
	ferido := calcularDano(g, &atk, &alvo, SemClima, tabelaTeste, sorteadorFixo{}).dano
	if cheio >= ferido {
		t.Errorf("com HP cheio deveria tomar menos dano: cheio=%d ferido=%d", cheio, ferido)
	}
}

func TestSturdySobreviveComUmDeHP(t *testing.T) {
	atk := lutador("groudon", []string{"ground"}, "", 100, 150, 140, 90)
	alvo := lutador("geodude", []string{"rock"}, "sturdy", 1, 80, 1, 20) // defesa mínima: o golpe seria letal
	b := &Batalha{Status: EmAndamento, Jogador: Lado{Time: []Lutador{atk}}, CPU: Lado{Time: []Lutador{alvo}}}

	b.executarGolpe(b.seu(), b.rival(), AcaoFisico, tabelaTeste, sorteadorFixo{})
	if b.CPU.atual().HP != 1 {
		t.Fatalf("sturdy deveria deixar com 1 HP, ficou %d", b.CPU.atual().HP)
	}
}

func TestIntimidateEDefiant(t *testing.T) {
	incineroar := lutador("incineroar", []string{"fire", "dark"}, "intimidate", 95, 115, 90, 60)
	kingambit := lutador("kingambit", []string{"dark", "steel"}, "defiant", 100, 135, 120, 50)
	charizard := lutador("charizard", []string{"fire", "flying"}, "", 78, 109, 85, 100)

	b, _ := NovaBatalha(1, []Lutador{incineroar}, []Lutador{charizard}, sorteadorFixo{})
	if got := b.CPU.atual().EstagioAtaque; got != -1 {
		t.Errorf("Intimidate deveria deixar o charizard com -1, veio %d", got)
	}

	b, _ = NovaBatalha(1, []Lutador{incineroar}, []Lutador{kingambit}, sorteadorFixo{})
	if got := b.CPU.atual().EstagioAtaque; got != 1 {
		t.Errorf("Defiant: -1 do intimidate +2 = +1, veio %d", got)
	}
	if !logContem(b, "furioso") {
		t.Errorf("log deveria mencionar o Defiant: %v", b.Log)
	}
}

func TestDroughtLigaOSol(t *testing.T) {
	torkoal := lutador("torkoal", []string{"fire"}, "drought", 70, 85, 140, 20)
	alvo := lutador("rattata", []string{"normal"}, "", 30, 56, 35, 72)
	b, _ := NovaBatalha(1, []Lutador{torkoal}, []Lutador{alvo}, sorteadorFixo{})
	if b.Clima != ClimaSol || b.TurnosClima != duracaoClima {
		t.Fatalf("drought deveria ligar o sol por %d turnos, veio %q/%d", duracaoClima, b.Clima, b.TurnosClima)
	}
}

func TestFakeOut_PrioridadeRecuoESoPrimeiroTurno(t *testing.T) {
	// Incineroar é lento, mas Fake Out tem prioridade: age antes e faz o rival recuar.
	inc := lutador("incineroar", []string{"fire", "dark"}, "", 95, 115, 90, 60)
	inc.AprendeFakeOut = true
	rapido := lutador("jolteon", []string{"electric"}, "", 200, 65, 95, 130)
	b, _ := NovaBatalha(1, []Lutador{inc}, []Lutador{rapido}, sorteadorFixo{})

	if err := b.ExecutarTurno(Escolha{Acao: AcaoFakeOut}, tabelaTeste, semSorte{}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.HasPrefix(b.Log[0], "Seu Incineroar usou Fake Out") {
		t.Errorf("Fake Out deveria agir primeiro, log: %v", b.Log)
	}
	if !logContem(b, "recuou") {
		t.Errorf("o rival deveria recuar, log: %v", b.Log)
	}
	if b.Jogador.atual().HP != b.Jogador.atual().HPMax {
		t.Error("o rival recuou, então o incineroar não deveria tomar dano")
	}

	err := b.ExecutarTurno(Escolha{Acao: AcaoFakeOut}, tabelaTeste, semSorte{})
	if !errors.Is(err, ErrAcaoInvalida) {
		t.Fatalf("Fake Out no segundo turno deveria ser inválido, veio %v", err)
	}
}

func TestFakeOutNaoAfetaFantasma(t *testing.T) {
	inc := lutador("incineroar", []string{"fire", "dark"}, "", 95, 115, 90, 60)
	inc.AprendeFakeOut = true
	fantasma := lutador("gengar", []string{"ghost"}, "", 60, 65, 75, 110)
	b, _ := NovaBatalha(1, []Lutador{inc}, []Lutador{fantasma}, sorteadorFixo{})

	_ = b.ExecutarTurno(Escolha{Acao: AcaoFakeOut}, tabelaTeste, sorteadorFixo{})
	if !logContem(b, "Não afeta") || logContem(b, "recuou") {
		t.Errorf("Fake Out não deveria afetar fantasma nem fazer recuar: %v", b.Log)
	}
}

func TestMaisRapidoAtacaPrimeiro(t *testing.T) {
	lento := lutador("snorlax", []string{"normal"}, "", 160, 110, 65, 30)
	rapido := lutador("jolteon", []string{"electric"}, "", 65, 65, 95, 130)
	b, _ := NovaBatalha(1, []Lutador{lento}, []Lutador{rapido}, sorteadorFixo{})

	_ = b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, sorteadorFixo{})
	if !strings.HasPrefix(b.Log[0], "Jolteon do rival usou") {
		t.Errorf("o jolteon deveria agir primeiro: %v", b.Log)
	}
}

func TestSubstituicaoEFimDaBatalha(t *testing.T) {
	forte := lutador("groudon", []string{"ground"}, "", 100, 150, 140, 90)
	fraco1 := lutador("magikarp", []string{"water"}, "", 1, 10, 1, 1)
	fraco2 := lutador("feebas", []string{"water"}, "", 1, 10, 1, 1)
	b, _ := NovaBatalha(1, []Lutador{forte}, []Lutador{fraco1, fraco2}, semSorte{})

	_ = b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, semSorte{})
	if b.CPU.Ativo != 1 || b.Status != EmAndamento {
		t.Fatalf("o rival deveria mandar o segundo pokémon (ativo=%d, status=%s)", b.CPU.Ativo, b.Status)
	}
	_ = b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, semSorte{})
	if b.Status != Vitoria {
		t.Fatalf("esperava vitória, veio %s", b.Status)
	}
	if err := b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, semSorte{}); !errors.Is(err, ErrBatalhaEncerrada) {
		t.Fatalf("jogar depois do fim deveria dar ErrBatalhaEncerrada, veio %v", err)
	}
}

func TestCPUEscolheGolpeSuperEfetivo(t *testing.T) {
	cpu := lutador("squirtle", []string{"water"}, "", 44, 50, 65, 43)
	alvoFogo := lutador("charmander", []string{"fire"}, "", 39, 60, 50, 65)
	alvoPlanta := lutador("bulbasaur", []string{"grass"}, "", 45, 49, 65, 45)

	if a, _ := melhorGolpe(&cpu, &alvoFogo, SemClima, tabelaTeste); a != AcaoTipo {
		t.Errorf("contra fogo deveria usar água, usou %s", a)
	}
	if a, _ := melhorGolpe(&cpu, &alvoPlanta, SemClima, tabelaTeste); a != AcaoFisico {
		t.Errorf("contra planta (resiste água) deveria usar físico, usou %s", a)
	}
}

func TestEstagios(t *testing.T) {
	if multiplicadorEstagio(1) != 1.5 || multiplicadorEstagio(-2) != 0.5 {
		t.Error("multiplicadores de estágio errados")
	}
	e := 5
	if aplicado := mudarEstagio(&e, 3); aplicado != 1 || e != 6 {
		t.Errorf("estágio deveria travar em +6 (aplicado=%d, e=%d)", aplicado, e)
	}
}

func TestProtect_BloqueiaENaoRepeteSeguido(t *testing.T) {
	eu := lutador("snorlax", []string{"normal"}, "", 160, 110, 65, 30)
	rival := lutador("jolteon", []string{"electric"}, "", 65, 65, 95, 130)
	b, _ := NovaBatalha(1, []Lutador{eu}, []Lutador{rival}, semSorte{})

	if err := b.ExecutarTurno(Escolha{Acao: AcaoProtect}, tabelaTeste, semSorte{}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if b.Jogador.atual().HP != b.Jogador.atual().HPMax || !logContem(b, "se protegeu do golpe") {
		t.Fatalf("Protect deveria bloquear o golpe do rival: %v", b.Log)
	}
	err := b.ExecutarTurno(Escolha{Acao: AcaoProtect}, tabelaTeste, semSorte{})
	if !errors.Is(err, ErrAcaoInvalida) {
		t.Fatalf("Protect duas vezes seguidas deveria ser inválido, veio %v", err)
	}
	// Depois de outra ação, volta a poder.
	_ = b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, semSorte{})
	if err := b.ExecutarTurno(Escolha{Acao: AcaoProtect}, tabelaTeste, semSorte{}); err != nil {
		t.Fatalf("Protect deveria voltar depois de um ataque: %v", err)
	}
}

func TestTroca_LimpaIntimidateEReativaHabilidade(t *testing.T) {
	charizard := lutador("charizard", []string{"fire", "flying"}, "", 78, 109, 85, 100)
	incineroar := lutador("incineroar", []string{"fire", "dark"}, "intimidate", 95, 115, 90, 60)
	incineroar.AprendeFakeOut = true
	rival := lutador("gyarados", []string{"water", "flying"}, "intimidate", 95, 125, 79, 81)
	b, _ := NovaBatalha(1, []Lutador{charizard, incineroar}, []Lutador{rival}, semSorte{})

	if b.Jogador.atual().EstagioAtaque != -1 {
		t.Fatal("o charizard deveria começar intimidado")
	}
	if err := b.ExecutarTurno(Escolha{Acao: AcaoTrocar, Alvo: 1}, tabelaTeste, semSorte{}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if b.Jogador.Time[0].EstagioAtaque != 0 {
		t.Error("quem sai deveria perder o -1 do Intimidate")
	}
	if !strings.HasPrefix(b.Log[0], "Volte, Charizard! Vai, Incineroar!") {
		t.Errorf("a troca deveria vir antes de tudo: %v", b.Log)
	}
	if b.CPU.atual().EstagioAtaque != -1 {
		t.Errorf("o Intimidate do incineroar deveria baixar o ataque do rival, veio %d", b.CPU.atual().EstagioAtaque)
	}
	if !b.Jogador.atual().PodeUsarFakeOut() {
		t.Error("quem entra por troca pode usar Fake Out no próximo turno")
	}
}

func TestTrocaForcada_JogadorEscolheQuemEntra(t *testing.T) {
	fraco := lutador("magikarp", []string{"water"}, "", 1, 10, 1, 1)
	b1 := lutador("pikachu", []string{"electric"}, "", 35, 55, 40, 90)
	b2 := lutador("snorlax", []string{"normal"}, "", 160, 110, 65, 30)
	rival := lutador("groudon", []string{"ground"}, "", 100, 150, 140, 90)
	b, _ := NovaBatalha(1, []Lutador{fraco, b1, b2}, []Lutador{rival}, semSorte{})

	_ = b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, semSorte{})
	if !b.PrecisaTrocar {
		t.Fatalf("com dois no banco, o jogador deveria escolher quem entra: %v", b.Log)
	}
	if acoes := b.AcoesDoJogador(); len(acoes) != 1 || acoes[0] != AcaoTrocar {
		t.Fatalf("só trocar deveria estar disponível, veio %v", acoes)
	}
	if err := b.ExecutarTurno(Escolha{Acao: AcaoFisico}, tabelaTeste, semSorte{}); !errors.Is(err, ErrAcaoInvalida) {
		t.Fatalf("atacar sem trocar deveria ser inválido, veio %v", err)
	}
	turno := b.Turno
	if err := b.ExecutarTurno(Escolha{Acao: AcaoTrocar, Alvo: 2}, tabelaTeste, semSorte{}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if b.Jogador.Ativo != 2 || b.PrecisaTrocar || b.Turno != turno {
		t.Fatalf("a troca forçada não gasta turno (ativo=%d precisa=%v turno=%d)", b.Jogador.Ativo, b.PrecisaTrocar, b.Turno)
	}
	if b.Jogador.Time[2].HP != b.Jogador.Time[2].HPMax {
		t.Fatal("o rival não pode atacar durante a troca forçada")
	}
}

func TestSegundoGolpeDeTipo(t *testing.T) {
	duplo := lutador("gyarados", []string{"water", "flying"}, "", 95, 125, 79, 81)
	if acoes := golpesDisponiveis(&duplo); len(acoes) != 3 || acoes[2] != AcaoTipo2 {
		t.Fatalf("pokémon de dois tipos deveria ter tipo2, veio %v", acoes)
	}
	if g := golpeDe(AcaoTipo2, &duplo); g.tipo != "flying" {
		t.Errorf("tipo2 do gyarados deveria ser flying, veio %s", g.tipo)
	}
	simples := lutador("pikachu", []string{"electric"}, "", 35, 55, 40, 90)
	if acoes := golpesDisponiveis(&simples); len(acoes) != 2 {
		t.Errorf("pokémon de um tipo não tem tipo2, veio %v", acoes)
	}
}

func TestIA_TrocaQuandoOConfrontoEstaRuim(t *testing.T) {
	// O rival tem um fogo/terra (4x fraco a água) na frente de um água,
	// e um água bem resistente no banco.
	agua := lutador("blastoise", []string{"water"}, "", 79, 85, 100, 78)
	fogo := lutador("camerupt", []string{"fire", "ground"}, "", 70, 105, 75, 40)
	reserva := lutador("vaporeon", []string{"water"}, "", 130, 110, 95, 65)
	b, _ := NovaBatalha(1, []Lutador{agua}, []Lutador{fogo, reserva}, semSorte{})

	if e := EscolherJogadaCPU(b, tabelaTeste, sorteadorFixo{}); e.Acao != AcaoTrocar || e.Alvo != 1 {
		t.Fatalf("o rival deveria trocar para o vaporeon, escolheu %+v", e)
	}
}

func TestIA_ProtegeContraFakeOut(t *testing.T) {
	eu := lutador("incineroar", []string{"fire", "dark"}, "", 95, 115, 90, 60)
	eu.AprendeFakeOut = true
	rival := lutador("snorlax", []string{"normal"}, "", 160, 110, 65, 30)
	b, _ := NovaBatalha(1, []Lutador{eu}, []Lutador{rival}, sorteadorFixo{})

	if e := EscolherJogadaCPU(b, tabelaTeste, sorteadorFixo{}); e.Acao != AcaoProtect {
		t.Fatalf("o rival deveria usar Protect contra a ameaça de Fake Out, escolheu %+v", e)
	}
}

func TestIA_SubstitutoComMelhorConfronto(t *testing.T) {
	eu := lutador("charizard", []string{"fire"}, "", 78, 109, 85, 100)
	planta := lutador("venusaur", []string{"grass"}, "", 80, 100, 100, 80)
	agua := lutador("blastoise", []string{"water"}, "", 79, 85, 100, 78)
	morto := lutador("magikarp", []string{"water"}, "", 20, 10, 55, 80)
	b, _ := NovaBatalha(1, []Lutador{eu}, []Lutador{morto, planta, agua}, semSorte{})
	b.CPU.Time[0].HP = 0

	if i := EscolherSubstitutoCPU(b, tabelaTeste); i != 2 {
		t.Fatalf("contra fogo, o rival deveria mandar o blastoise (2), mandou %d", i)
	}
}
