// aleatoriedade real; nos testes entra um sorteador fixo.
package sorteio

import "math/rand/v2"

// sorteador de produção (math/rand/v2).
type Aleatorio struct{}

// inteiro em [0, n).
func (Aleatorio) IntN(n int) int { return rand.IntN(n) }

// número em [0, 1).
func (Aleatorio) Float64() float64 { return rand.Float64() }
