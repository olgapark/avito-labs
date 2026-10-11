-include .env
export

.PHONY: generate migrate-up migrate-down migrate-status run test build

build:
	go build -o bin/tripgo ./cmd/server

run:
	go run ./cmd/server

test:
	go test -race ./...

generate:
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go api/trip-service.openapi.yaml

migrate-up:
	go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	go tool goose -dir migrations postgres "$$DATABASE_URL" down

migrate-status:
	go tool goose -dir migrations postgres "$$DATABASE_URL" status
