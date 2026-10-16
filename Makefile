# requer protoc, protoc-gen-go e protoc-gen-go-grpc no PATH:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

.PHONY: proto test up down jogar web

proto:
	protoc --proto_path=proto \
		--go_out=. --go_opt=module=pokebatalha \
		--go-grpc_out=. --go-grpc_opt=module=pokebatalha \
		catalogo/v1/catalogo.proto jogadores/v1/jogadores.proto

test:
	go test ./...

up:
	docker compose up --build -d

down:
	docker compose down

jogar:
	go run ./cmd/jogar

# front sem Docker (proxy de /api para as portas 8081-8083).
web:
	node web/dev-server.mjs
