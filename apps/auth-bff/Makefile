.PHONY: build test run lint

build:
	go build ./...

test:
	go test ./... -race -cover

run:
	go run ./cmd/server

lint:
	go vet ./...
