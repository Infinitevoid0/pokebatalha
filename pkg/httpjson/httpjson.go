// Package httpjson padroniza a leitura e escrita de JSON nos handlers HTTP.
package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const tamanhoMaximoCorpo = 1 << 20 // 1 MB

// Erro é o formato de toda resposta de erro da API.
type Erro struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

// Escrever serializa v como JSON com o status informado.
func Escrever(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // a conexão pode ter caído; não há o que fazer
}

// EscreverErro responde com o formato padrão de erro.
func EscreverErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	Escrever(w, status, Erro{Codigo: codigo, Mensagem: mensagem})
}

// ErrCorpoInvalido indica JSON malformado, campos desconhecidos ou corpo grande demais.
var ErrCorpoInvalido = errors.New("corpo da requisição inválido")

// Ler decodifica o corpo JSON em destino, rejeitando campos desconhecidos.
func Ler(r *http.Request, destino any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, tamanhoMaximoCorpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		return fmt.Errorf("%w: %v", ErrCorpoInvalido, err)
	}
	return nil
}
