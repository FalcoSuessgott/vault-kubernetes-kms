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
test: ## display test coverage
	go test --cover -parallel=1 -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | sort -rnk3

PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golang-ci.yml

..PHONY: setup-vault
setup-vault: ## setup a local vault dev server with transit engine + key
	./scripts/vault.sh

.PHONY: setup-registry
setup-registry: ## setup a local docker registry for pulling in kind
	./scripts/local-registry.sh

.PHONY: setup-prometheus
setup-prometheus: ## setup prometheus locally
	docker run \
		--rm \
		-v "${PWD}/assets/prometheus.yml:/etc/prometheus/prometheus.yml" \
		--name prometheus \
		-p 9090:9090 \
		prom/prometheus:latest

.PHONY: gen-load
gen-load: ## generate load on KMS plugin
	while true; do \
		go run cmd/v2_client/main.go $(shell openssl rand -base64 12);\
	done;

.PHONY: gen-secrets
gen-secrets: ## generate secrets on KMS plugin
	while true; do \
		kubectl create secret generic $(shell openssl rand -base64 12) -n default --from-literal=$(shell openssl rand -base64 12)=$(shell openssl rand -base64 12);\
	done;

.PHONY: setup-grafana
setup-grafana: ## setup grafana locally
	docker run \
		--rm \
		-v "${PWD}/assets/grafana_datasource.yml:/etc/grafana/provisioning/datasources/grafana_datasource.yml" \
		-v "${PWD}/assets/grafana_dashboard.yml:/etc/grafana/provisioning/dashboards/grafana_dashboard.yml" \
		-v "${PWD}/assets/dashboard.json:/var/lib/grafana/dashboards/dashboard.json" \
		--name grafana \
		-p 3000:3000 \
		grafana/grafana:latest

.PHONY: setup-kind
setup-kind: ## setup kind cluster with encrpytion provider configured
	kind delete cluster --name=kms || true
	kind create cluster --name=kms --config scripts/kind-config_v2.yaml

.PHONY: setup-local
setup-local: setup-vault setup-registry setup-kind ## complete local setup
