# vault-kms-plugin
A Kubernetes KMS Plugin that uses [HashiCorp Vaults](https://developer.hashicorp.com/vault) [Transit Engine](https://developer.hashicorp.com/vault/docs/secrets/transit) for securely encrypting Secrets, Config Maps and other Kubernetes Objects in etcd at Rest (on disk).

[![E2E](https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/e2e.yml/badge.svg)](https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/e2e.yml)
<img src="https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/test.yml/badge.svg" alt="drawing"/> <img src="https://github.com/FalcoSuessgott/vault-kubernetes-kms/actions/workflows/lint.yml/badge.svg" alt="drawing"/> <img src="https://img.shields.io/github/v/release/FalcoSuessgott/vault-kubernetes-kms" alt="drawing"/>
<a href="https://codecov.io/gh/FalcoSuessgott/vault-kubernetes-kms"><img src="https://codecov.io/gh/FalcoSuessgott/vault-kubernetes-kms/graph/badge.svg?token=naW3niAAt0"/></a>

## Why
HashiCorp Vault already offers useful [Kubernetes integrations](https://developer.hashicorp.com/vault/docs/platform/k8s), such as the Vault Secrets Operator for populating Kubernetes secrets from Vault Secrets or the Vault Agent Injector for injecting Vault secrets into a Pod using a sidecar container approach.

Wouldn't it be nice if you could also use your Vault server to encrypt Kubernetes secrets before they are written into etcd? The `vault-kubernetes-kms` plugin does exactly this!

Since the key used for encrypting secrets is not stored in Kubernetes, an attacker who intends to get unauthorized access to the plaintext values would need to compromise etcd and the Vault server.

## How does it work?
![img](docs/arch.svg)

`vault-kubernetes-kms` is supposed to run a s a static pod on every control plane node. It will create a unix socket and receive encryption requests through the socket from the `kube-apiserver`. The plugin will use a specified Vault transit encryption key to encrypt the data and send it back to the `kube-apiserver`, who will then send the encrypted response to `etcd`. To do so, you will have to configure the `kube-apiserver` to use a `EncryptionConfiguration` (See [https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/](https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/) for more details).

:warning: **`vault-kubernetes-kms` is in early stage! Running it in Production is not yet recommended. Im looking for early adopters in order to  gather important feedback.** :warning:

**[Check out the official documentation](https://falcosuessgott.github.io/vault-kubernetes-kms/)**

## Features
* support [Vault Token Auth](https://developer.hashicorp.com/vault/docs/auth/token) (not recommended for production) and [Vault Kubernetes Auth](https://developer.hashicorp.com/vault/docs/auth/kubernetes) using the Plugins Service Account
* support Kubernetes [KMS Plugin v1 (deprecated since `v1.28.0`) & v2 (stable in `v1.29.0`)](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin)
* automatic Token Renewal for avoiding Token expiry

## Without a KMS Provider
```bash
# create any secret
$> kubectl create secret generic secret-unencrypted -n default --from-literal=key=value

# proof that k8s secrets are stored unencrypted on disk and in etctd
$> kubectl -n kube-system exec etcd-minikube -- sh -c "ETCDCTL_API=3 etcdctl \
    --endpoints=https://127.0.0.1:2379 \
    --cert /var/lib/minikube/certs/etcd/server.crt \
    --key /var/lib/minikube/certs/etcd/server.key \
    --cacert /var/lib/minikube/certs/etcd/ca.crt \
    get /registry/secrets/default/secret-unencrypted" | hexdump -C
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 73 65 63 72 65 74  |s/default/secret|
00000020  2d 75 6e 65 6e 63 72 79  70 74 65 64 0a 6b 38 73  |-unencrypted.k8s|
00000030  00 0a 0c 0a 02 76 31 12  06 53 65 63 72 65 74 12  |.....v1..Secret.|
00000040  d1 01 0a b8 01 0a 12 73  65 63 72 65 74 2d 75 6e  |.......secret-un|
00000050  65 6e 63 72 79 70 74 65  64 12 00 1a 07 64 65 66  |encrypted....def|
00000060  61 75 6c 74 22 00 2a 24  33 62 31 66 34 34 31 32  |ault".*$3b1f4412|
00000070  2d 37 61 39 34 2d 34 38  62 35 2d 61 38 36 39 2d  |-7a94-48b5-a869-|
00000080  38 62 66 36 62 35 33 39  63 38 36 34 32 00 38 00  |8bf6b539c8642.8.|
00000090  42 08 08 fc 92 e9 ad 06  10 00 8a 01 60 0a 0e 6b  |B...........`..k|
000000a0  75 62 65 63 74 6c 2d 63  72 65 61 74 65 12 06 55  |ubectl-create..U|
000000b0  70 64 61 74 65 1a 02 76  31 22 08 08 fc 92 e9 ad  |pdate..v1"......|
000000c0  06 10 00 32 08 46 69 65  6c 64 73 56 31 3a 2c 0a  |...2.FieldsV1:,.|
000000d0  2a 7b 22 66 3a 64 61 74  61 22 3a 7b 22 2e 22 3a  |*{"f:data":{".":|
000000e0  7b 7d 2c 22 66 3a 6b 65  79 22 3a 7b 7d 7d 2c 22  |{},"f:key":{}},"|  # secret keys unencrypted
000000f0  66 3a 74 79 70 65 22 3a  7b 7d 7d 42 00 12 0c 0a  |f:type":{}}B....|
00000100  03 6b 65 79 12 05 76 61  6c 75 65 1a 06 4f 70 61  |.key..value..Opa|  # secret values unencrypted
00000110  7
```

## After configuring a KMS Provider

```bash
# create any k8s secret
$> kubectl create secret generic secret-encrypted -n default --from-literal=key=value

# proof that now secrets are stored encrypted on disk and in etctd
$> kubectl -n kube-system exec etcd-minikube -- sh -c "ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
    --cert /var/lib/minikube/certs/etcd/server.crt \
    --key /var/lib/minikube/certs/etcd/server.key \
    --cacert /var/lib/minikube/certs/etcd/ca.crt \
    get /registry/secrets/default/secret-encrypted" | hexdump -C
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 73 65 63 72 65 74  |s/default/secret|
00000020  2d 65 6e 63 72 79 70 74  65 64 0a 6b 38 73 3a 65  |-encrypted.k8s:e|
00000030  6e 63 3a 6b 6d 73 3a 76  32 3a 76 61 75 6c 74 3a  |nc:kms:v2:vault:|
00000040  0a 84 02 54 b2 35 ef 01  ca 9a 3b 00 00 00 00 9e  |...T.5....;.....|
00000050  c6 26 2a e6 23 f0 c0 b7  22 3a 99 d8 36 81 07 af  |.&*.#...":..6...|
00000060  ae d7 33 62 3f 99 62 ed  b1 f0 10 f9 47 05 46 0e  |..3b?.b.....G.F.|
00000070  c4 4f b5 63 aa 7d e5 3b  44 fd 9e 6d e7 42 2f 8f  |.O.c.}.;D..m.B/.|
00000080  44 27 57 f6 ee 62 69 9b  49 6b 00 bb d3 38 d4 85  |D'W..bi.Ik...8..|
00000090  ce 57 b6 fa 95 4b 88 ea  9c de 1f c9 e0 05 48 a5  |.W...K........H.|
000000a0  5f 08 01 c4 c9 f2 3d 5d  35 e6 0e e7 0a 89 18 ab  |_.....=]5.......|
000000b0  72 f2 ba 2b 3e cb 20 bf  cd 9a 0f 97 78 d4 79 05  |r..+>. .....x.y.|
000000c0  77 52 1b ba bd 55 2b 9f  e0 f1 af dc 04 3a b0 a9  |wR...U+......:..|
000000d0  70 8e 7a 97 10 8f b4 41  75 4b b8 24 dc 6f 10 04  |p.z....AuK.$.o..|
000000e0  4b b9 a0 fc a5 cc 02 e9  53 6e 1a be 31 c2 2b 38  |K.......Sn..1.+8|
000000f0  d3 d9 07 6f ee 40 9d 20  dc d6 68 29 e0 20 3f 8f  |...o.@. ..h). ?.|
00000100  0a 1c a0 03 4f bf 9f 4b  8a 76 9b 8c 06 5b 4f c8  |....O..K.v...[O.|
00000110  75 b7 a1 a3 d1 4e b2 00  81 53 ed 6a b2 d9 03 88  |u....N...S.j....|
00000120  cb 3c 3d bb 12 b4 88 d3  e0 c7 a7 e1 31 0c 18 55  |.<=.........1..U|
00000130  26 fb 38 86 5b fb 5c bc  2b e0 8b f3 56 84 78 b2  |&.8.[.\.+...V.x.|
00000140  ae fc 11 98 1e a7 b9 12  0a 31 37 31 30 37 34 34  |.........1710744|
00000150  32 30 33 1a 59 76 61 75  6c 74 3a 76 31 3a 30 4f  |203.Yvault:v1:0O| # encrypted secret stored in etcd on disk
00000160  31 7a 58 5a 31 54 34 33  37 70 53 59 7a 69 58 41  |1zXZ1T437pSYziXA|
00000170  32 30 62 4a 44 58 48 62  4a 31 55 65 50 75 6b 61  |20bJDXHbJ1UePuka|
00000180  70 70 47 4f 4f 54 51 65  6b 61 61 7a 6e 32 76 73  |ppGOOTQekaazn2vs|
00000190  35 4f 41 54 56 66 65 2b  31 63 75 7a 76 6a 64 6a  |5OATVfe+1cuzvjdj|
000001a0  41 43 2b 6f 31 45 61 6a  57 72 32 53 6c 57 0a     |AC+o1EajWr2SlW.|
000001af
```
