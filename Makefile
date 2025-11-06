GOCMD=go
GOCACHE_DIR?=$(CURDIR)/.cache/go-build
GOMODCACHE_DIR?=$(CURDIR)/.cache/go-mod
BIN_DIR?=$(CURDIR)/bin
BINARY_NAME?=atlassian-mcp
GOENV=GOCACHE=$(GOCACHE_DIR) GOMODCACHE=$(GOMODCACHE_DIR)
GOTEST=CGO_ENABLED=0 $(GOENV) $(GOCMD) test ./...
GOTIDY=$(GOENV) $(GOCMD) mod tidy
GOBUILD=CGO_ENABLED=0 $(GOENV) $(GOCMD) build
GORUN=$(GOENV) $(GOCMD) run ./cmd/server
GOLANGCI_LINT?=golangci-lint
LINT_ENV=CGO_ENABLED=0 XDG_CACHE_HOME=$(CURDIR)/.cache GOLANGCI_LINT_CACHE=$(CURDIR)/.cache/golangci

.PHONY: deps fmt lint test build run

deps:
	$(GOTIDY)

fmt:
	$(GOENV) $(GOCMD) fmt ./...

lint:
	$(LINT_ENV) $(GOLANGCI_LINT) run ./...

test:
	$(GOTEST)

build: | $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/server

run:
	$(GORUN)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)
