-- Banco próprio do catálogo (catalogo_db): cache dos dados da PokeAPI.
CREATE TABLE IF NOT EXISTS pokemons (
    id               INT PRIMARY KEY,
    nome             TEXT NOT NULL UNIQUE,
    tipos            TEXT[] NOT NULL,
    habilidades      TEXT[] NOT NULL,
    aprende_fake_out BOOLEAN NOT NULL,
    hp               INT NOT NULL,
    ataque           INT NOT NULL,
    defesa           INT NOT NULL,
    ataque_especial  INT NOT NULL,
    defesa_especial  INT NOT NULL,
    velocidade       INT NOT NULL,
    atualizado_em    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tipos (
    nome          TEXT PRIMARY KEY,
    dobro_para    TEXT[] NOT NULL,
    metade_para   TEXT[] NOT NULL,
    nenhum_para   TEXT[] NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
