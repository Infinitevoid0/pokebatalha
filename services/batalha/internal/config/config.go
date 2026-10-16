// configuração da batalha via ambiente.
package config

import (
	"time"

	"pokebatalha/pkg/config"
)

type Config struct {
	PortaHTTP         string
	DatabaseURL       string
	CatalogoGRPCAddr  string
	JogadoresGRPCAddr string
	GRPCTimeout       time.Duration
	RecompensaVitoria int
}

func Carregar() (Config, error) {
	dbURL, err := config.Obrigatorio("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	timeout, err := config.Duracao("GRPC_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	recompensa, err := config.Inteiro("RECOMPENSA_VITORIA", 50)
	if err != nil {
		return Config{}, err
	}
	return Config{
		PortaHTTP:         config.Texto("PORTA_HTTP", "8080"),
		DatabaseURL:       dbURL,
		CatalogoGRPCAddr:  config.Texto("CATALOGO_GRPC_ADDR", "localhost:9090"),
		JogadoresGRPCAddr: config.Texto("JOGADORES_GRPC_ADDR", "localhost:9091"),
		GRPCTimeout:       timeout,
		RecompensaVitoria: recompensa,
	}, nil
}
