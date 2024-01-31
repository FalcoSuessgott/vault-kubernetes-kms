#!/usr/bin/bash

minikube delete

minikube start --driver=docker --ports=8443:8443 --listen-address=0.0.0.0 --cni=bridge --kubernetes-version=v1.29.0

minikube cp ./scripts/encryption_provider_config.yml minikube:/opt/encryption_provider_config.yml