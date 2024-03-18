# Getting Started
!!! tip
    This Guide will walk you through the required steps of installing and configuring the `vault-kms-plugin` for Kubernetes.
    
    Checkout [https://falcosuessgott.github.io/hashicorp-vault-playground/home/](https://falcosuessgott.github.io/hashicorp-vault-playground/home/) a project that helps you quickly setting up Kubernetes HashiCorp Vault

!!! warning
    This guide uses the new version of the Kubernetes KMS Plugin API, which was introduced in Kubernetes v1.29.0 ([https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#kms-v2](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#kms-v2)).

## Requirements
In order to run this guide, you will need to have `minikube`, `kubectl` and `vault` installed on your system. 
Also it is recommended that you are using either MacOS or Linux as the operating system.

## Overview
This guide will:

1. Start `minikube` locally, bridge it to your localhost in order to access any application running on your system
2. Show that secrets are per default unencrypted stored in etcd
3. Start and configure Vaults Transit Engine (used for encrypted Kubernetes secrets) and the Kubernetes auth method (so the plugins service account can authenticate to Vault), as well as a KMS policy.
4. Deploy the KMS Plugin as a single Pod
5. Configure the `kube-apiserver` to use the encryption provider and restart the kube-apiserver
6. Show secrets are now encrypted stored in etcd
7. Encrypt all previously existing Secrets
8. Show decryption works after `kube-apiserver` performs a restart

## 1. Minikube Setup
Start `minikube`, bridge it to localhost, to access application running locally and enforce `v1.29.0` for KMSv2 Plugin usage:

```bash
$> minikube start --driver=docker \
    --ports=8443:8443 \
    --listen-address=0.0.0.0 \
    --cni=bridge \
    --kubernetes-version=v1.29.0

# you should now be able to run kubectl
$> kubectl get po -A                                                  
NAMESPACE     NAME                               READY   STATUS             RESTARTS        AGE
kube-system   coredns-76f75df574-7vpvc           0/1     Running            1 (11m ago)     19m
kube-system   etcd-minikube                      1/1     Running            1 (11m ago)     20m
kube-system   kube-apiserver-minikube            0/1     Running            0               11m
kube-system   kube-controller-manager-minikube   1/1     Running            1 (11m ago)     20m
kube-system   kube-proxy-bvvf5                   1/1     Running            1 (11m ago)     19m
kube-system   kube-scheduler-minikube            1/1     Running            1 (11m ago)     20m
```

## 2. Verify Secrets are unencrypted in etcd
```bash
# create any secret
$> kubectl create secret generic secret-unencrypted -n default --from-literal=key=value      
secret/secret-unencrypted created

# show the secret
$> kubectl get secret secret-unencrypted -o json | jq '.data | map_values(@base64d)'            
{
  "key": "value"
}

# list unenctypted secret in etcd
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

## 3. Vault Setup
Start Vault, configure Vaults Transit engine and configure the kubernetes auth method for minikube:

```bash
# start developemnt vault with root token "root"
$> vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=root

# perform these steps in another terminal
$> export VAULT_ADDR="http://127.0.0.1:8200"
$> export VAULT_SKIP_VERIFY="true"
$> export VAULT_TOKEN="root"

# verify connectivity 
$> vault status
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.15.0
Build Date      2023-09-22T16:53:10Z
Storage Type    inmem
Cluster Name    vault-cluster-6aa88b66
Cluster ID      4ba3d441-4bad-be25-d1d3-cb5516952648
HA Enabled      false

# enable transit engine
$> vault secrets enable transit
Success! Enabled the transit secrets engine at: transit/

$> vault write -f transit/keys/kms
Key                       Value
---                       -----
allow_plaintext_backup    false
auto_rotate_period        0s
deletion_allowed          false
derived                   false
exportable                false
imported_key              false
keys                      map[1:1706706145]
latest_version            1
min_available_version     0
min_decryption_version    1
min_encryption_version    0
name                      kms
supports_decryption       true
supports_derivation       true
supports_encryption       true
supports_signing          false
type                      aes256-gcm96

# create SA, SA token and CRB, this service account is used to verify other kubernetes service accounts
$> cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: vault-auth
  namespace: kube-system
---

apiVersion: v1
kind: Secret
metadata:
  name: vault-auth
  namespace: kube-system
  annotations:
    kubernetes.io/service-account.name: vault-auth
type: kubernetes.io/service-account-token
---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
   name: role-tokenreview-binding
   namespace: kube-system
roleRef:
   apiGroup: rbac.authorization.k8s.io
   kind: ClusterRole
   name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: vault-auth
  namespace: kube-system
EOF 
serviceaccount/vault-auth created
secret/vault-auth created
clusterrolebinding.rbac.authorization.k8s.io/role-tokenreview-binding created

# enable 8s auth on vault
$> vault auth enable kubernetes

# configure k8s auth for minikube on vault
$> token=$(kubectl get secret -n kube-system vault-auth -o go-template='{{ .data.token }}' | base64 --decode)
$> ca_cert=$(kubectl get cm kube-root-ca.crt -o jsonpath="{['data']['ca\.crt']}")
$> vault write auth/kubernetes/config \
    token_reviewer_jwt="${token}" \
    kubernetes_host="https://127.0.0.1:8443" \
    kubernetes_ca_cert="${ca_cert}"
Success! Data written to: auth/kubernetes/config

# create a k8s auth role "kms" on vault
$> vault write auth/kubernetes/role/kms \
    bound_service_account_names=default \
    bound_service_account_namespaces=kube-system \
    policies=kms ttl=24h
Success! Data written to: auth/kubernetes/role/kms

# write vault policy that is used when authentication with the "kms" auth role
$> vault policy write kms - <<EOF
# perform a simple vault login test
path "auth/token/lookup-self" {
    capabilities = ["read"]
}

# encrypt 
path "transit/encrypt/kms" {
   capabilities = [ "update" ]
}

# decrypt
path "transit/decrypt/kms" {
   capabilities = [ "update" ]
}

# get key version
path "transit/keys/kms" {
   capabilities = [ "read" ]
}
EOF
Success! Uploaded policy: kms
```

## 4. KMS Plugin Deployment
`minikube` and `vault` are now running on your system and can communite with eath other.

Now we can deploy the actual `vault-kubernetes-kms` plugin running as a Pod (in production this should be a static pod on every node):

```bash
# apply the manifest
$> cat <<EOF | kubectl apply -f  - 
apiVersion: v1
kind: Pod
metadata:
  name: vault-kubernetes-kms
  namespace: kube-system
spec:
  containers:
    - name: vault-kubernetes-kms
      image: falcosuessgott/vault-kubernetes-kms:v0.0.3
      command:
        - /vault-kubernetes-kms
        - --vault-address=https://host.minikube.internal
        - --vault-k8s-mount=kubernetes
        - --vault-k8s-role=kms
      volumeMounts:
        # mouunt /opt/kms host directory
        - name: kms
          mountPath: /opt/kms
  volumes:
    # mouunt /opt/kms host directory
    - name: kms
      hostPath:
        path: /opt/kms
EOF 

# see the logs wether k8s auth was successful
$> kubectl logs -n kube-system vault-kubernetes-kms
{"level":"info","timestamp":"2024-01-31T13:18:24.852Z","caller":"cmd/main.go:111","message":"starting kms plugin","socket":"unix:///opt/vaultkms.socket","debug":false,"vault-address":"http://host.minikube.internal:8200","vault-namespace":"","vault-token":"","vault-k8s-mount":"kubernetes","vault-k8s-role":"kms","vault-transit-mount":"transit","vault-transit-key":"kms"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:144","message":"Successfully authenticated to vault"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:151","message":"Successfully created unix socket","socket":"/opt/vaultkms.socket"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:158","message":"Listening for connection"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:169","message":"Successfully registered kms plugin"}
```

## 5. `kube-apiserver` Configuration 
Last but not least, you will have to configure you kube-apiserver to start encrypting secrets, by providing an encryption provider config and update the kube-apiserver command:

```bash
$> minikube ssh
minikube> vim.tiny /opt/kms/encryption_provider_config.yml
  ---
  apiVersion: apiserver.config.k8s.io/v1
  kind: EncryptionConfiguration
  resources:
    - resources:
        - secrets
      providers:
        - kms:
            apiVersion: v2
            name: vault 
            endpoint: unix:///opt/kms/vaultkms.socket
        - identity: {}
minikube> sudo -i
minikube> vim.tiny /etc/kubernetes/manifests/kube-apiserver.yaml
```

You will have to add the `encryption-provider-config` arg to the `kube-apiserver` command, pointing to the encryption provider config copied to minikube: 

```yaml
# ...
spec:
  containers:
  - command:
    - kube-apiserver
    # enabling the encryption provider config
    - --encryption-provider-config=/opt/kms/encryption_provider_config.yml
# ...
```

Also you will have to mount the `/opt/kms` directory, for accessing the socket, that is created by the plugin and the encryption provider config:

```yaml
# ...
volumeMounts:
  - name: socket
    mountPath: /opt/kms
# ...
# mount kms socket
volumes:
  - name: socket
    hostPath:
      path: /opt/kms
# ...
```

After performing these changes, the `kube-apiserver` will restart itself, since its a static Pod.
You can also delete the pod by running: `kubectl delete pod/kube-apiserver-minikube -n kube-system`.

You can use `watch` to monitor the pods:

```bash
$> watch kubectl get pod -n kube-system
NAME                               READY   STATUS    RESTARTS       AGE
coredns-76f75df574-dwtfv           1/1     Running   0              151m
etcd-minikube                      1/1     Running   0              152m
kube-apiserver-minikube            0/1     Running   1              50s  # restarted
kube-controller-manager-minikube   1/1     Running   0              152m
kube-proxy-hqpmw                   1/1     Running   0              151m
kube-scheduler-minikube            1/1     Running   0              152m
storage-provisioner                1/1     Running   7 (118m ago)   152m
vault-kubernetes-kms               1/1     Running   0              49m
```

## 6. Verify Secrets are now encrypted now in etcd
```bash
# create a new secret that is going to be encrypted in etcd
$> kubectl create secret generic secret-encrypted -n default --from-literal=key=value
secret/secret-encrypted created

# proof secrets are now encrypted
$> kubectl -n kube-system exec etcd-minikube -- sh -c "ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
    --cert /var/lib/minikube/certs/etcd/server.crt \
    --key /var/lib/minikube/certs/etcd/server.key \
    --cacert /var/lib/minikube/certs/etcd/ca.crt \
    get /registry/secrets/default/secret-encrypted" | hexdump -C 
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 73 65 63 72 65 74  |s/default/secret|
00000020  2d 65 6e 63 72 79 70 74  65 64 0a 6b 38 73 3a 65  |-encrypted.k8s:e|
00000030  6e 63 3a 6b 6d 73 3a 76  32 3a 76 61 75 6c 74 3a  |nc:kms:v2:vault:|
00000040  0a a4 02 2f 0b 6f 44 78  f8 9b 2c 23 b7 d8 99 e0  |.../.oDx..,#....|
00000050  6f 2b 71 48 00 27 08 31  58 c2 2d 9c 8c 41 54 87  |o+qH.'.1X.-..AT.|
00000060  cd 38 7e 90 78 ea 5d 3d  81 5e d4 67 ac f9 11 25  |.8~.x.]=.^.g...%|
00000070  ca eb 68 f3 ae 43 e0 eb  ce 0f fa dc d2 97 91 bb  |..h..C..........|
00000080  e4 04 2f fe 7e f7 83 0f  ef cc 4c 5e 41 f2 3f 42  |../.~.....L^A.?B|
00000090  5a 47 4d e4 3b 6d dc 78  e2 a3 65 f8 bb 84 88 e5  |ZGM.;m.x..e.....|
000000a0  9f 34 1f 53 d2 2a 59 8f  ac 77 a4 58 57 e9 4b 06  |.4.S.*Y..w.XW.K.|
000000b0  f8 e9 80 f1 cf 96 aa 50  1a 24 1c 6a f6 6c e7 2d  |.......P.$.j.l.-|
000000c0  58 ec 30 13 27 6c 4d 43  f5 60 07 8d 11 6f 43 4c  |X.0.'lMC.`...oCL|
000000d0  ae 2b f0 69 01 18 05 a0  22 9b e9 9b 10 c6 83 7f  |.+.i....".......|
000000e0  bb 5c 3e 89 cb 33 68 52  fc 16 c0 37 0a f9 8e 5d  |.\>..3hR...7...]|
000000f0  7c 88 4f cd 02 f1 94 c9  69 52 d7 bc 61 7d b0 aa  ||.O.....iR..a}..|
00000100  bd 4e 7b a1 d9 91 79 17  c8 2a 3d ec 1c a0 60 8e  |.N{...y..*=...`.|
00000110  73 1c 1e 4e 1b 09 81 fb  3a 2b 6c 1c a4 87 7c 3f  |s..N....:+l...|?|
00000120  f2 6a 21 1b f8 42 d4 33  57 64 da be 47 43 a8 92  |.j!..B.3Wd..GC..|
00000130  09 95 61 1b cd 97 5c 30  f1 e5 bf 21 ba 82 21 68  |..a...\0...!..!h|
00000140  3a 14 8b e9 0e 8a 83 6b  ed de 24 f3 5b fd 02 f0  |:......k..$.[...|
00000150  bd 22 b1 ea f3 15 13 9d  c2 a9 01 cf 36 78 5a 77  |."..........6xZw|
00000160  fd 83 fe 46 0e 49 bf 12  0a 31 37 30 36 37 30 37  |...F.I...1706707|
00000170  37 37 34 1a 59 76 61 75  6c 74 3a 76 31 3a 68 66  |774.Yvault:v1:hf|
00000180  54 61 73 58 37 39 4c 63  38 37 68 7a 38 48 77 31  |TasX79Lc87hz8Hw1|
00000190  6a 4e 6d 57 6a 6f 65 56  39 4d 55 73 43 4c 2f 74  |jNmWjoeV9MUsCL/t|
000001a0  6d 34 6d 78 4e 6a 41 46  2b 51 51 4a 72 36 4c 6b  |m4mxNjAF+QQJr6Lk|
000001b0  36 52 69 6a 32 7a 62 57  73 57 44 44 70 65 30 6b  |6Rij2zbWsWDDpe0k|
000001c0  39 68 59 72 4a 4b 39 2f  55 6c 30 69 79 42 28 01  |9hYrJK9/Ul0iyB(.|
000001d0  0a                                                |.|
000001d1

# see the corresponding encryption request in the plugin logs
$> kubectl logs -n kube-system vault-kubernetes-kms                                                              
{"level":"info","timestamp":"2024-01-31T13:31:29.159Z","caller":"kms/plugin.go:112","message":"encryption request","request_id":"f1eb6db8-390e-4bd4-8481-c56e46c1d685"}
```

## 7. Encrypt existing secrets
You can encrypt all previous existing secrets using: 

```bash
$> kubectl get secrets --all-namespaces -o json | kubectl replace -f -`
```

## 8. Verify decryption after restart
If we restart the kube-apiserver the secrets have been Successfully decrypted:

```bash
$> kubectl delete pod/kube-apiserver-minikube -n kube-system    
pod "kube-apiserver-minikube" deleted

# secrets have been successfully decrypted
$> kubectl get secret secret-unencrypted -o json | jq '.data | map_values(@base64d)'            
{
  "key": "value"
}
```

## Some last thoughts
For production usage you should consider:

* deploy the `vault-kubenetes-kms` Pod using a dedicated Service Account, instead of `default` (also adjust the kubernetes auth role)
* use HTTPS for the communication between Kubernetes & HashiCorp Vault (see [https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/](https://falcosuessgott.github.io/vault-kubernetes-kms/configuration/))
* deploy the `vault-kubernetes-kms` plugin as a static pod on all control plane nodes
* automate the deployment using your preferred automation method
