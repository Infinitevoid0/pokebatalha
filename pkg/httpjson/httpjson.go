// leitura e escrita de JSON padronizadas.
package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const tamanhoMaximoCorpo = 1 << 20 // 1 MB

// formato padrão de erro da API.
type Erro struct {
	Codigo   string `json:"codigo"`
	Mensagem string `json:"mensagem"`
}

// escreve v em JSON com o status.
func Escrever(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // a conexão pode ter caído; nada a fazer
}

func EscreverErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	Escrever(w, status, Erro{Codigo: codigo, Mensagem: mensagem})
}

// corpo com JSON malformado, campo desconhecido ou corpo grande demais.
var ErrCorpoInvalido = errors.New("corpo da requisição inválido")

// decodifica o JSON e recusa campos desconhecidos.
func Ler(r *http.Request, destino any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, tamanhoMaximoCorpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		return fmt.Errorf("%w: %v", ErrCorpoInvalido, err)
	}
	return nil
}
