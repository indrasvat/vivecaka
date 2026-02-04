BINARY    := vivecaka
BINDIR    := bin
MODULE    := github.com/indrasvat/vivecaka
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE      := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS   := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.DEFAULT_GOAL := help

## ── Build & Run ─────────────────────────────────────────────────────

.PHONY: build
build: ## Build binary to bin/vivecaka
	@mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(BINARY) ./cmd/vivecaka

.PHONY: install
install: ## Install to $GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/vivecaka

.PHONY: run
run: ## Run with go run
	go run -ldflags "$(LDFLAGS)" ./cmd/vivecaka

.PHONY: dev
dev: ## Run with auto-reload (requires air: go install github.com/air-verse/air@latest)
	@command -v air >/dev/null 2>&1 && air || (echo "Install air: go install github.com/air-verse/air@latest" && go run ./cmd/vivecaka)

## ── Quality ─────────────────────────────────────────────────────────

.PHONY: fmt
fmt: ## Format code with gofmt
	@gofmt -l -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || (echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

.PHONY: test
test: ## Run tests with race detector
	go test -race -cover -count=1 ./...

.PHONY: coverage
coverage: ## Generate and open coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: ci
ci: fmt vet lint test build ## Run all quality checks (fmt, vet, lint, test, build)

## ── Dependencies ────────────────────────────────────────────────────

.PHONY: deps
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

.PHONY: tools
tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/evilmartians/lefthook@latest
	go install github.com/goreleaser/goreleaser/v2@latest

## ── Git Hooks ───────────────────────────────────────────────────────

.PHONY: hooks-install
hooks-install: ## Install lefthook git hooks
	@command -v lefthook >/dev/null 2>&1 || (echo "Install: go install github.com/evilmartians/lefthook@latest" && exit 1)
	lefthook install

.PHONY: hooks-uninstall
hooks-uninstall: ## Remove lefthook git hooks
	lefthook uninstall

## ── Release ─────────────────────────────────────────────────────────

.PHONY: snapshot
snapshot: ## Build snapshot release with goreleaser (local only)
	@command -v goreleaser >/dev/null 2>&1 || (echo "Install: go install github.com/goreleaser/goreleaser/v2@latest" && exit 1)
	goreleaser release --snapshot --clean

.PHONY: release
release: ## Run goreleaser (CI only, requires GITHUB_TOKEN)
	goreleaser release --clean

## ── Maintenance ─────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BINDIR) coverage.out coverage.html dist/

.PHONY: help
help: ## Show this help message
	@printf "\n\033[1m%s\033[0m\n\n" "vivecaka — GitHub PR TUI"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
