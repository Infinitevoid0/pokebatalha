package dominio

import (
	"errors"
	"testing"
)

type sorteadorFixo int

func (s sorteadorFixo) IntN(int) int { return int(s) }

func TestSortearTier_FaixasDeProbabilidade(t *testing.T) {
	casos := map[int]string{
		0: "lendario", 1: "lendario",
		2: "epico", 9: "epico",
		10: "raro", 29: "raro",
		30: "comum", 99: "comum",
	}
	for rolagem, esperado := range casos {
		if got := SortearTier(sorteadorFixo(rolagem)); got != esperado {
			t.Errorf("rolagem %d: esperava %s, veio %s", rolagem, esperado, got)
		}
	}
}

func TestChancesSomam100(t *testing.T) {
	soma := 0
	for _, c := range Chances {
		soma += c.Porcentagem
	}
	if soma != 100 {
		t.Fatalf("as chances somam %d, deveriam somar 100", soma)
	}
}

func TestNormalizarNome(t *testing.T) {
	if n, err := NormalizarNome("  theo  "); err != nil || n != "theo" {
		t.Errorf("esperava 'theo' sem erro, veio %q, %v", n, err)
	}
	for _, invalido := range []string{"", "ab", "um nome grande demais pra caber"} {
		if _, err := NormalizarNome(invalido); !errors.Is(err, ErrNomeInvalido) {
			t.Errorf("%q deveria ser inválido, veio %v", invalido, err)
		}
	}
}

type rolagens struct{ valores []int }

func (r *rolagens) IntN(int) int {
	v := r.valores[0]
	r.valores = r.valores[1:]
	return v
}

func TestSortearComPity(t *testing.T) {
	casos := []struct {
		nome      string
		pity      Pity
		rolagem   int
		tier      string
		garantido bool
		depois    Pity
	}{
		{"sorteio normal sobe os contadores", Pity{3, 3}, 50, "comum", false, Pity{4, 4}},
		{"épico natural zera só o de épico", Pity{3, 3}, 5, "epico", false, Pity{0, 4}},
		{"lendário natural zera os dois", Pity{3, 3}, 0, "lendario", false, Pity{0, 0}},
		{"10º pacote garante épico", Pity{9, 9}, 5, "epico", true, Pity{0, 10}},
		{"garantia de épico pode virar lendário", Pity{9, 9}, 1, "lendario", true, Pity{0, 0}},
		{"50º pacote garante lendário", Pity{2, 49}, 99, "lendario", true, Pity{0, 0}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			r := SortearComPity(&rolagens{valores: []int{c.rolagem}}, c.pity)
			if r.Tier != c.tier || r.Garantido != c.garantido || r.Pity != c.depois {
				t.Fatalf("esperava %s garantido=%v %+v, veio %s garantido=%v %+v",
					c.tier, c.garantido, c.depois, r.Tier, r.Garantido, r.Pity)
			}
		})
	}
}
