# PokéBatalha

Jogo de coleção e batalha de Pokémon feito com **3 microsserviços em Go**. Você abre pacotes (gacha) para montar sua coleção e enfrenta um rival controlado pela CPU em batalhas por turnos.

A raridade de cada Pokémon vem de um **índice de poder** que soma stats, tipagem, habilidade e Fake Out. Por isso stats brutos não bastam: o Incineroar (stats 360, mas Intimidate e Fake Out) é lendário, e o Charizard (stats 368, sem habilidade relevante) é épico.

## Arquitetura

```
                 ┌──────────────────────┐
  cliente ──REST─┤ jogadores   :8082    ├──gRPC──┐
  (cmd/jogar,    │ contas, moedas, gacha│        │
   curl,         └──────────▲───────────┘        ▼
   Postman)                 │ gRPC        ┌──────────────┐  REST   ┌─────────┐
          ──REST─┐          │             │ catalogo     ├────────►│ PokeAPI │
                 ▼          │             │ :8081        │         └─────────┘
          ┌─────────────────┴──┐  gRPC    │ cache + poder│
          │ batalha     :8083  ├─────────►└──────┬───────┘
          │ turnos, habilidades│                 │
          └─────────┬──────────┘                 │
                    │                            │
              batalha_db     jogadores_db   catalogo_db   (mesmo Postgres,
                                                           bancos e usuários separados)
```

| Serviço | REST (host) | gRPC (rede interna) | Banco | Fala com |
|---|---|---|---|---|
| `catalogo` | 8081 | 9090 | `catalogo_db` | PokeAPI (REST externo) |
| `jogadores` | 8082 | 9090 | `jogadores_db` | catalogo (gRPC) |
| `batalha` | 8083 | não expõe | `batalha_db` | catalogo e jogadores (gRPC) |
| `web` (NGINX) | 8080 | — | — | serve `web/` e faz proxy de `/api/*` para os três |

## Como rodar

Precisa de Docker. Na raiz do projeto:

```bash
docker compose up --build -d
docker compose logs -f          # logs em JSON de todos os serviços
```

Depois é só abrir **http://localhost:8080** no navegador para jogar pela interface web.

Para jogar no terminal (precisa de Go 1.24+):

```bash
go run ./cmd/jogar
```

Cada conta começa com 100 moedas, o pacote custa 30 e cada vitória paga 50 (tudo configurável por variável de ambiente no `docker-compose.yml`).

Na primeira vez que sobe, o `catalogo` baixa a Pokédex inteira da PokeAPI para o próprio banco, em segundo plano (leva alguns minutos e só acontece uma vez). Dá para jogar durante esse tempo, só que os primeiros pacotes e batalhas ficam mais lentos. Para desligar isso, use `AQUECER_CACHE: "false"`.

Para desligar: `docker compose down` (ou `docker compose down -v` para apagar os bancos).

## Front-end web

Interface em HTML, CSS e JavaScript puros (sem framework e sem etapa de build), na pasta `web/`. Um container NGINX serve os arquivos e repassa `/api/<servico>/` para cada microsserviço. Front e APIs ficam na mesma origem, então não há CORS e o back-end não precisou mudar.

| Tela | O que faz |
|---|---|
| Login | Cria conta ou entra pelo ID. O ID fica salvo no navegador ("Continuar como #12") |
| Menu | Moedas, contadores do pity e as opções. "Abrir pacote" fica desabilitado sem moedas |
| Gacha | Máquina em pixel art: gira, solta a pokébola, flash na cor do tier e revelação com stats e composição do poder |
| Coleção | Grade de cards com sprite, tier, poder e tipos. Filtros por tier e tipo, ordenação, e detalhes completos ao clicar |
| Montar time | Até 3 Pokémon; a ordem dos cliques define a ordem do time |
| Batalha | Os dois lados com sprite, HP, tipos, habilidade, estágios e o time completo do rival. O log é digitado linha a linha e a tela acompanha (HP descendo, trocas, Protect, desmaios, flash de super efetivo). Teclas 1 a 6 escolhem a ação |

Os sprites vêm do repositório oficial da PokeAPI no GitHub, carregados direto pelo navegador e com fallback se faltar algum.

```
web/
├── index.html, favicon.svg
├── nginx.conf            proxy /api + cache
├── dev-server.mjs        mesmo proxy em Node, para rodar o front sem Docker
├── css/                  reset, tokens (paleta e fontes), componentes, telas, animações
├── js/
│   ├── api.js            fetch com timeout, retry em GET e erros tipados
│   ├── state.js          estado global + pub/sub
│   ├── router.js         rotas por #hash
│   ├── screens/          login, menu, gacha, coleção, batalha (com seleção de time)
│   └── components/       card, HP, badges, diálogo, log, menu de ações, máquina de gacha...
└── assets/img/           pokébola em SVG
```

**Sem Docker** (com os serviços rodando nas portas 8081 a 8083): `node web/dev-server.mjs` e abra http://localhost:8080.

## Endpoints

### catalogo (http://localhost:8081)

| Método | Rota | O que faz |
|---|---|---|
| GET | `/pokemons/{id ou nome}` | Busca na PokeAPI (ou no cache) e devolve com o índice de poder e o tier |
| GET | `/pokemons/sorteio?tier=raro&poder=350` | Pokémon aleatório do tier (com `poder`, entre os mais próximos desse valor) |
| GET | `/tipos/efetividade?ataque=fire&defensor=grass,steel` | Multiplicador de dano |
| GET | `/tiers` | Fórmula do índice de poder, cortes de tier e pesos das habilidades |

### jogadores (http://localhost:8082)

| Método | Rota | Corpo | Sucesso |
|---|---|---|---|
| POST | `/jogadores` | `{"nome": "theo"}` | 201 |
| GET | `/jogadores/{id}` | | 200 (inclui o pity) |
| POST | `/jogadores/{id}/pacotes` | | 201 |
| GET | `/jogadores/{id}/colecao` | | 200 (com stats e poder, do mais forte ao mais fraco) |
| GET | `/gacha/chances` | | 200 |

### batalha (http://localhost:8083)

| Método | Rota | Corpo | Sucesso |
|---|---|---|---|
| POST | `/batalhas` | `{"jogador_id": 1, "time": [727, 6]}` | 201 |
| GET | `/batalhas/{id}` | | 200 |
| POST | `/batalhas/{id}/turnos` | `{"acao": "tipo"}` ou `{"acao": "trocar", "alvo": 2}` | 200 |
| GET | `/habilidades` | | 200 |

Todo erro volta no mesmo formato: `{"codigo": "saldo_insuficiente", "mensagem": "..."}`.

| Status | Quando |
|---|---|
| 400 | JSON inválido, campo desconhecido, ID/tier/ação inválidos, time vazio |
| 404 | Pokémon, jogador ou batalha não existe |
| 409 | Nome em uso, batalha já terminou, dois turnos ou dois pacotes enviados ao mesmo tempo |
| 422 | Saldo insuficiente, Pokémon que não é seu no time |
| 502 | A PokeAPI caiu, deu timeout ou respondeu algo inválido |
| 503 | Outro microsserviço (via gRPC) está fora do ar, ou o catálogo ainda não tem Pokémon daquele tier |
| 500 | Qualquer outra falha (detalhe só no log, nunca para o cliente) |

## Como o jogo funciona

### Índice de poder e tiers

Cada Pokémon tem um **poder**, que é a soma de quatro pesos na mesma escala:

| Peso | Como é calculado |
|---|---|
| Stats | HP + maior ataque (físico ou especial) + média das defesas + velocidade, que é o que a batalha usa |
| Tipagem ofensiva | Cobertura dos golpes do Pokémon (um por tipo) contra os 18 tipos |
| Tipagem defensiva | Dano médio que a combinação de tipos recebe dos 18 tipos (resistências e imunidades valem pontos) |
| Habilidade | Peso da habilidade mais forte entre as implementadas (ex: Huge Power 70, Intimidate 45, Levitate 25) |
| Fake Out | +20 para quem aprende o golpe |

O tier sai do total:

| Tier | Poder | Pokémon na Pokédex |
|---|---|---|
| comum | menos de 300 | ~42% |
| raro | 300 a 369 | ~27% |
| épico | 370 a 429 | ~20% |
| lendário | 430 ou mais | ~10% |

Os cortes foram calibrados com os dados reais dos 1025 Pokémon (CSVs oficiais da PokeAPI). Exemplos:

| Pokémon | Stats | Tipagem | Habilidade | Fake Out | Poder | Tier |
|---|---|---|---|---|---|---|
| Incineroar | 360 | +40 | +45 | +20 | 465 | lendário |
| Charizard | 368 | +27 | 0 | 0 | 395 | épico |
| Kyogre | 455 | +10 | +40 | 0 | 505 | lendário |

Os pesos e cortes ficam em `services/catalogo/internal/dominio/poder.go`.

### Gacha e pity

Cada pacote sorteia o tier (lendário 2%, épico 8%, raro 20%, comum 70%) e depois um Pokémon desse tier. Há duas garantias:

- **Épico ou melhor no 10º pacote seguido sem nenhum.** Nesse caso ainda há 20% de chance de vir lendário.
- **Lendário no 50º pacote seguido sem lendário.**

Os contadores aparecem no menu do jogo e em `GET /jogadores/{id}`.

### Batalha

Time de 1 a 3 Pokémon. Para cada um seu, o rival recebe um do **mesmo tier e de poder parecido**, então um lendário enfrenta um lendário de força equivalente. Tudo é calculado no nível 50. A cada turno você escolhe:

| Ação | Poder | Detalhes |
|---|---|---|
| Ataque Físico | 50 | Sem tipo, sempre dano neutro |
| Ataque de Tipo | 70 | Um para cada tipo do Pokémon (dois tipos = dois golpes = mais cobertura), de 0x a 4x |
| Fake Out | 40 | Prioridade +3, faz o alvo recuar. Só no primeiro turno em campo, e só para quem aprende |
| Protect | — | Prioridade +4, bloqueia o golpe do rival (inclusive Fake Out). Não pode repetir seguido |
| Trocar | — | Acontece antes de tudo. Quem sai perde os estágios (ex: o -1 do Intimidate), quem entra ativa a habilidade e toma o golpe do turno |

Quando um Pokémon seu desmaia, você escolhe quem entra, e essa troca não gasta turno.

Dano: `(22 × poder × ataque / defesa) / 50 + 2`, multiplicado por efetividade, clima, habilidades e um fator aleatório de 0.85 a 1.0.

**IA do rival.** Ela estima o dano de cada golpe com a mesma fórmula e:
- usa Fake Out quando pode;
- usa Protect quando você ameaça Fake Out;
- troca de Pokémon quando o confronto está ruim e tem alguém melhor no banco;
- às vezes se protege quando vai ser nocauteado antes de agir;
- escolhe o substituto com melhor confronto quando um dos dela desmaia.

Numa simulação de 30 partidas com o mesmo time:

| Jogador | Vitórias |
|---|---|
| Escolhendo ações aleatórias | 7% |
| Escolhendo golpes pela efetividade e trocando em desvantagem | 53% |

**Habilidades.** Cerca de 35 habilidades reais estão implementadas de forma simplificada. A lista completa com descrição está em `GET /habilidades` e em `services/batalha/internal/dominio/habilidades.go`.

## Estrutura de pastas

```
cmd/jogar/                     cliente de terminal
proto/                         contratos gRPC (.proto)
gen/                           código gerado pelo protoc
pkg/                           compartilhado: config, logs, JSON, banco, ciclo de vida
services/<servico>/
    cmd/main.go                monta as dependências e sobe os servidores
    internal/config/           variáveis de ambiente
    internal/dominio/          tipos, erros e regras puras
    internal/servico/          casos de uso (lógica de negócio)
    internal/repositorio/      acesso ao Postgres + schema.sql
    internal/httpapi/          handlers REST
    internal/grpcapi/          servidor gRPC
    internal/pokeapi/          (catalogo) client da API externa
    internal/catalogo/         (jogadores) client gRPC do catálogo
    internal/clientes/         (batalha) clients gRPC de catálogo e jogadores
deploy/postgres/init.sql       cria um banco e um usuário por serviço
```

## Onde cada critério da avaliação aparece

| Critério | Onde |
|---|---|
| 1.1 Pastas separadas | Cada serviço separa handlers (`httpapi`, `grpcapi`), negócio (`servico`, `dominio`), dados (`repositorio`) e clients externos (`pokeapi`, `catalogo`, `clientes`) |
| 1.2 Config por ambiente | `internal/config` de cada serviço + `docker-compose.yml` |
| 1.3 go.mod limpo | Módulo `pokebatalha`, só três dependências diretas: pgx, grpc, protobuf |
| 2.1 Banco próprio | `deploy/postgres/init.sql`: um banco e um usuário por serviço, sem CONNECT nos bancos dos outros |
| 2.2 Repositório isolado | `internal/repositorio`, usado pelo serviço através de interface |
| 2.3 Não encontrado ≠ infra | `pgx.ErrNoRows` vira erro de domínio; qualquer outro erro do banco vira 500 |
| 3.1 API externa | `catalogo/internal/pokeapi/cliente.go`: context com timeout, `defer resp.Body.Close()`, confere `StatusCode` |
| 3.2 / 3.3 REST e JSON | Tabelas acima; `pkg/httpjson` rejeita campos desconhecidos |
| 4.1 Negócio ≠ infra | Erros de negócio e de infra separados em `dominio/erros`; `responderErro` em cada `httpapi`; códigos gRPC traduzidos nos clients |
| 5.1 Erros idiomáticos | `if err != nil` + `%w` + `errors.Is`. Só há dois descartes com `_`, ambos comentados e intencionais: escrever a resposta HTTP (a conexão pode ter caído) e esvaziar o corpo de uma resposta de erro da PokeAPI |
| 5.2 Interfaces | Repositório, clients externos e sorteador são interfaces, trocadas por versões falsas nos testes |
| 5.3 Nomes consistentes | Português em todo o código do projeto |
| 6.1 Testes | `go test ./...`: índice de poder, pity, motor de batalha e IA, client da PokeAPI (httptest), handlers, clients gRPC (bufconn) |
| gRPC entre serviços | `proto/`, `grpcapi/` e `clientes/` |
| Observabilidade | `pkg/observabilidade`: logs JSON (slog) de cada requisição HTTP e chamada gRPC |

## Testes

```bash
go test ./...
```

Os testes não precisam de banco, Docker nem internet: a PokeAPI é simulada com `httptest` e o gRPC roda em memória com `bufconn`.

## Regenerar o código gRPC

Precisa de `protoc`, `protoc-gen-go` (v1.34.2) e `protoc-gen-go-grpc` (v1.5.1) no PATH:

```bash
make proto
```

## Problemas comuns

- **`checksum mismatch` ou erro no `go.sum`**: rode `rm go.sum && go mod tidy`.
- **Primeira busca de um Pokémon demora**: é a ida à PokeAPI. Depois fica no cache do `catalogo_db`.
- **Porta em uso**: mude o lado esquerdo do `ports:` no `docker-compose.yml` (ex: `"9081:8080"`) e use `JOGADORES_URL`/`BATALHA_URL` no cliente.
- **Mudei o front e o navegador mostra a versão antiga**: aperte Ctrl+F5. O NGINX já manda o navegador conferir JS e CSS a cada carregamento, então isso só acontece com cache antigo.
- **Página abre, mas tudo dá "servidor indisponível"**: confira com `docker compose ps` se os três serviços estão de pé. O NGINX reencontra os containers sozinho quando eles são recriados.
