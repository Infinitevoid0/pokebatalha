// leitura de variáveis de ambiente com padrão e validação.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// valor da variável ou o padrão.
func Texto(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

// variável como int; erro se inválida.
func Inteiro(chave string, padrao int) (int, error) {
	v := os.Getenv(chave)
	if v == "" {
		return padrao, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("variável %s deve ser um inteiro: %w", chave, err)
	}
	return n, nil
}

// variável como duração (ex: "5s").
func Duracao(chave string, padrao time.Duration) (time.Duration, error) {
	v := os.Getenv(chave)
	if v == "" {
		return padrao, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("variável %s deve ser uma duração (ex: 5s): %w", chave, err)
	}
	return d, nil
}

// erro se a variável não existir.
func Obrigatorio(chave string) (string, error) {
	v := os.Getenv(chave)
	if v == "" {
		return "", fmt.Errorf("variável de ambiente %s é obrigatória", chave)
	}
	return v, nil
}
