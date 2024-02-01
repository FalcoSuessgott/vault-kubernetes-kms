projectname?=vault-kubernetes-kms

default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

PHONY: fmt
fmt: ## format go files
	gofumpt -w .
	gci write .
	
.PHONY: build
build: ## build plugin and image
	./scripts/build.sh

.PHONY: run
run: build ## apply kms k8s manifest
	./scripts/run.sh

..PHONY: vault
vault: ## creates vault dev server with transit engine + key
	./scripts/vault.sh

.PHONY: minikube
minikube: ## starts minikube
	./scripts/minikube.sh $(version)

PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml