package pokeapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pokebatalha/services/catalogo/internal/dominio"
)

const jsonIncineroar = `{
  "id": 727, "name": "incineroar",
  "types": [{"slot": 2, "type": {"name": "dark"}}, {"slot": 1, "type": {"name": "fire"}}],
  "abilities": [
    {"ability": {"name": "blaze"}, "slot": 1, "is_hidden": false},
    {"ability": {"name": "intimidate"}, "slot": 3, "is_hidden": true}
  ],
  "stats": [
    {"base_stat": 95, "stat": {"name": "hp"}},
    {"base_stat": 115, "stat": {"name": "attack"}},
    {"base_stat": 90, "stat": {"name": "defense"}},
    {"base_stat": 80, "stat": {"name": "special-attack"}},
    {"base_stat": 90, "stat": {"name": "special-defense"}},
    {"base_stat": 60, "stat": {"name": "speed"}}
  ],
  "moves": [{"move": {"name": "flare-blitz"}}, {"move": {"name": "fake-out"}}]
}`

func TestBuscarPokemon_Sucesso(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pokemon/incineroar" {
			t.Errorf("caminho inesperado: %s", r.URL.Path)
		}
		w.Write([]byte(jsonIncineroar))
	}))
	defer srv.Close()

	p, err := NovoCliente(srv.URL, time.Second).BuscarPokemon(context.Background(), "incineroar")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if p.ID != 727 || p.Nome != "incineroar" {
		t.Errorf("identificação errada: %+v", p)
	}
	if len(p.Tipos) != 2 || p.Tipos[0] != "fire" || p.Tipos[1] != "dark" {
		t.Errorf("tipos deveriam vir ordenados por slot, veio %v", p.Tipos)
	}
	if !p.AprendeFakeOut {
		t.Error("deveria detectar que aprende fake-out")
	}
	if p.Status.Ataque != 115 || p.Status.Velocidade != 60 {
		t.Errorf("status errados: %+v", p.Status)
	}
}

func TestBuscarPokemon_404EhErroDeNegocio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()

	_, err := NovoCliente(srv.URL, time.Second).BuscarPokemon(context.Background(), "naoexiste")
	if !errors.Is(err, dominio.ErrPokemonNaoEncontrado) {
		t.Fatalf("esperava ErrPokemonNaoEncontrado, veio %v", err)
	}
	if errors.Is(err, dominio.ErrServicoExternoIndisponivel) {
		t.Fatal("404 não deveria ser tratado como erro de infraestrutura")
	}
}

func TestBuscarPokemon_500EhErroDeInfra(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NovoCliente(srv.URL, time.Second).BuscarPokemon(context.Background(), "pikachu")
	if !errors.Is(err, dominio.ErrServicoExternoIndisponivel) {
		t.Fatalf("esperava ErrServicoExternoIndisponivel, veio %v", err)
	}
}

func TestBuscarPokemon_TimeoutEhErroDeInfra(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	inicio := time.Now()
	_, err := NovoCliente(srv.URL, 50*time.Millisecond).BuscarPokemon(context.Background(), "pikachu")
	if !errors.Is(err, dominio.ErrServicoExternoIndisponivel) {
		t.Fatalf("esperava ErrServicoExternoIndisponivel, veio %v", err)
	}
	if time.Since(inicio) > time.Second {
		t.Fatal("o timeout do context não foi respeitado")
	}
}

func TestBuscarPokemon_JSONQuebradoEhErroDeInfra(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"id": "isso não é número"`))
	}))
	defer srv.Close()

	_, err := NovoCliente(srv.URL, time.Second).BuscarPokemon(context.Background(), "pikachu")
	if !errors.Is(err, dominio.ErrServicoExternoIndisponivel) {
		t.Fatalf("esperava ErrServicoExternoIndisponivel, veio %v", err)
	}
}

func TestBuscarTipo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"name": "fire", "damage_relations": {
			"double_damage_to": [{"name": "grass"}, {"name": "steel"}],
			"half_damage_to": [{"name": "water"}],
			"no_damage_to": []}}`))
	}))
	defer srv.Close()

	rel, err := NovoCliente(srv.URL, time.Second).BuscarTipo(context.Background(), "fire")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got := rel.Multiplicador([]string{"grass", "steel"}); got != 4 {
		t.Errorf("fire contra grass/steel deveria ser 4x, veio %v", got)
	}
}
