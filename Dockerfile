# Um Dockerfile para os três serviços: o argumento SERVICO escolhe qual compilar.
#   docker build --build-arg SERVICO=catalogo -t pokebatalha-catalogo .

FROM golang:1.24-alpine AS build
WORKDIR /src

# Baixa as dependências primeiro para aproveitar o cache de camadas.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG SERVICO
RUN test -n "$SERVICO" || (echo "defina --build-arg SERVICO=catalogo|jogadores|batalha" && exit 1)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./services/${SERVICO}/cmd

# Imagem final mínima: só o binário e os certificados (para HTTPS na PokeAPI).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
