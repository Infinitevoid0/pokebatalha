// Package httpapi expõe o serviço de jogadores via REST.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"pokebatalha/pkg/httpjson"
	"pokebatalha/services/jogadores/internal/dominio"
	"pokebatalha/services/jogadores/internal/servico"
)

// Servico é o que o handler precisa da camada de negócio.
type Servico interface {
	CriarJogador(ctx context.Context, nome string) (dominio.Jogador, error)
	BuscarJogador(ctx context.Context, id int64) (dominio.Jogador, error)
	Colecao(ctx context.Context, id int64) (servico.Colecao, error)
	AbrirPacote(ctx context.Context, id int64) (servico.ResultadoPacote, error)
}

// Handler agrupa as rotas REST de jogadores.
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
	mux.HandleFunc("GET /saude", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.Escrever(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /gacha/chances", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.Escrever(w, http.StatusOK, map[string]any{
			"chances":              dominio.Chances,
			"pity_epico_ou_melhor": dominio.PityEpico,
			"pity_lendario":        dominio.PityLendario,
		})
	})
	mux.HandleFunc("POST /jogadores", h.criar)
	mux.HandleFunc("GET /jogadores/{id}", h.buscar)
	mux.HandleFunc("GET /jogadores/{id}/colecao", h.colecao)
	mux.HandleFunc("POST /jogadores/{id}/pacotes", h.abrirPacote)
	return mux
}

type requisicaoCriar struct {
	Nome string `json:"nome"`
}

// POST /jogadores {"nome": "theo"}
func (h *Handler) criar(w http.ResponseWriter, r *http.Request) {
	var req requisicaoCriar
	if err := httpjson.Ler(r, &req); err != nil {
		h.responderErro(w, r, err)
		return
	}
	j, err := h.svc.CriarJogador(r.Context(), req.Nome)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusCreated, comPity(j))
}

func (h *Handler) buscar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	j, err := h.svc.BuscarJogador(r.Context(), id)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusOK, comPity(j))
}

// respostaJogador mostra o jogador com o pity já explicado.
type respostaJogador struct {
	dominio.Jogador
	Pity dominio.InfoPity `json:"pity"`
}

func comPity(j dominio.Jogador) respostaJogador {
	return respostaJogador{Jogador: j, Pity: j.Pity.Info()}
}

func (h *Handler) colecao(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	c, err := h.svc.Colecao(r.Context(), id)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusOK, c)
}

// POST /jogadores/{id}/pacotes — 201 porque cria um item novo na coleção.
func (h *Handler) abrirPacote(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	res, err := h.svc.AbrirPacote(r.Context(), id)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	httpjson.Escrever(w, http.StatusCreated, res)
}

var errIDInvalido = errors.New("id inválido")

func idDaRota(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("%w: %q", errIDInvalido, r.PathValue("id"))
	}
	return id, nil
}

// responderErro traduz o erro para o status HTTP certo.
func (h *Handler) responderErro(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	// Erros de negócio.
	case errors.Is(err, httpjson.ErrCorpoInvalido),
		errors.Is(err, errIDInvalido),
		errors.Is(err, dominio.ErrNomeInvalido):
		httpjson.EscreverErro(w, http.StatusBadRequest, "requisicao_invalida", err.Error())
	case errors.Is(err, dominio.ErrJogadorNaoEncontrado):
		httpjson.EscreverErro(w, http.StatusNotFound, "jogador_nao_encontrado", err.Error())
	case errors.Is(err, dominio.ErrNomeEmUso):
		httpjson.EscreverErro(w, http.StatusConflict, "nome_em_uso", err.Error())
	case errors.Is(err, dominio.ErrCompraSimultanea):
		httpjson.EscreverErro(w, http.StatusConflict, "compra_simultanea", err.Error())
	case errors.Is(err, dominio.ErrSaldoInsuficiente):
		// 422: a requisição é válida, mas a regra de negócio impede.
		httpjson.EscreverErro(w, http.StatusUnprocessableEntity, "saldo_insuficiente", err.Error())

	// Erro de infraestrutura: o catálogo (ou a PokeAPI atrás dele) não respondeu.
	case errors.Is(err, dominio.ErrServicoIndisponivel):
		h.log.Error("dependência indisponível", "caminho", r.URL.Path, "erro", err)
		httpjson.EscreverErro(w, http.StatusServiceUnavailable, "servico_indisponivel",
			"o catálogo está indisponível, tente de novo em instantes")

	default:
		h.log.Error("erro interno", "caminho", r.URL.Path, "erro", err)
		httpjson.EscreverErro(w, http.StatusInternalServerError, "erro_interno", "erro interno do servidor")
	}
}
