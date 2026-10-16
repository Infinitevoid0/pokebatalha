// configuração de jogadores via ambiente.
package config

import (
	"time"

	"pokebatalha/pkg/config"
)

type Config struct {
	PortaHTTP        string
	PortaGRPC        string
	DatabaseURL      string
	CatalogoGRPCAddr string
	GRPCTimeout      time.Duration
	MoedasIniciais   int
	PrecoPacote      int
}

func Carregar() (Config, error) {
	dbURL, err := config.Obrigatorio("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	// o sorteio de comum pode levar várias idas à PokeAPI.
	timeout, err := config.Duracao("GRPC_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	moedas, err := config.Inteiro("MOEDAS_INICIAIS", 100)
	if err != nil {
		return Config{}, err
	}
	preco, err := config.Inteiro("PRECO_PACOTE", 30)
	if err != nil {
		return Config{}, err
	}
	return Config{
		PortaHTTP:        config.Texto("PORTA_HTTP", "8080"),
		PortaGRPC:        config.Texto("PORTA_GRPC", "9090"),
		DatabaseURL:      dbURL,
		CatalogoGRPCAddr: config.Texto("CATALOGO_GRPC_ADDR", "localhost:9090"),
		GRPCTimeout:      timeout,
		MoedasIniciais:   moedas,
		PrecoPacote:      preco,
	}, nil
}
