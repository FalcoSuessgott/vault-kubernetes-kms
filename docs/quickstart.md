# Quickstart
This Guide will walk you through the required steps of installing and configuring the `vault-kms-plugin` for Kubernetes. It currently uses token based authentication and HTTP communication, which is not secure enough when running in production.

!!! tip
    Checkout [https://falcosuessgott.github.io/hashicorp-vault-playground/home/](https://falcosuessgott.github.io/hashicorp-vault-playground/home/) a project that helps you quickly setting up HashiCorp Vault locally with many useful Kubernetes Labs already pre configured.

!!! warning
    This guide uses the new version of the Kubernetes KMS Plugin API, which was introduced in Kubernetes v1.29.0 ([https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#kms-v2](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#kms-v2)).

## Requirements
In order to run this guide, you will need to have `kind`, `kubectl` and `vault` installed on your system. This guide has been tested on MacOS and Linux.

!!! note
    `vault-kubernetes-kms` is only published as `amd` (x86_64) images.

    You will make sure, you actually pull `amd` images. You can test it, by using `docker run -it ubuntu /usr/bin/uname -p` which, should output `86_64`.

    If you need `arm` images, raise an issue.

## Clone the repository
```bash
$> git clone --depth 1 https://github.com/FalcoSuessgott/vault-kubernetes-kms.git
$> cd vault-kubernetes-kms
```

## Setup a Vault in development mode
```bash
# invokes ./scripts/setup.vault.sh
$> make setup-vault

# point your vault CLI to the local Vault server
$> export VAULT_ADDR="http://127.0.0.1:8200"
$> export VAULT_SKIP_VERIFY="true"
$> export VAULT_TOKEN="root"
$> vault status
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.15.6
Build Date      2024-02-28T17:07:34Z
Storage Type    inmem
Cluster Name    vault-cluster-32a0c10b
Cluster ID      2081a49b-8372-3857-3754-b576e0877682
HA Enabled      false

# a transit engine `transit` & key `kms` has been created
$> vault list transit/keys
Keys
----
kms
```

## setup vault with encryption provider config usage
Now, we have a local running Vault server, lets start a local running Kubernetes cluster using `kind`, which will deploy the `vault-kubernetes-kms` plugin as a static pod on the control plane as well as its required `encryption_provider_config` (see [https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/](https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/) for the required configuration steps):

```bash
# can take up to 2 minutes
$> kind create cluster --name=kms --config assets/kind-config.yaml
Creating cluster "kms" ...
 ✓ Ensuring node image (kindest/node:v1.29.2) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-kms"
You can now use your cluster with:

kubectl cluster-info --context kind-kms

Have a nice day! 👋

$> kubectl get pod -n kube-system
NAME                                        READY   STATUS    RESTARTS   AGE
coredns-76f75df574-q9nvb                    1/1     Running   0          97s
coredns-76f75df574-vwmxz                    1/1     Running   0          97s
etcd-kms-control-plane                      1/1     Running   0          2m
kindnet-wngbr                               1/1     Running   0          98s
kube-apiserver-kms-control-plane            1/1     Running   0          118s
kube-controller-manager-kms-control-plane   1/1     Running   0          118s
kube-proxy-rvl9z                            1/1     Running   0          98s
kube-scheduler-kms-control-plane            1/1     Running   0          2m
vault-kubernetes-kms-kms-control-plane      1/1     Running   0          116s # vaul-kubernetes-kms pod
```

## Creating Kubernetes secrets encrypted on etcd disk
Time for creating new Kubernetes secrets and check how they are now stored in etcd storage due to a kms encryption provider configured:

```bash
# create any secret
$> kubectl create secret generic secret-encrypted -n default --from-literal=key=value
secret/secret-encrypted created

# show the secret
§>  kubectl get secret secret-encrypted -o json | jq '.data | map_values(@base64d)'
{
  "key": "value"
}

# show secret in etcd storage
$> kubectl -n kube-system exec etcd-kms-control-plane -- sh -c "ETCDCTL_API=3 etcdctl \
    --endpoints=https://127.0.0.1:2379 \
    --cert /etc/kubernetes/pki/etcd/server.crt \
    --key /etc/kubernetes/pki/etcd/server.key \
    --cacert /etc/kubernetes/pki/etcd/ca.crt \
    get /registry/secrets/default/secret-encrypted" | hexdump -C
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
00000180  36 34 1a 59 76 61 75 6c  74 3a 76 31 3a 6d 42 41  |64.Yvault:v1:mBA|  # encrypted value
00000190  4a 47 56 56 35 72 46 78  36 47 4c 4f 62 33 46 50  |JGVV5rFx6GLOb3FP|
000001a0  37 4a 38 73 5a 79 4a 38  2f 68 36 61 48 2b 46 57  |7J8sZyJ8/h6aH+FW|
000001b0  55 46 2f 67 53 68 30 65  41 31 4e 51 45 47 6e 30  |UF/gSh0eA1NQEGn0|
000001c0  5a 30 38 66 6a 59 45 53  30 4c 31 79 35 45 49 50  |Z08fjYES0L1y5EIP|
000001d0  33 67 4c 72 77 61 35 4b  61 44 35 43 63 28 01 0a  |3gLrwa5KaD5Cc(..|
000001e0
```

## Encrypt all existing secrets
You can encrypt all previous existing secrets using:

```bash
$> kubectl get secrets --all-namespaces -o json | kubectl replace -f -
```

## Verify decryption after restart
If we restart the `kube-apiserver` the secrets have been Successfully decrypted:

```bash
$> kubectl delete pod/etcd-kms-control-plane -n kube-system
pod "kube-apiserver-minikube" deleted

# secret has been successfully decrypted
$> kubectl get secret secret-encrypted -o json | jq '.data | map_values(@base64d)'
{
  "key": "value"
}
```

## Teardown
```bash
# kind cluster
$> kind delete cluster -n kms

# vault
$> kill $(pgrep -x vault)
```

## Some last thoughts
For production usage you should consider:

* use HTTPS for the communication between Kubernetes & HashiCorp Vault (see [https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/](https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/))
* deploy the `vault-kubernetes-kms` plugin as a static pod on all control plane nodes
* automate the deployment using your preferred automation method (see [https://falcosuessgott.github.io/vault-kubernetes-kms/integration/](https://falcosuessgott.github.io/vault-kubernetes-kms/integration/))
