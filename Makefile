BINARY := agentctl
CMD     := ./cmd/agentctl

.PHONY: build test lint

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...
