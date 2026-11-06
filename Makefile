-include .env
export

.PHONY: generate migrate migrate-up migrate-down migrate-status run test test-integration lint build

build:
	go build -o bin/tripgo ./cmd/service

run:
	go run ./cmd/service

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-integration:
	go test -race -tags=integration -count=1 ./internal/repository/...

lint:
	go tool golangci-lint run --config golangci.yml

generate:
	go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go api/trip-service.openapi.yaml
	go generate ./internal/...

migrate: migrate-up

migrate-up:
	go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	go tool goose -dir migrations postgres "$$DATABASE_URL" down

migrate-status:
	go tool goose -dir migrations postgres "$$DATABASE_URL" status
