-- Banco próprio do serviço de batalha (batalha_db).
-- O estado da partida é guardado como JSONB; "versao" evita que dois
-- turnos enviados ao mesmo tempo sobrescrevam um ao outro.
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
