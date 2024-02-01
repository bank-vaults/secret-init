# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

export PATH := $(abspath bin/):${PATH}

# Dependency versions
GOLANGCI_VERSION = 1.53.3
LICENSEI_VERSION = 0.8.0
COSIGN_VERSION = 2.2.2
GORELEASER_VERSION = 1.18.2
BATS_VERSION = 1.2.1

# GoReleaser distribution
GORELEASER_DISTRIBUTION := oss

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

.PHONY: stop
stop: ## Stop development environment
	docker compose stop

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
	goreleaser release --rm-dist --skip-publish --snapshot

##@ Checks

.PHONY: check
check: test lint ## Run checks (tests and linters)

.PHONY: test
test: ## Run tests
	go test -race -v ./...

.PHONY: test-e2e
test-e2e: ## Run e2e tests
	@export BATS_LIB_PATH=${PWD}/bin/bats-core/libexec/bats-core/lib && \
	bats e2e

.PHONY: lint
lint: lint-go lint-docker lint-yaml
lint: ## Run linters

.PHONY: lint-go
lint-go:
	golangci-lint run $(if ${CI},--out-format github-actions,)

.PHONY: lint-docker
lint-docker:
	hadolint Dockerfile

.PHONY: lint-yaml
lint-yaml:
	yamllint $(if ${CI},-f github,) --no-warnings .

.PHONY: fmt
fmt: ## Format code
	golangci-lint run --fix

.PHONY: license-check
license-check: ## Run license check
	licensei check
	licensei header

##@ Dependencies

deps: bin/golangci-lint bin/licensei bin/cosign bin/goreleaser bin/bats
deps: ## Install dependencies

bin/golangci-lint:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- v${GOLANGCI_VERSION}

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
	@mkdir -p tmpgoreleaser
	curl -sfL https://goreleaser.com/static/run > tmpgoreleaser/goreleaser
	@sed -i '' -e 's|"\$$TMP_DIR/goreleaser" "\$$@"|mv "\$$TMP_DIR/goreleaser" "bin/"\nrm -rf "\$$TMP_DIR"|' tmpgoreleaser/goreleaser
	bash tmpgoreleaser/goreleaser DISTRIBUTION=${GORELEASER_DISTRIBUTION} VERSION=v${GORELEASER_VERSION}
	@rm -rf tmpgoreleaser

bin/bats:
	@mkdir -p bin/bats-core
	@mkdir -p tmpbats
	git clone https://github.com/bats-core/bats-core.git tmpbats
	bash tmpbats/install.sh bin/bats-core
	@ln -sF ${PWD}/bin/bats-core/bin/bats ${PWD}/bin
	@rm -rf tmpbats
	git clone https://github.com/bats-core/bats-support.git bin/bats-core/libexec/bats-core/lib/bats-support
	git clone https://github.com/bats-core/bats-assert.git bin/bats-core/libexec/bats-core/lib/bats-assert
