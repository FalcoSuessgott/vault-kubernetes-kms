default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

PHONY: fmt
fmt: ## format go files
	gofumpt -w .
	gci write .
	pre-commit run -a

.PHONY: docs
docs: ## render docs locally
	mkdocs serve

PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml

..PHONY: setup-vault
setup-vault: ## setup a local vault dev server with transit engine + key
	./scripts/vault.sh

.PHONY: setup-registry
setup-registry: ## setup a local docker registry for pulling in kind
	./scripts/local-registry.sh

.PHONY: setup-kind
setup-kind: ## setup kind cluster with encrpytion provider configured
	kind delete cluster --name=kms || true
	kind create cluster --name=kms --config scripts/kind-config.yaml

.PHONY: setup-local
setup-local: setup-vault setup-registry setup-kind ## complete local setup
