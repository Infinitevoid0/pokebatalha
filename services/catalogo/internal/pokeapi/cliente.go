// client REST da PokeAPI: timeout, fecha o Body e confere o StatusCode.
package pokeapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"pokebatalha/services/catalogo/internal/dominio"
)

// /pokemon traz todos os golpes, então a resposta pode ser grande.
const tamanhoMaximoResposta = 10 << 20 // 10 MB

// chamadas HTTP para a PokeAPI.
type Cliente struct {
	urlBase string
	http    *http.Client
	timeout time.Duration
}

// o timeout vale por chamada.
func NovoCliente(urlBase string, timeout time.Duration) *Cliente {
	return &Cliente{
		urlBase: strings.TrimRight(urlBase, "/"),
		http:    &http.Client{},
		timeout: timeout,
	}
}

// quem chama traduz para o erro de negócio certo.
var errNaoEncontrado = errors.New("recurso não existe na PokeAPI")

// faz GET em urlBase+caminho e decodifica o JSON.
func (c *Cliente) obterJSON(ctx context.Context, caminho string, destino any) error {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.urlBase+caminho, nil)
	if err != nil {
		return fmt.Errorf("montar requisição para a PokeAPI: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pokebatalha-catalogo/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		// timeout, DNS, conexão recusada: infraestrutura.
		return fmt.Errorf("%w: PokeAPI: %v", dominio.ErrServicoExternoIndisponivel, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return errNaoEncontrado
	case resp.StatusCode != http.StatusOK:
		// descarta o corpo para reaproveitar a conexão.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, tamanhoMaximoResposta))
		return fmt.Errorf("%w: PokeAPI respondeu status %d", dominio.ErrServicoExternoIndisponivel, resp.StatusCode)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, tamanhoMaximoResposta)).Decode(destino); err != nil {
		return fmt.Errorf("%w: resposta inválida da PokeAPI: %v", dominio.ErrServicoExternoIndisponivel, err)
	}
	return nil
}

type nomeado struct {
	Name string `json:"name"`
}

type respostaPokemon struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Types []struct {
		Slot int     `json:"slot"`
		Type nomeado `json:"type"`
	} `json:"types"`
	Abilities []struct {
		Ability nomeado `json:"ability"`
		Slot    int     `json:"slot"`
	} `json:"abilities"`
	Stats []struct {
		BaseStat int     `json:"base_stat"`
		Stat     nomeado `json:"stat"`
	} `json:"stats"`
	Moves []struct {
		Move nomeado `json:"move"`
	} `json:"moves"`
}

// busca /pokemon/{idOuNome}.
func (c *Cliente) BuscarPokemon(ctx context.Context, idOuNome string) (dominio.Pokemon, error) {
	var r respostaPokemon
	if err := c.obterJSON(ctx, "/pokemon/"+idOuNome, &r); err != nil {
		if errors.Is(err, errNaoEncontrado) {
			return dominio.Pokemon{}, fmt.Errorf("%w: %q", dominio.ErrPokemonNaoEncontrado, idOuNome)
		}
		return dominio.Pokemon{}, err
	}
	return r.paraDominio(), nil
}

func (r respostaPokemon) paraDominio() dominio.Pokemon {
	sort.Slice(r.Types, func(i, j int) bool { return r.Types[i].Slot < r.Types[j].Slot })
	sort.Slice(r.Abilities, func(i, j int) bool { return r.Abilities[i].Slot < r.Abilities[j].Slot })

	p := dominio.Pokemon{ID: r.ID, Nome: r.Name, Tipos: []string{}, Habilidades: []string{}}
	for _, t := range r.Types {
		p.Tipos = append(p.Tipos, t.Type.Name)
	}
	for _, a := range r.Abilities {
		p.Habilidades = append(p.Habilidades, a.Ability.Name)
	}
	for _, m := range r.Moves {
		if m.Move.Name == "fake-out" {
			p.AprendeFakeOut = true
			break
		}
	}
	for _, s := range r.Stats {
		switch s.Stat.Name {
		case "hp":
			p.Status.HP = s.BaseStat
		case "attack":
			p.Status.Ataque = s.BaseStat
		case "defense":
			p.Status.Defesa = s.BaseStat
		case "special-attack":
			p.Status.AtaqueEspecial = s.BaseStat
		case "special-defense":
			p.Status.DefesaEspecial = s.BaseStat
		case "speed":
			p.Status.Velocidade = s.BaseStat
		}
	}
	return p
}

type respostaTipo struct {
	Name            string `json:"name"`
	DamageRelations struct {
		DoubleDamageTo []nomeado `json:"double_damage_to"`
		HalfDamageTo   []nomeado `json:"half_damage_to"`
		NoDamageTo     []nomeado `json:"no_damage_to"`
	} `json:"damage_relations"`
}

// busca /type/{nome}.
func (c *Cliente) BuscarTipo(ctx context.Context, nome string) (dominio.RelacaoTipo, error) {
	var r respostaTipo
	if err := c.obterJSON(ctx, "/type/"+nome, &r); err != nil {
		if errors.Is(err, errNaoEncontrado) {
			return dominio.RelacaoTipo{}, fmt.Errorf("%w: %q", dominio.ErrTipoInvalido, nome)
		}
		return dominio.RelacaoTipo{}, err
	}
	return dominio.RelacaoTipo{
		Tipo:       r.Name,
		DobroPara:  nomes(r.DamageRelations.DoubleDamageTo),
		MetadePara: nomes(r.DamageRelations.HalfDamageTo),
		NenhumPara: nomes(r.DamageRelations.NoDamageTo),
	}, nil
}

func nomes(lista []nomeado) []string {
	out := make([]string, 0, len(lista))
	for _, n := range lista {
		out = append(out, n.Name)
	}
	return out
}
