# vault-kms-plugin
A Kubernetes KMS Plugin that uses [HashiCorp Vaults](https://developer.hashicorp.com/vault) [Transit Engine](https://developer.hashicorp.com/vault/docs/secrets/transit) for securely encrypting Secrets, Config Maps and other Kubernetes Objects in etcd at rest (on disk).

[![E2E](https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/e2e.yml/badge.svg)](https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/e2e.yml)
<img src="https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/test.yml/badge.svg" alt="drawing"/> <img src="https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/lint.yml/badge.svg" alt="drawing"/> <img src="https://img.shields.io/github/v/release/FalcoSuessgott/vault-kubernetes-kms" alt="drawing"/>
<a href="https://codecov.io/gh/FalcoSuessgott/vault-kubernetes-kms"><img src="https://codecov.io/gh/FalcoSuessgott/vault-kubernetes-kms/graph/badge.svg?token=naW3niAAt0"/></a>

## Why
HashiCorp Vault already offers useful [Kubernetes integrations](https://developer.hashicorp.com/vault/docs/platform/k8s), such as the Vault Secrets Operator for populating Kubernetes secrets from Vault Secrets or the Vault Agent Injector for injecting Vault secrets into a Pod using a sidecar container approach.

Wouldn't it be nice if you could also use your Vault server to encrypt Kubernetes secrets before they are written into etcd? The `vault-kubernetes-kms` plugin does exactly this!

Since the key used for encrypting secrets is not stored in Kubernetes, an attacker who intends to get unauthorized access to the plaintext values would need to compromise etcd and the Vault server.
## How does it work?
![img](arch.svg)

`vault-kubernetes-kms` is supposed to run as a static pod on every control plane node. It will create a unix socket and receive encryption requests through the socket from the `kube-apiserver`. The plugin will use a specified Vault transit encryption key to encrypt the data and send it back to the `kube-apiserver`, who will then send the encrypted response to `etcd`. To do so, you will have to configure the `kube-apiserver` to use a `EncryptionConfiguration` (See [https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/](https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/) for more details).

:warning: **`vault-kubernetes-kms` is in early stage! Running it in Production is not yet recommended. Im looking for early adopters in order to  gather important feedback.** :warning:

## Features
* support [Vault Token Auth](https://developer.hashicorp.com/vault/docs/auth/token) (not recommended for production), [AppRole](https://developer.hashicorp.com/vault/docs/auth/approle) and [Vault Kubernetes Auth](https://developer.hashicorp.com/vault/docs/auth/kubernetes) using the Plugins Service Account
* support Kubernetes [KMS Plugin v1 (deprecated since `v1.28.0`) & v2 (stable in `v1.29.0`)](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin)
* automatic Token Renewal for avoiding Token expiry
