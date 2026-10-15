// Package httpapi expõe o catálogo via REST.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"pokebatalha/pkg/httpjson"
	"pokebatalha/services/catalogo/internal/dominio"
)

// Servico é o que o handler precisa da camada de negócio.
type Servico interface {
	BuscarPorID(ctx context.Context, id int) (dominio.Pokemon, error)
	BuscarPorNome(ctx context.Context, nome string) (dominio.Pokemon, error)
	Sortear(ctx context.Context, tier dominio.Tier, poderReferencia int) (dominio.Pokemon, error)
	Efetividade(ctx context.Context, ataque string, defensor []string) (float64, error)
	Regras() dominio.Regras
}

// Handler agrupa as rotas REST do catálogo.
type Handler struct {
	svc Servico
	log *slog.Logger
}

// Novo cria o handler.
func Novo(svc Servico, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Rotas registra os endpoints.
func (h *Handler) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /saude", h.saude)
	mux.HandleFunc("GET /pokemons/sorteio", h.sortear)
	mux.HandleFunc("GET /pokemons/{idOuNome}", h.buscar)
	mux.HandleFunc("GET /tipos/efetividade", h.efetividade)
	mux.HandleFunc("GET /tiers", h.listarTiers)
	return mux
}

func (h *Handler) saude(w http.ResponseWriter, _ *http.Request) {
	httpjson.Escrever(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /pokemons/{idOuNome} — aceita número (727) ou nome (incineroar).
func (h *Handler) buscar(w http.ResponseWriter, r *http.Request) {
	chave := r.PathValue("idOuNome")

	var (
		p   dominio.Pokemon
		err error
	)
	if id, errConv := strconv.Atoi(chave); errConv == nil {
		p, err = h.svc.BuscarPorID(r.Context(), id)
	} else {
		p, err = h.svc.BuscarPorNome(r.Context(), chave)
	}
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusOK, p)
}

// GET /pokemons/sorteio?tier=raro&poder=350 (poder é opcional)
func (h *Handler) sortear(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tier, err := dominio.ParseTier(q.Get("tier"))
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	poder := 0
	if v := q.Get("poder"); v != "" {
		if poder, err = strconv.Atoi(v); err != nil || poder < 0 {
			h.responderErro(w, r, fmt.Errorf("%w: poder deve ser um número positivo", dominio.ErrIDInvalido))
			return
		}
	}
	p, err := h.svc.Sortear(r.Context(), tier, poder)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusOK, p)
}

// GET /tipos/efetividade?ataque=fire&defensor=grass,steel
func (h *Handler) efetividade(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ataque := q.Get("ataque")
	var defensor []string
	if d := q.Get("defensor"); d != "" {
		defensor = strings.Split(d, ",")
	}

	m, err := h.svc.Efetividade(r.Context(), ataque, defensor)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusOK, map[string]any{
		"ataque":        ataque,
		"defensor":      defensor,
		"multiplicador": m,
	})
}

// GET /tiers — fórmula do índice de poder e cortes de cada tier.
func (h *Handler) listarTiers(w http.ResponseWriter, _ *http.Request) {
	httpjson.Escrever(w, http.StatusOK, h.svc.Regras())
}

// responderErro traduz o erro da camada de negócio para o status HTTP certo.
func (h *Handler) responderErro(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	// Erro de negócio: o cliente pediu algo inválido ou inexistente.
	case errors.Is(err, dominio.ErrIDInvalido),
		errors.Is(err, dominio.ErrTierInvalido),
		errors.Is(err, dominio.ErrTipoInvalido):
		httpjson.EscreverErro(w, http.StatusBadRequest, "requisicao_invalida", err.Error())
	case errors.Is(err, dominio.ErrPokemonNaoEncontrado):
		httpjson.EscreverErro(w, http.StatusNotFound, "pokemon_nao_encontrado", err.Error())

	// O cache ainda não tem Pokémon desse tier: vale tentar de novo logo.
	case errors.Is(err, dominio.ErrCatalogoAquecendo):
		httpjson.EscreverErro(w, http.StatusServiceUnavailable, "catalogo_aquecendo", err.Error())

	// Erro de infraestrutura: a PokeAPI caiu, deu timeout ou respondeu lixo.
	case errors.Is(err, dominio.ErrServicoExternoIndisponivel):
		h.log.Error("falha na PokeAPI", "caminho", r.URL.Path, "erro", err)
		httpjson.EscreverErro(w, http.StatusBadGateway, "servico_externo_indisponivel",
			"a PokeAPI não respondeu, tente de novo em instantes")

	// Qualquer outra coisa (banco fora do ar, bug): não vaza detalhe para o cliente.
	default:
		h.log.Error("erro interno", "caminho", r.URL.Path, "erro", err)
		httpjson.EscreverErro(w, http.StatusInternalServerError, "erro_interno", "erro interno do servidor")
	}
}
