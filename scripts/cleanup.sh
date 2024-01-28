#!/usr/bin/env bash

minikube delete

kill $(pgrep -x vault) || true 

rm vault-kubernetes-kms nohup.out

