#!/usr/bin/env bash

eval $(minikube docker-env)

CGO_ENABLED=0 go build -o vault-kubernetes-kms cmd/main.go

docker build -t vault-kubernetes-kms .