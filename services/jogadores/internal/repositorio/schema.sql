-- jogadores_db: contas e coleções.
CREATE TABLE IF NOT EXISTS jogadores (
    id        BIGSERIAL PRIMARY KEY,
    nome      TEXT NOT NULL UNIQUE,
    moedas    INT NOT NULL CHECK (moedas >= 0),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS colecao (
    id           BIGSERIAL PRIMARY KEY,
    jogador_id   BIGINT NOT NULL REFERENCES jogadores(id),
    pokemon_id   INT NOT NULL,
    nome         TEXT NOT NULL,
    tier         TEXT NOT NULL,
    tipos        TEXT[] NOT NULL,
    adquirido_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS colecao_jogador_idx ON colecao (jogador_id);

-- pity (ALTER para bancos já existentes).
ALTER TABLE jogadores ADD COLUMN IF NOT EXISTS pacotes_sem_epico INT NOT NULL DEFAULT 0;
ALTER TABLE jogadores ADD COLUMN IF NOT EXISTS pacotes_sem_lendario INT NOT NULL DEFAULT 0;
