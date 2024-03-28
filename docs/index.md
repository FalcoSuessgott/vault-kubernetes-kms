# vault-kms-plugin
A Kubernetes KMS Plugin that uses [HashiCorp Vaults](https://developer.hashicorp.com/vault) [Transit Engine](https://developer.hashicorp.com/vault/docs/secrets/transit) for securely encrypting Secrets, Config Maps and other Kubernetes Objects in etcd.

<img src="https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/test.yml/badge.svg" alt="drawing"/> <img src="https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/lint.yml/badge.svg" alt="drawing"/> <img src="https://img.shields.io/github/v/release/FalcoSuessgott/vault-kubernetes-kms" alt="drawing"/>

## Overview
![img](arch.svg)

:warning: **`vault-kubernetes-kms` is in early stage! Running it in Production is not yet recommended. Im looking for early adopters in order to  gather important feedback.** :warning:

## Features
* support [Vault Token Auth](https://developer.hashicorp.com/vault/docs/auth/token) (not recommended for production) and [Vault Kubernetes Auth](https://developer.hashicorp.com/vault/docs/auth/kubernetes) using the Plugins Service Account
* support Kubernetes [KMS Plugin v1 (deprecated since `v1.28.0`) & v2 (stable in `v1.29.0`)](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin)
* automatic Token Renewal for avoiding Token expiry
