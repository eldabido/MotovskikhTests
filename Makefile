include .env
export $(shell sed 's/=.*//' .env)

VERSION := "-X main.Version=$(shell git rev-parse --short HEAD)_$(shell date -u +%Y-%m-%d_%H:%M:%S)"

default: run

run:
	@go run -ldflags $(VERSION) $(RACE_RUN_FLAG) ./cmd/main/main.go

b: lint test

test:
	@go test ./...

lint:
	@golangci-lint run
	@echo Linters are ok!

gen:
	@echo "Removing generated files..."
	@rm -rf ./generated
	@mkdir -p generated
	@swagger generate model -f ./swagger.yml -t ./generated --accept-definitions-only
	@go generate ./...
