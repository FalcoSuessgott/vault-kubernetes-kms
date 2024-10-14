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

PHONY: test
test: ## test
	gotestsum -- -v --shuffle=on -race -coverprofile="coverage.out" -covermode=atomic ./...

PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml

..PHONY: setup-vault
setup-vault: ## setup a local vault dev server with transit engine + key
	./scripts/vault.sh

.PHONY: setup-registry
setup-registry: ## setup a local docker registry for pulling in kind
	./scripts/local-registry.sh

.PHONY: gen-load
gen-load: ## generate load on KMS plugin
	while true; do \
		go run cmd/v2_client/main.go $$(openssl rand -base64 12);\
	done;

.PHONY: gen-secrets
gen-secrets: ## generate secrets on KMS plugin
	while true; do \
		kubectl create secret generic $$(openssl rand -hex 8 | tr '[:upper:]' '[:lower:]')\
			--from-literal=$$(openssl rand -hex 8 | tr '[:upper:]' '[:lower:]')=$$(openssl rand -hex 8 | tr '[:upper:]' '[:lower:]');\
	done;

.PHONY: setup-kind
setup-kind: ## setup kind cluster with encrpytion provider configured
	kind delete cluster --name=kms || true
	kind create cluster --name=kms --config scripts/kind-config_v2.yaml

.PHONY: setup-o11y
setup-o11y: ## install grafana and prometheus via helm
	kubectl apply -f scripts/svc.yml

	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update

	helm install prometheus prometheus-community/prometheus --values scripts/prometheus_values.yml
	helm install grafana grafana/grafana --values scripts/grafana_values.yml

	kubectl get secret --namespace default grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo

.PHONY: setup-local
setup-local: setup-vault setup-registry setup-kind ## complete local setup

.PHONY: destroy
destroy: ## destroy kind cluster
	kind delete cluster --name=kms
