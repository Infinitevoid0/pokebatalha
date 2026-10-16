// jogadores, moedas e gacha.
package dominio

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// erros de negócio.
var (
	ErrNomeInvalido         = errors.New("nome inválido")
	ErrNomeEmUso            = errors.New("nome já está em uso")
	ErrJogadorNaoEncontrado = errors.New("jogador não encontrado")
	ErrSaldoInsuficiente    = errors.New("saldo insuficiente")
	ErrQuantidadeInvalida   = errors.New("quantidade inválida")
	ErrTimeInvalido         = errors.New("time inválido")
	ErrPokemonNaoPertence   = errors.New("o jogador não possui esse pokémon")
	ErrCompraSimultanea     = errors.New("outro pacote está sendo aberto ao mesmo tempo, tente de novo")
)

// infra: outro microsserviço não respondeu.
var ErrServicoIndisponivel = errors.New("serviço indisponível")

// conta de quem joga.
type Jogador struct {
	ID       int64     `json:"id"`
	Nome     string    `json:"nome"`
	Moedas   int       `json:"moedas"`
	Pity     Pity      `json:"-"`
	CriadoEm time.Time `json:"criado_em"`
}

// atributos base.
type Stats struct {
	HP             int `json:"hp"`
	Ataque         int `json:"ataque"`
	Defesa         int `json:"defesa"`
	AtaqueEspecial int `json:"ataque_especial"`
	DefesaEspecial int `json:"defesa_especial"`
	Velocidade     int `json:"velocidade"`
}

// de onde vem o índice de poder.
type ComposicaoPoder struct {
	Total           int    `json:"total"`
	Stats           int    `json:"stats"`
	Tipagem         int    `json:"tipagem"`
	Habilidade      int    `json:"habilidade"`
	FakeOut         int    `json:"fake_out"`
	HabilidadeAtiva string `json:"habilidade_ativa,omitempty"`
}

// dados atuais do Pokémon (do catálogo).
type Detalhes struct {
	Tier  string          `json:"tier"`
	Tipos []string        `json:"tipos"`
	Poder ComposicaoPoder `json:"poder"`
	Stats Stats           `json:"stats"`
}

// o que o catálogo devolve ao abrir um pacote.
type PokemonSorteado struct {
	PokemonID int
	Nome      string
	Detalhes  Detalhes
}

// um Pokémon que o jogador possui.
type ItemColecao struct {
	ID          int64     `json:"id"`
	PokemonID   int       `json:"pokemon_id"`
	Nome        string    `json:"nome"`
	Tier        string    `json:"tier"`
	Tipos       []string  `json:"tipos"`
	AdquiridoEm time.Time `json:"adquirido_em"`
	Detalhes    *Detalhes `json:"detalhes,omitempty"`
}

const (
	tamanhoMinimoNome = 3
	tamanhoMaximoNome = 20
)

// tira espaços das pontas e valida o tamanho.
func NormalizarNome(nome string) (string, error) {
	nome = strings.TrimSpace(nome)
	n := utf8.RuneCountInString(nome)
	if n < tamanhoMinimoNome || n > tamanhoMaximoNome {
		return "", fmt.Errorf("%w: precisa ter entre %d e %d caracteres",
			ErrNomeInvalido, tamanhoMinimoNome, tamanhoMaximoNome)
	}
	return nome, nil
}
