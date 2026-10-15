// Package config carrega a configuração do catálogo a partir do ambiente.
package config

import (
	"time"

	"pokebatalha/pkg/config"
)

// Config reúne tudo que o serviço lê do ambiente.
type Config struct {
	PortaHTTP      string
	PortaGRPC      string
	DatabaseURL    string
	PokeAPIURL     string
	PokeAPITimeout time.Duration
	AquecerCache   bool
}

// Carregar lê as variáveis de ambiente.
func Carregar() (Config, error) {
	dbURL, err := config.Obrigatorio("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	timeout, err := config.Duracao("POKEAPI_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	return Config{
		PortaHTTP:      config.Texto("PORTA_HTTP", "8080"),
		PortaGRPC:      config.Texto("PORTA_GRPC", "9090"),
		DatabaseURL:    dbURL,
		PokeAPIURL:     config.Texto("POKEAPI_URL", "https://pokeapi.co/api/v2"),
		PokeAPITimeout: timeout,
		AquecerCache:   config.Texto("AQUECER_CACHE", "true") == "true",
	}, nil
}
