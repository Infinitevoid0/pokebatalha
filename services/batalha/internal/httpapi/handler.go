// expõe a batalha via REST.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"pokebatalha/pkg/httpjson"
	"pokebatalha/services/batalha/internal/dominio"
)

// o que o handler precisa do negócio.
type Servico interface {
	CriarBatalha(ctx context.Context, jogadorID int64, pokemonIDs []int) (*dominio.Batalha, error)
	Buscar(ctx context.Context, id int64) (*dominio.Batalha, error)
	Jogar(ctx context.Context, id int64, acao string, alvo int) (*dominio.Batalha, error)
}

// rotas REST da batalha.
type Handler struct {
	svc Servico
	log *slog.Logger
}

func Novo(svc Servico, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Rotas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /saude", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.Escrever(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /habilidades", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.Escrever(w, http.StatusOK, dominio.Habilidades)
	})
	mux.HandleFunc("POST /batalhas", h.criar)
	mux.HandleFunc("GET /batalhas/{id}", h.buscar)
	mux.HandleFunc("POST /batalhas/{id}/turnos", h.jogar)
	return mux
}

// estado da batalha + ações disponíveis.
type resposta struct {
	*dominio.Batalha
	AcoesDisponiveis  []dominio.Acao `json:"acoes_disponiveis"`
	TrocasDisponiveis []int          `json:"trocas_disponiveis"`
}

func responder(w http.ResponseWriter, status int, b *dominio.Batalha) {
	httpjson.Escrever(w, status, resposta{
		Batalha:           b,
		AcoesDisponiveis:  b.AcoesDoJogador(),
		TrocasDisponiveis: b.Jogador.TrocasDisponiveis(),
	})
}

type requisicaoCriar struct {
	JogadorID int64 `json:"jogador_id"`
	Time      []int `json:"time"`
}

// cria batalha: POST /batalhas {"jogador_id": 1, "time": [727, 6]}
func (h *Handler) criar(w http.ResponseWriter, r *http.Request) {
	var req requisicaoCriar
	if err := httpjson.Ler(r, &req); err != nil {
		h.responderErro(w, r, err)
		return
	}
	b, err := h.svc.CriarBatalha(r.Context(), req.JogadorID, req.Time)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	responder(w, http.StatusCreated, b)
}

func (h *Handler) buscar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	b, err := h.svc.Buscar(r.Context(), id)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	responder(w, http.StatusOK, b)
}

type requisicaoTurno struct {
	Acao string `json:"acao"`
	Alvo int    `json:"alvo"` // índice no time, só para "trocar"
}

// joga um turno: POST /batalhas/{id}/turnos {"acao": "tipo"} ou {"acao": "trocar", "alvo": 2}
func (h *Handler) jogar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	var req requisicaoTurno
	if err := httpjson.Ler(r, &req); err != nil {
		h.responderErro(w, r, err)
		return
	}
	b, err := h.svc.Jogar(r.Context(), id, req.Acao, req.Alvo)
	if err != nil {
		h.responderErro(w, r, err)
		return
	}
	responder(w, http.StatusOK, b)
}

var errIDInvalido = errors.New("id inválido")

func idDaRota(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("%w: %q", errIDInvalido, r.PathValue("id"))
	}
	return id, nil
}

// traduz o erro no status HTTP certo.
func (h *Handler) responderErro(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	// erros de negócio.
	case errors.Is(err, httpjson.ErrCorpoInvalido),
		errors.Is(err, errIDInvalido),
		errors.Is(err, dominio.ErrTimeInvalido),
		errors.Is(err, dominio.ErrAcaoInvalida):
		httpjson.EscreverErro(w, http.StatusBadRequest, "requisicao_invalida", err.Error())
	case errors.Is(err, dominio.ErrBatalhaNaoEncontrada):
		httpjson.EscreverErro(w, http.StatusNotFound, "batalha_nao_encontrada", err.Error())
	case errors.Is(err, dominio.ErrJogadorNaoEncontrado):
		httpjson.EscreverErro(w, http.StatusNotFound, "jogador_nao_encontrado", err.Error())
	case errors.Is(err, dominio.ErrPokemonNaoEncontrado):
		httpjson.EscreverErro(w, http.StatusNotFound, "pokemon_nao_encontrado", err.Error())
	case errors.Is(err, dominio.ErrBatalhaEncerrada):
		httpjson.EscreverErro(w, http.StatusConflict, "batalha_encerrada", err.Error())
	case errors.Is(err, dominio.ErrConflitoDeVersao):
		httpjson.EscreverErro(w, http.StatusConflict, "conflito", err.Error())
	case errors.Is(err, dominio.ErrPokemonNaoPertence):
		httpjson.EscreverErro(w, http.StatusUnprocessableEntity, "pokemon_nao_pertence", err.Error())

	// infra: catálogo ou jogadores fora do ar.
	case errors.Is(err, dominio.ErrServicoIndisponivel):
		h.log.Error("dependência indisponível", "caminho", r.URL.Path, "erro", err)
		httpjson.EscreverErro(w, http.StatusServiceUnavailable, "servico_indisponivel",
			"um serviço interno está indisponível, tente de novo em instantes")

	default:
		h.log.Error("erro interno", "caminho", r.URL.Path, "erro", err)
		httpjson.EscreverErro(w, http.StatusInternalServerError, "erro_interno", "erro interno do servidor")
	}
}
