// Comando jogar: cliente de terminal do PokéBatalha. Não é um microsserviço,
// só consome as APIs REST de jogadores e batalha.
//
//	go run ./cmd/jogar
//
// Variáveis opcionais: JOGADORES_URL (padrão http://localhost:8082)
// e BATALHA_URL (padrão http://localhost:8083).
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type cliente struct {
	jogadoresURL string
	batalhaURL   string
	http         *http.Client
	entrada      *bufio.Scanner
}

type pity struct {
	EpicoEm    int `json:"epico_garantido_em"`
	LendarioEm int `json:"lendario_garantido_em"`
}

type jogador struct {
	ID     int64  `json:"id"`
	Nome   string `json:"nome"`
	Moedas int    `json:"moedas"`
	Pity   pity   `json:"pity"`
}

type stats struct {
	HP             int `json:"hp"`
	Ataque         int `json:"ataque"`
	Defesa         int `json:"defesa"`
	AtaqueEspecial int `json:"ataque_especial"`
	DefesaEspecial int `json:"defesa_especial"`
	Velocidade     int `json:"velocidade"`
}

type poder struct {
	Total           int    `json:"total"`
	Stats           int    `json:"stats"`
	Tipagem         int    `json:"tipagem"`
	Habilidade      int    `json:"habilidade"`
	FakeOut         int    `json:"fake_out"`
	HabilidadeAtiva string `json:"habilidade_ativa"`
}

type detalhes struct {
	Tier  string   `json:"tier"`
	Tipos []string `json:"tipos"`
	Poder poder    `json:"poder"`
	Stats stats    `json:"stats"`
}

type itemColecao struct {
	PokemonID int       `json:"pokemon_id"`
	Nome      string    `json:"nome"`
	Tier      string    `json:"tier"`
	Tipos     []string  `json:"tipos"`
	Detalhes  *detalhes `json:"detalhes"`
}

type lutador struct {
	Nome       string   `json:"nome"`
	Tier       string   `json:"tier"`
	Poder      int      `json:"poder"`
	Tipos      []string `json:"tipos"`
	Habilidade string   `json:"habilidade"`
	HP         int      `json:"hp"`
	HPMax      int      `json:"hp_max"`
	Estagio    int      `json:"estagio_ataque"`
	EstagioVel int      `json:"estagio_velocidade"`
}

type lado struct {
	Time  []lutador `json:"time"`
	Ativo int       `json:"ativo"`
}

type batalha struct {
	ID                int64    `json:"id"`
	Status            string   `json:"status"`
	Turno             int      `json:"turno"`
	Clima             string   `json:"clima"`
	Jogador           lado     `json:"jogador"`
	CPU               lado     `json:"cpu"`
	PrecisaTrocar     bool     `json:"precisa_trocar"`
	Log               []string `json:"log"`
	Recompensa        int      `json:"recompensa"`
	AcoesDisponiveis  []string `json:"acoes_disponiveis"`
	TrocasDisponiveis []int    `json:"trocas_disponiveis"`
}

func main() {
	c := &cliente{
		jogadoresURL: variavel("JOGADORES_URL", "http://localhost:8082"),
		batalhaURL:   variavel("BATALHA_URL", "http://localhost:8083"),
		http:         &http.Client{},
		entrada:      bufio.NewScanner(os.Stdin),
	}
	if err := c.rodar(); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func variavel(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

func (c *cliente) perguntar(texto string) (string, error) {
	fmt.Print(texto)
	if !c.entrada.Scan() {
		if err := c.entrada.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(c.entrada.Text()), nil
}

func (c *cliente) rodar() error {
	fmt.Println("=== PokéBatalha ===")
	j, err := c.entrar()
	if err != nil {
		return err
	}

	for {
		fmt.Printf("\n[%s | %d moedas | pity: épico+ em %d, lendário em %d]\n",
			j.Nome, j.Moedas, j.Pity.EpicoEm, j.Pity.LendarioEm)
		fmt.Println("  1) Abrir pacote  2) Ver coleção  3) Batalhar  0) Sair")
		op, err := c.perguntar("> ")
		if err != nil {
			return err
		}
		switch op {
		case "1":
			err = c.abrirPacote(&j)
		case "2":
			_, err = c.mostrarColecao(j.ID)
		case "3":
			err = c.batalhar(&j)
		case "0":
			fmt.Println("Até a próxima!")
			return nil
		default:
			fmt.Println("Opção inválida.")
		}
		if errors.Is(err, io.EOF) {
			return err
		}
		if err != nil {
			fmt.Println("⚠", err)
		}
	}
}

func (c *cliente) entrar() (jogador, error) {
	for {
		resp, err := c.perguntar("Seu ID de jogador (Enter para criar uma conta nova): ")
		if err != nil {
			return jogador{}, err
		}
		var j jogador
		if resp == "" {
			nome, err := c.perguntar("Escolha um nome: ")
			if err != nil {
				return jogador{}, err
			}
			err = c.chamar(http.MethodPost, c.jogadoresURL+"/jogadores", map[string]string{"nome": nome}, &j)
			if err == nil {
				fmt.Printf("Conta criada! Seu ID é %d (guarde para entrar depois).\n", j.ID)
				return j, nil
			}
			fmt.Println("⚠", err)
			continue
		}
		if err := c.chamar(http.MethodGet, c.jogadoresURL+"/jogadores/"+resp, nil, &j); err != nil {
			fmt.Println("⚠", err)
			continue
		}
		fmt.Printf("Bem-vindo de volta, %s!\n", j.Nome)
		return j, nil
	}
}

func (c *cliente) abrirPacote(j *jogador) error {
	var res struct {
		Pokemon   itemColecao `json:"pokemon"`
		Saldo     int         `json:"saldo"`
		Garantido bool        `json:"garantido_pelo_pity"`
		Pity      pity        `json:"pity"`
	}
	url := fmt.Sprintf("%s/jogadores/%d/pacotes", c.jogadoresURL, j.ID)
	if err := c.chamar(http.MethodPost, url, nil, &res); err != nil {
		return err
	}
	j.Moedas, j.Pity = res.Saldo, res.Pity

	p := res.Pokemon
	extra := ""
	if res.Garantido {
		extra = " (garantido pelo pity!)"
	}
	fmt.Printf("\nVocê abriu o pacote e tirou... %s %s (%s)%s\n",
		destaqueTier(p.Tier), exibir(p.Nome), tipos(p.Tipos), extra)
	if d := p.Detalhes; d != nil {
		fmt.Printf("  %s\n  %s\n", composicao(d.Poder), linhaStats(d.Stats))
	}
	return nil
}

func (c *cliente) mostrarColecao(id int64) ([]itemColecao, error) {
	var res struct {
		Pokemons              []itemColecao `json:"pokemons"`
		DetalhesIndisponiveis bool          `json:"detalhes_indisponiveis"`
	}
	if err := c.chamar(http.MethodGet, fmt.Sprintf("%s/jogadores/%d/colecao", c.jogadoresURL, id), nil, &res); err != nil {
		return nil, err
	}
	if len(res.Pokemons) == 0 {
		fmt.Println("Sua coleção está vazia. Abra um pacote!")
		return nil, nil
	}
	if res.DetalhesIndisponiveis {
		fmt.Println("(o catálogo está fora do ar: mostrando sem stats)")
	}

	fmt.Printf("\n  %-5s %-17s %-9s %5s  %-15s %-17s %s\n", "ID", "Pokémon", "Tier", "Poder", "Tipos", "Habilidade", "HP/Atq/Def/AtE/DfE/Vel")
	vistos := map[int]int{}
	var unicos []itemColecao
	for _, p := range res.Pokemons {
		if vistos[p.PokemonID]++; vistos[p.PokemonID] == 1 {
			unicos = append(unicos, p)
		}
	}
	for _, p := range unicos {
		nome := exibir(p.Nome)
		if n := vistos[p.PokemonID]; n > 1 {
			nome = fmt.Sprintf("%s x%d", nome, n)
		}
		if d := p.Detalhes; d != nil {
			s := d.Stats
			fmt.Printf("  %-5d %-17s %-9s %5d  %-15s %-17s %d/%d/%d/%d/%d/%d\n",
				p.PokemonID, nome, d.Tier, d.Poder.Total, tipos(d.Tipos), exibir(d.Poder.HabilidadeAtiva),
				s.HP, s.Ataque, s.Defesa, s.AtaqueEspecial, s.DefesaEspecial, s.Velocidade)
		} else {
			fmt.Printf("  %-5d %-17s %-9s %5s  %s\n", p.PokemonID, nome, p.Tier, "?", tipos(p.Tipos))
		}
	}
	return res.Pokemons, nil
}

func (c *cliente) batalhar(j *jogador) error {
	colecao, err := c.mostrarColecao(j.ID)
	if err != nil || len(colecao) == 0 {
		return err
	}
	resp, err := c.perguntar("\nIDs do seu time (até 3, separados por espaço; o primeiro começa em campo): ")
	if err != nil {
		return err
	}
	var time []int
	for _, campo := range strings.Fields(resp) {
		id, err := strconv.Atoi(campo)
		if err != nil {
			return fmt.Errorf("%q não é um número", campo)
		}
		time = append(time, id)
	}

	var b batalha
	corpo := map[string]any{"jogador_id": j.ID, "time": time}
	if err := c.chamar(http.MethodPost, c.batalhaURL+"/batalhas", corpo, &b); err != nil {
		return err
	}
	mostrarTimeRival(b)

	for {
		mostrarLog(b.Log)
		if b.Status != "em_andamento" {
			break
		}
		mostrarCampo(b)

		acao, alvo, err := c.escolherAcao(b)
		if err != nil {
			return err
		}
		var prox batalha
		url := fmt.Sprintf("%s/batalhas/%d/turnos", c.batalhaURL, b.ID)
		if err := c.chamar(http.MethodPost, url, map[string]any{"acao": acao, "alvo": alvo}, &prox); err != nil {
			fmt.Println("⚠", err)
			continue
		}
		b = prox
	}

	if b.Status == "vitoria" {
		j.Moedas += b.Recompensa
		fmt.Printf("\n🏆 VITÓRIA! +%d moedas.\n", b.Recompensa)
	} else {
		fmt.Println("\n💀 Derrota... Abra mais pacotes e tente de novo.")
	}
	return nil
}

func (c *cliente) escolherAcao(b batalha) (string, int, error) {
	seu := b.Jogador.Time[b.Jogador.Ativo]
	if b.PrecisaTrocar {
		fmt.Println("Seu Pokémon desmaiou! Quem entra?")
		alvo, err := c.escolherTroca(b)
		return "trocar", alvo, err
	}

	nomeTipo := func(i int) string {
		if len(seu.Tipos) > i {
			return nomeTipo(seu.Tipos[i])
		}
		return "?"
	}
	nomes := map[string]string{
		"fisico":   "Ataque Físico (poder 50, sempre neutro)",
		"tipo":     fmt.Sprintf("Ataque de %s (poder 70)", nomeTipo(0)),
		"tipo2":    fmt.Sprintf("Ataque de %s (poder 70)", nomeTipo(1)),
		"fake_out": "Fake Out (prioridade, faz o rival recuar)",
		"protect":  "Protect (bloqueia o golpe do rival; não pode repetir seguido)",
		"trocar":   "Trocar de Pokémon (vai primeiro, mas quem entra toma o golpe)",
	}
	for i, a := range b.AcoesDisponiveis {
		fmt.Printf("  %d) %s\n", i+1, nomes[a])
	}
	for {
		resp, err := c.perguntar("> ")
		if err != nil {
			return "", 0, err
		}
		n, err := strconv.Atoi(resp)
		if err != nil || n < 1 || n > len(b.AcoesDisponiveis) {
			fmt.Println("Escolha um dos números acima.")
			continue
		}
		acao := b.AcoesDisponiveis[n-1]
		if acao != "trocar" {
			return acao, 0, nil
		}
		alvo, err := c.escolherTroca(b)
		return acao, alvo, err
	}
}

func (c *cliente) escolherTroca(b batalha) (int, error) {
	for i, idx := range b.TrocasDisponiveis {
		fmt.Printf("  %d) %s\n", i+1, linhaLutador(b.Jogador.Time[idx]))
	}
	for {
		resp, err := c.perguntar("> ")
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(resp)
		if err == nil && n >= 1 && n <= len(b.TrocasDisponiveis) {
			return b.TrocasDisponiveis[n-1], nil
		}
		fmt.Println("Escolha um dos números acima.")
	}
}

func mostrarTimeRival(b batalha) {
	fmt.Println("\nTime do rival:")
	for _, l := range b.CPU.Time {
		fmt.Printf("  %-16s %-9s poder %d  %s\n", exibir(l.Nome), l.Tier, l.Poder, tipos(l.Tipos))
	}
}

func mostrarLog(linhas []string) {
	fmt.Println()
	for _, l := range linhas {
		fmt.Println("  " + l)
	}
}

func mostrarCampo(b batalha) {
	seu, rival := b.Jogador.Time[b.Jogador.Ativo], b.CPU.Time[b.CPU.Ativo]
	fmt.Printf("\n--- Turno %d", b.Turno+1)
	if b.Clima != "" {
		fmt.Printf(" | clima: %s", b.Clima)
	}
	fmt.Println(" ---")
	fmt.Println("  Rival: " + linhaLutador(rival) + "   " + banco(b.CPU))
	fmt.Println("  Você:  " + linhaLutador(seu) + "   " + banco(b.Jogador))
}

func banco(l lado) string {
	vivos := 0
	for i, p := range l.Time {
		if i != l.Ativo && p.HP > 0 {
			vivos++
		}
	}
	return fmt.Sprintf("(banco: %d)", vivos)
}

func linhaLutador(l lutador) string {
	extra := ""
	if l.Habilidade != "" {
		extra += " | " + exibir(l.Habilidade)
	}
	if l.Estagio != 0 {
		extra += fmt.Sprintf(" | atq %+d", l.Estagio)
	}
	if l.EstagioVel != 0 {
		extra += fmt.Sprintf(" | vel %+d", l.EstagioVel)
	}
	return fmt.Sprintf("%-16s %s %3d/%-3d %s%s",
		exibir(l.Nome), barra(l.HP, l.HPMax), l.HP, l.HPMax, tipos(l.Tipos), extra)
}

func composicao(p poder) string {
	partes := []string{fmt.Sprintf("stats %d", p.Stats), fmt.Sprintf("tipagem %+d", p.Tipagem)}
	if p.Habilidade > 0 {
		partes = append(partes, fmt.Sprintf("%s %+d", exibir(p.HabilidadeAtiva), p.Habilidade))
	}
	if p.FakeOut > 0 {
		partes = append(partes, fmt.Sprintf("Fake Out %+d", p.FakeOut))
	}
	return fmt.Sprintf("Poder %d = %s", p.Total, strings.Join(partes, ", "))
}

func linhaStats(s stats) string {
	return fmt.Sprintf("HP %d | Atq %d | Def %d | AtE %d | DfE %d | Vel %d",
		s.HP, s.Ataque, s.Defesa, s.AtaqueEspecial, s.DefesaEspecial, s.Velocidade)
}

func barra(hp, max int) string {
	const largura = 20
	cheio := 0
	if max > 0 {
		cheio = hp * largura / max
	}
	return "[" + strings.Repeat("#", cheio) + strings.Repeat("-", largura-cheio) + "]"
}

func destaqueTier(t string) string {
	switch t {
	case "lendario":
		return "★★★ LENDÁRIO ★★★"
	case "epico":
		return "★★ ÉPICO ★★"
	case "raro":
		return "★ RARO ★"
	default:
		return "comum:"
	}
}

func exibir(nome string) string {
	if nome == "" {
		return "-"
	}
	partes := strings.Split(nome, "-")
	for i, p := range partes {
		if p != "" {
			partes[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(partes, " ")
}

var nomesTipos = map[string]string{
	"normal": "Normal", "fire": "Fogo", "water": "Água", "electric": "Elétrico",
	"grass": "Planta", "ice": "Gelo", "fighting": "Lutador", "poison": "Venenoso",
	"ground": "Terra", "flying": "Voador", "psychic": "Psíquico", "bug": "Inseto",
	"rock": "Pedra", "ghost": "Fantasma", "dragon": "Dragão", "dark": "Sombrio",
	"steel": "Aço", "fairy": "Fada",
}

func nomeTipo(t string) string {
	if n, ok := nomesTipos[t]; ok {
		return n
	}
	return t
}

func tipos(ts []string) string {
	nomes := make([]string, 0, len(ts))
	for _, t := range ts {
		nomes = append(nomes, nomeTipo(t))
	}
	return strings.Join(nomes, "/")
}

// chamar faz a requisição, confere o status e decodifica a resposta.
func (c *cliente) chamar(metodo, url string, corpo, destino any) error {
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()

	var leitor io.Reader
	if corpo != nil {
		dados, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(dados)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, url, leitor)
	if err != nil {
		return err
	}
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("não consegui falar com o servidor (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var e struct {
			Mensagem string `json:"mensagem"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil || e.Mensagem == "" {
			return fmt.Errorf("servidor respondeu %s", resp.Status)
		}
		return errors.New(e.Mensagem)
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}
