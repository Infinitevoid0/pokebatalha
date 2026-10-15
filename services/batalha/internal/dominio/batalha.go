package dominio

import (
	"fmt"
	"slices"
)

// Sorteador permite trocar a aleatoriedade por algo previsível nos testes.
type Sorteador interface {
	IntN(n int) int
	Float64() float64
}

// StatusBatalha indica se a batalha continua ou quem venceu.
type StatusBatalha string

const (
	EmAndamento StatusBatalha = "em_andamento"
	Vitoria     StatusBatalha = "vitoria"
	Derrota     StatusBatalha = "derrota"
)

// Clima afeta golpes de Fogo e Água.
type Clima string

const (
	SemClima   Clima = ""
	ClimaSol   Clima = "sol"
	ClimaChuva Clima = "chuva"
)

const (
	duracaoClima      = 5
	TamanhoMaximoTime = 3
)

// Lado é o time de um dos jogadores e qual Pokémon está em campo.
type Lado struct {
	Time  []Lutador `json:"time"`
	Ativo int       `json:"ativo"`
}

func (l *Lado) atual() *Lutador { return &l.Time[l.Ativo] }

// TrocasDisponiveis lista os índices do banco que podem entrar.
func (l *Lado) TrocasDisponiveis() []int {
	trocas := []int{}
	for i := range l.Time {
		if i != l.Ativo && !l.Time[i].Desmaiado() {
			trocas = append(trocas, i)
		}
	}
	return trocas
}

// Batalha é o estado completo de uma partida.
type Batalha struct {
	ID          int64         `json:"id"`
	JogadorID   int64         `json:"jogador_id"`
	Status      StatusBatalha `json:"status"`
	Turno       int           `json:"turno"`
	Clima       Clima         `json:"clima,omitempty"`
	TurnosClima int           `json:"turnos_clima,omitempty"`
	Jogador     Lado          `json:"jogador"`
	CPU         Lado          `json:"cpu"`
	// PrecisaTrocar: o Pokémon do jogador desmaiou e ele escolhe quem entra
	// (a troca forçada não gasta turno e o rival não age).
	PrecisaTrocar bool     `json:"precisa_trocar"`
	Log           []string `json:"log"` // o que aconteceu no último turno
	Recompensa    int      `json:"recompensa,omitempty"`
	Versao        int      `json:"-"`
}

// combatente junta o lutador com o nome que aparece no log.
type combatente struct {
	l    *Lutador
	nome string
}

func (b *Batalha) seu() combatente {
	l := b.Jogador.atual()
	return combatente{l: l, nome: "Seu " + Exibir(l.Nome)}
}

func (b *Batalha) rival() combatente {
	l := b.CPU.atual()
	return combatente{l: l, nome: Exibir(l.Nome) + " do rival"}
}

func (b *Batalha) daqueleLado(l *Lado) (eu, oponente combatente) {
	if l == &b.CPU {
		return b.rival(), b.seu()
	}
	return b.seu(), b.rival()
}

func (b *Batalha) registrar(formato string, args ...any) {
	b.Log = append(b.Log, fmt.Sprintf(formato, args...))
}

// NovaBatalha monta a partida e ativa as habilidades de entrada dos dois
// primeiros Pokémon (o mais rápido primeiro).
func NovaBatalha(jogadorID int64, timeJogador, timeCPU []Lutador, s Sorteador) (*Batalha, error) {
	if len(timeJogador) == 0 || len(timeJogador) > TamanhoMaximoTime || len(timeCPU) == 0 {
		return nil, fmt.Errorf("%w: o time precisa ter de 1 a %d pokémon", ErrTimeInvalido, TamanhoMaximoTime)
	}
	b := &Batalha{
		JogadorID: jogadorID,
		Status:    EmAndamento,
		Jogador:   Lado{Time: timeJogador},
		CPU:       Lado{Time: timeCPU},
	}

	seu, rival := b.seu(), b.rival()
	b.registrar("A batalha começou! Você envia %s, o rival envia %s.", Exibir(seu.l.Nome), Exibir(rival.l.Nome))
	if maisRapido(seu.l, rival.l, s) {
		b.aoEntrar(seu, rival)
		b.aoEntrar(rival, seu)
	} else {
		b.aoEntrar(rival, seu)
		b.aoEntrar(seu, rival)
	}
	return b, nil
}

func maisRapido(a, c *Lutador, s Sorteador) bool {
	va, vc := velocidadeEfetiva(a), velocidadeEfetiva(c)
	if va != vc {
		return va > vc
	}
	return s.IntN(2) == 0 // empate: cara ou coroa
}

// AcoesDoJogador lista o que o jogador pode fazer agora.
func (b *Batalha) AcoesDoJogador() []Acao {
	if b.Status != EmAndamento {
		return []Acao{}
	}
	if b.PrecisaTrocar {
		return []Acao{AcaoTrocar}
	}
	return acoesDisponiveis(&b.Jogador)
}

func acoesDisponiveis(lado *Lado) []Acao {
	l := lado.atual()
	acoes := golpesDisponiveis(l)
	if l.PodeUsarFakeOut() {
		acoes = append(acoes, AcaoFakeOut)
	}
	if !l.UsouProtect {
		acoes = append(acoes, AcaoProtect)
	}
	if len(lado.TrocasDisponiveis()) > 0 {
		acoes = append(acoes, AcaoTrocar)
	}
	return acoes
}

func validarEscolha(lado *Lado, e Escolha) error {
	disponiveis := acoesDisponiveis(lado)
	if !slices.Contains(disponiveis, e.Acao) {
		return fmt.Errorf("%w: %q não está disponível agora (disponíveis: %v)", ErrAcaoInvalida, e.Acao, disponiveis)
	}
	if e.Acao == AcaoTrocar && !slices.Contains(lado.TrocasDisponiveis(), e.Alvo) {
		return fmt.Errorf("%w: não dá para trocar pelo pokémon %d (disponíveis: %v)",
			ErrAcaoInvalida, e.Alvo, lado.TrocasDisponiveis())
	}
	return nil
}

// jogada é a escolha de um dos lados dentro do turno. O lutador que age é
// resolvido na hora de agir, porque uma troca antes pode ter mudado quem
// está em campo.
type jogada struct {
	lado    *Lado
	escolha Escolha
	recuou  bool
}

// ExecutarTurno resolve um turno inteiro: escolha do jogador, escolha da CPU,
// efeitos de fim de turno e substituição de quem desmaiou.
func (b *Batalha) ExecutarTurno(e Escolha, t TabelaTipos, s Sorteador) error {
	if b.Status != EmAndamento {
		return ErrBatalhaEncerrada
	}
	if b.PrecisaTrocar {
		return b.trocaForcada(e)
	}
	if err := validarEscolha(&b.Jogador, e); err != nil {
		return err
	}
	escolhaCPU := EscolherJogadaCPU(b, t, s)

	b.Log = nil
	b.Turno++

	primeira := &jogada{lado: &b.Jogador, escolha: e}
	segunda := &jogada{lado: &b.CPU, escolha: escolhaCPU}
	if !agePrimeiro(primeira, segunda, s) {
		primeira, segunda = segunda, primeira
	}

	for i, j := range []*jogada{primeira, segunda} {
		atk, alvo := b.daqueleLado(j.lado)
		if atk.l.Desmaiado() {
			continue
		}

		switch j.escolha.Acao {
		case AcaoTrocar:
			b.trocar(j.lado, j.escolha.Alvo, true)
			continue
		case AcaoProtect:
			atk.l.protegido, atk.l.UsouProtect = true, true
			b.registrar("%s está se protegendo!", atk.nome)
			continue
		}

		atk.l.UsouProtect = false
		if j.recuou {
			b.registrar("%s recuou e não conseguiu se mover!", atk.nome)
			continue
		}
		acertou := b.executarGolpe(atk, alvo, j.escolha.Acao, t, s)
		if i == 0 && acertou && j.escolha.Acao == AcaoFakeOut &&
			!alvo.l.Desmaiado() && alvo.l.Habilidade != "inner-focus" {
			segunda.recuou = true
		}
	}

	b.fimDeTurno()
	b.substituirDesmaiados(t)
	return nil
}

// trocaForcada coloca em campo o Pokémon escolhido depois de um desmaio.
func (b *Batalha) trocaForcada(e Escolha) error {
	trocas := b.Jogador.TrocasDisponiveis()
	if e.Acao != AcaoTrocar || !slices.Contains(trocas, e.Alvo) {
		return fmt.Errorf("%w: seu pokémon desmaiou, escolha quem entra com \"trocar\" (disponíveis: %v)",
			ErrAcaoInvalida, trocas)
	}
	b.Log = nil
	b.PrecisaTrocar = false
	b.trocar(&b.Jogador, e.Alvo, false)
	return nil
}

func prioridade(a Acao) int {
	switch a {
	case AcaoTrocar:
		return 6
	case AcaoProtect:
		return 4
	case AcaoFakeOut:
		return 3
	default:
		return 0
	}
}

// agePrimeiro: maior prioridade vence; empatou, vence o mais rápido.
func agePrimeiro(a, c *jogada, s Sorteador) bool {
	if pa, pc := prioridade(a.escolha.Acao), prioridade(c.escolha.Acao); pa != pc {
		return pa > pc
	}
	return maisRapido(a.lado.atual(), c.lado.atual(), s)
}

// trocar tira o Pokémon atual e coloca o do índice alvo. Quem sai perde os
// estágios (inclusive o -1 do Intimidate); quem entra ativa a habilidade.
func (b *Batalha) trocar(lado *Lado, alvo int, noMeioDoTurno bool) {
	sai, _ := b.daqueleLado(lado)
	sai.l.EstagioAtaque, sai.l.EstagioVelocidade = 0, 0
	sai.l.UsouProtect, sai.l.protegido = false, false

	lado.Ativo = alvo
	entra, oponente := b.daqueleLado(lado)
	entra.l.TurnosEmCampo = 0
	entra.l.recemChegado = noMeioDoTurno

	switch {
	case lado == &b.CPU && sai.l.Desmaiado():
		b.registrar("O rival envia %s!", Exibir(entra.l.Nome))
	case lado == &b.CPU:
		b.registrar("O rival recolhe %s e envia %s!", Exibir(sai.l.Nome), Exibir(entra.l.Nome))
	case sai.l.Desmaiado():
		b.registrar("Vai, %s!", Exibir(entra.l.Nome))
	default:
		b.registrar("Volte, %s! Vai, %s!", Exibir(sai.l.Nome), Exibir(entra.l.Nome))
	}
	b.aoEntrar(entra, oponente)
}

// executarGolpe aplica o dano e registra o resultado. Devolve se acertou.
func (b *Batalha) executarGolpe(atk, alvo combatente, acao Acao, t TabelaTipos, s Sorteador) bool {
	g := golpeDe(acao, atk.l)
	b.registrar("%s usou %s!", atk.nome, g.nome)

	if alvo.l.protegido {
		b.registrar("%s se protegeu do golpe!", alvo.nome)
		return false
	}

	r := calcularDano(g, atk.l, alvo.l, b.Clima, t, s)
	switch {
	case r.imunePor != "":
		b.registrar("%s é imune graças a %s!", alvo.nome, Exibir(r.imunePor))
		return false
	case r.mult == 0:
		b.registrar("Não afeta %s...", alvo.nome)
		return false
	}

	alvo.l.HP = max(alvo.l.HP-r.dano, 0)
	switch {
	case r.mult >= 2:
		b.registrar("É super efetivo!")
	case r.mult < 1:
		b.registrar("Não é muito efetivo...")
	}
	b.registrar("%s perdeu %d de HP (%d/%d).", alvo.nome, r.dano, alvo.l.HP, alvo.l.HPMax)
	if r.sturdy {
		b.registrar("%s aguentou firme com Sturdy!", alvo.nome)
	}

	if alvo.l.Desmaiado() {
		b.registrar("%s desmaiou!", alvo.nome)
		if atk.l.Habilidade == "moxie" && mudarEstagio(&atk.l.EstagioAtaque, 1) > 0 {
			b.registrar("Moxie de %s aumentou o ataque!", atk.nome)
		}
	}
	return true
}

// aoEntrar ativa as habilidades que disparam quando o Pokémon entra em campo.
func (b *Batalha) aoEntrar(c, oponente combatente) {
	switch c.l.Habilidade {
	case "intimidate":
		b.intimidar(c, oponente)
	case "intrepid-sword":
		if mudarEstagio(&c.l.EstagioAtaque, 1) > 0 {
			b.registrar("Intrepid Sword de %s aumentou o ataque!", c.nome)
		}
	case "drought", "orichalcum-pulse":
		b.mudarClima(ClimaSol, c)
	case "drizzle":
		b.mudarClima(ClimaChuva, c)
	case "protosynthesis", "quark-drive":
		b.registrar("%s ativou %s: %s aumentado!", c.nome, Exibir(c.l.Habilidade), c.l.atributoImpulsionado())
	case "sword-of-ruin", "beads-of-ruin":
		b.registrar("%s de %s enfraquece a defesa do oponente!", Exibir(c.l.Habilidade), c.nome)
	case "tablets-of-ruin", "vessel-of-ruin":
		b.registrar("%s de %s enfraquece o ataque do oponente!", Exibir(c.l.Habilidade), c.nome)
	}
}

func (b *Batalha) intimidar(c, alvo combatente) {
	if alvo.l.Desmaiado() {
		return
	}
	if slices.Contains(imunesAIntimidate, alvo.l.Habilidade) {
		b.registrar("%s não se deixou intimidar (%s)!", alvo.nome, Exibir(alvo.l.Habilidade))
		return
	}
	if mudarEstagio(&alvo.l.EstagioAtaque, -1) == 0 {
		b.registrar("O ataque de %s não pode baixar mais!", alvo.nome)
		return
	}
	b.registrar("Intimidate de %s reduziu o ataque de %s!", c.nome, alvo.nome)

	if alvo.l.Habilidade == "defiant" || alvo.l.Habilidade == "competitive" {
		if mudarEstagio(&alvo.l.EstagioAtaque, 2) > 0 {
			b.registrar("%s ficou furioso com %s! Ataque +2!", alvo.nome, Exibir(alvo.l.Habilidade))
		}
	}
}

func (b *Batalha) mudarClima(clima Clima, origem combatente) {
	b.Clima = clima
	b.TurnosClima = duracaoClima
	if clima == ClimaSol {
		b.registrar("%s deixou o sol forte!", origem.nome)
	} else {
		b.registrar("%s fez começar a chover!", origem.nome)
	}
}

func (b *Batalha) fimDeTurno() {
	for _, c := range []combatente{b.seu(), b.rival()} {
		c.l.protegido = false
		if c.l.Desmaiado() {
			continue
		}
		if c.l.recemChegado {
			// Entrou por troca neste turno: o próximo é o "primeiro turno"
			// dele em campo (dá para usar Fake Out).
			c.l.recemChegado = false
			continue
		}
		c.l.TurnosEmCampo++
		if c.l.Habilidade == "speed-boost" && mudarEstagio(&c.l.EstagioVelocidade, 1) > 0 {
			b.registrar("Speed Boost de %s aumentou a velocidade!", c.nome)
		}
	}

	if b.TurnosClima > 0 {
		b.TurnosClima--
		if b.TurnosClima == 0 {
			b.Clima = SemClima
			b.registrar("O clima voltou ao normal.")
		}
	}
}

// substituirDesmaiados: o rival escolhe o melhor substituto; o jogador
// escolhe o dele (se só sobrar um, ele entra sozinho). Sem ninguém, acabou.
func (b *Batalha) substituirDesmaiados(t TabelaTipos) {
	if b.CPU.atual().Desmaiado() {
		trocas := b.CPU.TrocasDisponiveis()
		if len(trocas) == 0 {
			b.Status = Vitoria
			b.registrar("Todos os Pokémon do rival desmaiaram. Você venceu!")
			return
		}
		b.trocar(&b.CPU, EscolherSubstitutoCPU(b, t), false)
	}

	if b.Jogador.atual().Desmaiado() {
		trocas := b.Jogador.TrocasDisponiveis()
		switch len(trocas) {
		case 0:
			b.Status = Derrota
			b.registrar("Todos os seus Pokémon desmaiaram. Você perdeu!")
		case 1:
			b.trocar(&b.Jogador, trocas[0], false)
		default:
			b.PrecisaTrocar = true
			b.registrar("Escolha quem entra no lugar.")
		}
	}
}
