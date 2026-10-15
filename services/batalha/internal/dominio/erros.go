// Package dominio é o motor da batalha: regras puras, sem banco, HTTP ou
// gRPC. Tudo aqui é testável com um sorteador previsível.
package dominio

import "errors"

// Erros de negócio.
var (
	ErrTimeInvalido         = errors.New("time inválido")
	ErrAcaoInvalida         = errors.New("ação inválida")
	ErrBatalhaNaoEncontrada = errors.New("batalha não encontrada")
	ErrBatalhaEncerrada     = errors.New("a batalha já terminou")
	ErrConflitoDeVersao     = errors.New("a batalha foi alterada por outra requisição, consulte de novo")
	ErrJogadorNaoEncontrado = errors.New("jogador não encontrado")
	ErrPokemonNaoPertence   = errors.New("o jogador não possui esse pokémon")
	ErrPokemonNaoEncontrado = errors.New("pokémon não encontrado")
)

// Erro de infraestrutura: catálogo ou jogadores não responderam.
var ErrServicoIndisponivel = errors.New("serviço indisponível")
