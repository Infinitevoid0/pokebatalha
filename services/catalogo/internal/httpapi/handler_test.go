package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"pokebatalha/pkg/httpjson"
	"pokebatalha/services/catalogo/internal/dominio"
)

// devolve sempre o mesmo resultado/erro.
type servicoFalso struct {
	pokemon dominio.Pokemon
	erro    error
}

func (s servicoFalso) BuscarPorID(context.Context, int) (dominio.Pokemon, error) {
	return s.pokemon, s.erro
}
func (s servicoFalso) BuscarPorNome(context.Context, string) (dominio.Pokemon, error) {
	return s.pokemon, s.erro
}
func (s servicoFalso) Sortear(context.Context, dominio.Tier, int) (dominio.Pokemon, error) {
	return s.pokemon, s.erro
}
func (s servicoFalso) Efetividade(context.Context, string, []string) (float64, error) {
	return 2, s.erro
}
func (s servicoFalso) Regras() dominio.Regras { return dominio.Regras{} }

func TestStatusHTTPPorTipoDeErro(t *testing.T) {
	casos := []struct {
		nome      string
		erro      error
		caminho   string
		statusEsp int
		codigoEsp string
	}{
		{"sucesso", nil, "/pokemons/727", http.StatusOK, ""},
		{"não encontrado", dominio.ErrPokemonNaoEncontrado, "/pokemons/naoexiste", http.StatusNotFound, "pokemon_nao_encontrado"},
		{"id inválido", dominio.ErrIDInvalido, "/pokemons/0", http.StatusBadRequest, "requisicao_invalida"},
		{"tier inválido", nil, "/pokemons/sorteio?tier=mitico", http.StatusBadRequest, "requisicao_invalida"},
		{"PokeAPI fora", fmt.Errorf("%w: timeout", dominio.ErrServicoExternoIndisponivel), "/pokemons/25", http.StatusBadGateway, "servico_externo_indisponivel"},
		{"catálogo aquecendo", dominio.ErrCatalogoAquecendo, "/pokemons/sorteio?tier=lendario", http.StatusServiceUnavailable, "catalogo_aquecendo"},
		{"poder inválido", nil, "/pokemons/sorteio?tier=raro&poder=abc", http.StatusBadRequest, "requisicao_invalida"},
		{"erro desconhecido", errors.New("banco caiu"), "/pokemons/25", http.StatusInternalServerError, "erro_interno"},
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			h := Novo(servicoFalso{pokemon: dominio.Pokemon{ID: 727, Nome: "incineroar"}, erro: c.erro}, log)
			rec := httptest.NewRecorder()
			h.Rotas().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.caminho, nil))

			if rec.Code != c.statusEsp {
				t.Fatalf("status: esperava %d, veio %d (%s)", c.statusEsp, rec.Code, rec.Body)
			}
			if c.codigoEsp != "" {
				var e httpjson.Erro
				if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
					t.Fatalf("corpo de erro não é JSON: %v", err)
				}
				if e.Codigo != c.codigoEsp {
					t.Errorf("código: esperava %q, veio %q", c.codigoEsp, e.Codigo)
				}
			}
		})
	}
}
