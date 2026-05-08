BINARY := git-wt
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/git-wt

test:
	go test -count=1 -race ./...

# Skip the e2e package which builds the binary and shells out — slower and
# unnecessary on PR-iteration loops. CI runs `make test` on main; PRs run
# `make test-short` + `make lint`.
test-short:
	go test -count=1 ./internal/...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

dev:
	go run ./cmd/git-wt $(ARGS)

clean:
	rm -f $(BINARY)
	rm -rf dist/

.PHONY: build test test-short lint fmt vet dev clean
