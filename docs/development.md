# Development
This guide walks you through the required steps of building and running this plugin locally with and without Kubernetes.

## Local Development without Kubernetes
You don't have to deploy the plugin in a Kubernetes Cluster, you can just execute it locally, given you have a Vault running:

```bash
$> make setup-vault
$> go run main.go -vault-address=http://127.0.0.1:8200 -auth-method=token -token=root -socket=unix:///tmp/kms.socket
{"level":"info","timestamp":"2024-08-25T15:36:19.233+1000","caller":"cmd/plugin.go:154","message":"starting kms plugin","auth-method":"token","socket":"unix:///tmp/kms.socket","debug":false,"vault-address":"http://127.0.0.1:8200","vault-namespace":"","transit-engine":"transit","transit-key":"kms","health-port":":8080","disable-v1":false}
{"level":"info","timestamp":"2024-08-25T15:36:19.235+1000","caller":"cmd/plugin.go:167","message":"Successfully authenticated to vault"}
{"level":"info","timestamp":"2024-08-25T15:36:19.235+1000","caller":"cmd/plugin.go:174","message":"Successfully dialed to unix domain socket","socket":"unix:///tmp/kms.socket"}
{"level":"info","timestamp":"2024-08-25T15:36:19.235+1000","caller":"cmd/plugin.go:184","message":"Successfully registered kms plugin v1"}
{"level":"info","timestamp":"2024-08-25T15:36:19.235+1000","caller":"cmd/plugin.go:191","message":"Successfully registered kms plugin v2"}
```

In order to send encryption and decryption requests you can use the client CLI tool in `cmd/v2_Client/main.go`. This tool simply connects to the plugin and encrypts a given string and decrypts it back to its plaintext version:

```bash
$> go run cmd/v2_client/main.go encrypt this string
"encrypt this string" -> "dmF1bHQ6djE6VzJMcHp4UmJMdHV4TWNnUnVWMWJQQzBHMWZ0VkwvZFVUMldLRzQ0RUtCa1VJcjVwVjgxMFd3T29pRmVhQzVNPQ==" -> "encrypt this string"
```

## Local Development with Kubernetes

The following steps describe how to build & run the vault-kubernetes-kms completely locally using `docker`, `vault` & `kind`.

### Requirements
Obviously you will need all the tools mentioned above installed. Also this setup is only tested on Linux and MacOS.

### Components
Basically, we will need:

1. A local Vault server initialized & unsealed and with a transit engine enabled as well as a transit key created.
2. A local (docker) registry so kind can pull the currently unreleased `vault-kubernetes-kms` image.
3. A local Kubernetes Cluster (kind) configured to use the local registry as well as the required settings for the kube-apiservers encryption provider config.

#### 1. Local Vault Server using `vault`
The following snippets sets up a local vault development server and creates a transit engine as well as a transit key.

This script is located in `scripts/vault.sh` and is available via `make setup-vault`:

```bash
{!../scripts/vault.sh!}
```


#### 2. Local Container/Docker Registry using `docker`
The following snippet, starts a local container registry, builds the current commits `vault-kubernetes-kms` image and tags & pushes the image to the local registry.

This script is located in `scripts/local-registry.sh` and is available via `make setup-registry`:

```bash
{!../scripts/local-registry.sh!}
```

#### 3. Local Kubernetes Cluster using `kind`
Last but not least, we combine the above mentioned tools and consume them with `kind`

The following `kind`-config configures the local running registry, copies the encryption provider config and the `vault-kubernetes-kms` static pod manifest to the Kubernetes host and patches the `kube-apiserver` for using the provided encryption provider config.

This can be run via `make setup-kind`, which runs `kind create cluster --name=kms --config scripts/kind-config.yaml` under the hood:

```yaml
{!../scripts/kind-config_v2.yaml!}
```

**the `vault-kubernetes-kms` manifest:**

for development purposes, we use the vault dev servers configured root token (`"root"`):

```yaml
{!../scripts/vault-kubernetes-kms.yml!}
```

## Putting it together
So if you wanna run all components locally and build the current commits plugin, it would look like this:

```bash
$> make setup-vault
$> make setup-registry
$> make setup-kind
kind delete cluster --name=kms || true
Deleting cluster "kms" ...
Deleted nodes: ["kms-control-plane"]
kind create cluster --name=kms --config scripts/kind-config.yaml
Creating cluster "kms" ...
 ✓ Ensuring node image (kindest/node:v1.29.2) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-kms"
You can now use your cluster with:

# testing kubectl
$> kubectl get pod -A
NAMESPACE            NAME                                        READY   STATUS    RESTARTS   AGE
kube-system          coredns-76f75df574-7pzq4                    1/1     Running   0          17m
kube-system          coredns-76f75df574-pkqrj                    1/1     Running   0          17m
kube-system          etcd-kms-control-plane                      1/1     Running   0          17m
kube-system          kindnet-w2hgj                               1/1     Running   0          17m
kube-system          kube-apiserver-kms-control-plane            1/1     Running   0          17m
kube-system          kube-controller-manager-kms-control-plane   1/1     Running   0          17m
kube-system          kube-proxy-w66mx                            1/1     Running   0          17m
kube-system          kube-scheduler-kms-control-plane            1/1     Running   0          17m
kube-system          vault-kubernetes-kms-kms-control-plane      1/1     Running   0          17m
local-path-storage   local-path-provisioner-7577fdbbfb-rmqq8     1/1     Running   0          17m

# creating a kubernetes secret
$> kubectl create secret generic secret -n default --from-literal=key=value
secret/secret created

# checking encryption value in etcd
$> kubectl -n kube-system exec etcd-kms-control-plane -- sh -c "ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
    --cert /etc/kubernetes/pki/etcd/server.crt \
    --key /etc/kubernetes/pki/etcd/server.key \
    --cacert /etc/kubernetes/pki/etcd/ca.crt \
    get /registry/secrets/default/secret" | hexdump -C
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 73 65 63 72 65 74  |s/default/secret|
00000020  2d 65 6e 63 72 79 70 74  65 64 0a 6b 38 73 3a 65  |-encrypted.k8s:e|
00000030  6e 63 3a 6b 6d 73 3a 76  32 3a 76 61 75 6c 74 2d  |nc:kms:v2:vault-|
00000040  6b 75 62 65 72 6e 65 74  65 73 2d 6b 6d 73 3a 0a  |kubernetes-kms:.|
00000050  a4 02 7f fe e1 bb 63 29  71 62 b6 1f c0 be d5 a0  |......c)qb......|
00000060  a8 38 0b e6 a1 bc 4b bb  16 ff 3f d3 3f 14 e4 be  |.8....K...?.?...|
00000070  7e fa 53 de d5 06 75 08  3a fd 5f fb e9 a3 b1 29  |~.S...u.:._....)|
00000080  e2 9f 26 1c ef bb 1b 24  37 bc f3 ab 9c df 46 c4  |..&....$7.....F.|
00000090  8f 47 33 e5 c0 76 54 3b  e7 f4 3b da 0d bf 80 e0  |.G3..vT;..;.....|
000000a0  52 88 cd 1a 6f c6 ec 7f  bb 51 4b ef 0c c7 b6 8f  |R...o....QK.....|
000000b0  31 2d 6b 96 3d 37 ee cb  f0 56 83 40 d8 b4 21 75  |1-k.=7...V.@..!u|
000000c0  31 78 e7 ab ec 5f 6e f7  bf 84 86 34 2a aa 65 1b  |1x..._n....4*.e.|
000000d0  8a 2b ce 6c ae 6f b6 df  11 5b ec 14 9d b9 00 74  |.+.l.o...[.....t|
000000e0  9d 0c 01 11 c4 67 48 67  3d d3 8f 58 1a 0d da 34  |.....gHg=..X...4|
000000f0  0d 55 19 91 cc 7e db c3  36 a2 6d 2f ea 28 10 ab  |.U...~..6.m/.(..|
00000100  9b 1e 71 a9 d4 b1 74 6b  2f cc ef aa 30 d9 1a b8  |..q...tk/...0...|
00000110  68 30 3b 5b c5 3a 32 69  6a 75 4d 43 68 1f 33 23  |h0;[.:2ijuMCh.3#|
00000120  af 56 8c 15 c9 17 cb 8a  46 fc 9f 5a 24 da 25 16  |.V......F..Z$.%.|
00000130  15 31 ce 41 59 6b b8 c6  7d 5e b3 ee 07 a7 65 3b  |.1.AYk..}^....e;|
00000140  a8 f2 8a ab e7 d0 37 bc  9c e6 e6 33 71 57 c5 6c  |......7....3qW.l|
00000150  09 ff e9 65 c9 8c 9f aa  1c e2 df a4 ad fc a0 02  |...e............|
00000160  2b 6d 93 5e 44 20 64 28  d7 3f e1 98 eb 84 ab 22  |+m.^D d(.?....."|
00000170  82 92 7a b6 b2 b8 12 0a  31 37 31 31 32 34 32 33  |..z.....17112423|
00000180  36 34 1a 59 76 61 75 6c  74 3a 76 31 3a 6d 42 41  |64.Yvault:v1:mBA|
00000190  4a 47 56 56 35 72 46 78  36 47 4c 4f 62 33 46 50  |JGVV5rFx6GLOb3FP|
000001a0  37 4a 38 73 5a 79 4a 38  2f 68 36 61 48 2b 46 57  |7J8sZyJ8/h6aH+FW|
000001b0  55 46 2f 67 53 68 30 65  41 31 4e 51 45 47 6e 30  |UF/gSh0eA1NQEGn0|
000001c0  5a 30 38 66 6a 59 45 53  30 4c 31 79 35 45 49 50  |Z08fjYES0L1y5EIP|
000001d0  33 67 4c 72 77 61 35 4b  61 44 35 43 63 28 01 0a  |3gLrwa5KaD5Cc(..|
000001e0

# receiving the secret
$>  kubectl get secret secret -o json | jq '.data | map_values(@base64d)'
{
  "key": "value"
}
```
