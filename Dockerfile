# um Dockerfile para os três serviços; SERVICO escolhe qual compilar.

FROM golang:1.24-alpine AS build
WORKDIR /src

# dependências primeiro, para aproveitar o cache de camadas.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG SERVICO
RUN test -n "$SERVICO" || (echo "defina --build-arg SERVICO=catalogo|jogadores|batalha" && exit 1)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./services/${SERVICO}/cmd

# imagem mínima: só o binário e os certificados (HTTPS da PokeAPI).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
