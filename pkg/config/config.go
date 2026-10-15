// Package config lê variáveis de ambiente com valores padrão e validação.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Texto devolve a variável de ambiente ou o valor padrão se ela estiver vazia.
func Texto(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

// Inteiro devolve a variável como int, ou erro se ela tiver um valor inválido.
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

// Duracao devolve a variável como time.Duration (ex: "5s", "300ms").
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

// Obrigatorio devolve a variável ou erro se ela não estiver definida.
func Obrigatorio(chave string) (string, error) {
	v := os.Getenv(chave)
	if v == "" {
		return "", fmt.Errorf("variável de ambiente %s é obrigatória", chave)
	}
	return v, nil
}
