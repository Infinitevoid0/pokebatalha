// Package sorteio fornece a fonte de aleatoriedade real usada em produção.
// Nos testes, cada serviço injeta um sorteador determinístico no lugar deste.
package sorteio

import "math/rand/v2"

// Aleatorio implementa os sorteadores dos serviços com math/rand/v2.
type Aleatorio struct{}

// IntN devolve um inteiro em [0, n).
func (Aleatorio) IntN(n int) int { return rand.IntN(n) }

// Float64 devolve um número em [0.0, 1.0).
func (Aleatorio) Float64() float64 { return rand.Float64() }
