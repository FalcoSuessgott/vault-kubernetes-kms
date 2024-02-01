#!/usr/bin/bash
set -x 

version=v1.27.0
[ "$1" == "v2" ] && version=v1.29.0

minikube delete

minikube start --driver=docker --ports=8443:8443 --listen-address=0.0.0.0 --cni=bridge --kubernetes-version=$version

[ "$1" == "v1" ] && minikube cp ./scripts/encryption_provider_config_v1.yml minikube:/opt/encryption_provider_config.yml
[ "$1" == "v2" ] && minikube cp ./scripts/encryption_provider_config_v2.yml minikube:/opt/encryption_provider_config.yml