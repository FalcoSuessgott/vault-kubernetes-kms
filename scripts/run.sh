#!/usr/bin/env bash

kubectl delete pod/vault-kubernetes-kms -n kube-system || true 

minikube ssh "sudo rm /opt/vaultkms.socket"

kubectl apply -f scripts/vault-kubernetes-kms.yml