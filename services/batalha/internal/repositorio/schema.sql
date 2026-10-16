-- batalha_db: estado da partida em JSONB; "versao" impede que dois turnos
-- simultâneos se sobrescrevam.
CREATE TABLE IF NOT EXISTS batalhas (
    id            BIGSERIAL PRIMARY KEY,
    jogador_id    BIGINT NOT NULL,
    status        TEXT NOT NULL,
    estado        JSONB NOT NULL,
    versao        INT NOT NULL DEFAULT 1,
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS batalhas_jogador_idx ON batalhas (jogador_id);
