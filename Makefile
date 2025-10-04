GOCMD=go
GOCACHE_DIR?=$(CURDIR)/.cache/go-build
GOTEST=CGO_ENABLED=0 GOCACHE=$(GOCACHE_DIR) $(GOCMD) test ./...
GOTIDY=GOCACHE=$(GOCACHE_DIR) $(GOCMD) mod tidy
GORUN=GOCACHE=$(GOCACHE_DIR) $(GOCMD) run ./cmd/server
GOLANGCI_LINT?=golangci-lint
LINT_ENV=CGO_ENABLED=0 XDG_CACHE_HOME=$(CURDIR)/.cache GOLANGCI_LINT_CACHE=$(CURDIR)/.cache/golangci

.PHONY: deps fmt lint test run

deps:
	$(GOTIDY)

fmt:
	$(GOCMD) fmt ./...

lint:
	$(LINT_ENV) $(GOLANGCI_LINT) run ./...

test:
	$(GOTEST)

run:
	$(GORUN)
