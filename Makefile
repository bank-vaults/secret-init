# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

export PATH := $(abspath bin/):${PATH}

##@ General

# Targets commented with ## will be visible in "make help" info.
# Comments marked with ##@ will be used as categories for a group of targets.

.PHONY: help
.DEFAULT_GOAL := help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: up
up: ## Start development environment
	docker compose up -d

.PHONY: down
down: ## Destroy development environment
	docker compose down -v

##@ Build

.PHONY: build
build: ## Build binary
	@mkdir -p build
	go build -race -o build/secret-init .

.PHONY: artifacts
artifacts: container-image binary-snapshot
artifacts: ## Build artifacts

.PHONY: container-image
container-image: ## Build container image
	docker build .

.PHONY: binary-snapshot
binary-snapshot: ## Build binary snapshot
	VERSION=v${GORELEASER_VERSION} $(GORELEASER_BIN) release --clean --skip=publish --snapshot

##@ Checks

.PHONY: check
check: lint test ## Run checks (tests and linters)

.PHONY: lint
lint: lint-go lint-docker lint-yaml
lint: ## Run linters

.PHONY: lint-go
lint-go:
	$(GOLANGCI_LINT_BIN) run $(if ${CI},--out-format colored-line-number,)

.PHONY: lint-docker
lint-docker:
	$(HADOLINT_BIN) Dockerfile

.PHONY: lint-yaml
lint-yaml:
	$(YAMLLINT_BIN) $(if ${CI},-f github,) --no-warnings .

.PHONY: test
test: ## Run tests
	go test -race -v ./...

.PHONY: test-e2e
test-e2e: ## Run e2e tests
	@export BATS_LIB_PATH=${PWD}/bin/bats-core/libexec/bats-core/lib && \
	$(BATS_BIN) e2e

.PHONY: fmt
fmt: ## Format code
	$(GOLANGCI_LINT_BIN) run --fix

.PHONY: license-check
license-check: ## Run license check
	$(LICENSEI_BIN) check
	$(LICENSEI_BIN) header

##@ Dependencies

# Dependency versions
GOLANGCI_LINT_VERSION = 1.61.0
LICENSEI_VERSION = 0.9.0
COSIGN_VERSION = 2.4.0
GORELEASER_VERSION = 2.2.0
BATS_VERSION = 1.11.0

# Dependency binaries
GOLANGCI_LINT_BIN := golangci-lint
LICENSEI_BIN := licensei
COSIGN_BIN := cosign
GORELEASER_BIN := goreleaser
BATS_BIN := bats

# TODO: add support for hadolint and yamllint dependencies
HADOLINT_BIN := hadolint
YAMLLINT_BIN := yamllint

# If we have "bin" dir, use those binaries instead
ifneq ($(wildcard ./bin/.),)
	GOLANGCI_LINT_BIN := bin/$(GOLANGCI_LINT_BIN)
	LICENSEI_BIN := bin/$(LICENSEI_BIN)
	COSIGN_BIN := bin/$(COSIGN_BIN)
	GORELEASER_BIN := bin/$(GORELEASER_BIN)
	BATS_BIN := bin/$(BATS_BIN)
endif

deps: bin/golangci-lint bin/licensei bin/cosign bin/goreleaser bin/bats
deps: ## Install dependencies

bin/golangci-lint:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- v${GOLANGCI_LINT_VERSION}

bin/licensei:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/goph/licensei/master/install.sh | bash -s -- v${LICENSEI_VERSION}

bin/cosign:
	@mkdir -p bin
	@OS=$$(uname -s); \
	case $$OS in \
		"Linux") \
			curl -sSfL https://github.com/sigstore/cosign/releases/download/v${COSIGN_VERSION}/cosign-linux-amd64 -o bin/cosign; \
			;; \
		"Darwin") \
			curl -sSfL https://github.com/sigstore/cosign/releases/download/v${COSIGN_VERSION}/cosign-darwin-arm64 -o bin/cosign; \
			;; \
		*) \
			echo "Unsupported OS: $$OS"; \
			exit 1; \
			;; \
	esac
	@chmod +x bin/cosign

bin/goreleaser:
	@mkdir -p bin
	curl -sfL https://goreleaser.com/static/run -o bin/goreleaser
	@chmod +x bin/goreleaser

bin/bats:
	@mkdir -p bin/bats-core
	@mkdir -p tmpbats
	git clone --branch v${BATS_VERSION} --depth 1 https://github.com/bats-core/bats-core.git tmpbats
	bash tmpbats/install.sh bin/bats-core
	@ln -sF ${PWD}/bin/bats-core/bin/bats ${PWD}/bin
	@rm -rf tmpbats
	git clone https://github.com/bats-core/bats-support.git bin/bats-core/libexec/bats-core/lib/bats-support
	git clone https://github.com/bats-core/bats-assert.git bin/bats-core/libexec/bats-core/lib/bats-assert
