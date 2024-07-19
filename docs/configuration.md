# Configuration
Enabling KMS Encryption in your Cluster involves 3 steps:

1. Preparing your Vaults Transit Engine & Auth Method
2. Deploying the `vault-kubernetes-kms` Plugin
3. Enabling the encryption provider configuration for the `kube-apiserver`.

## Preparing HashiCorp  Vault
The `vault-kms-plugin` requires a Vault Authentication, that allows encrypting and decrypting data with a given Transit Key.

### Transit Engine
!!! tip
    You can also perform these steps using Vaults UI or Vaults Terraform Provider (recommended).

The following steps enable a transit engine `transit` and create transit key `kms`:
```bash
$> export VAULT_ADDR="https://vault.tld.de"   # change to your Vaults API Address
$> export VAULT_TOKEN="hhvs.XXXX"             # change to a token allowed to create a transit engine and a transit key
$> vault secrets enable transit
$> vault write -f transit/keys/kms
```

!!! warning
    Absolutely make sure to either Backup (snapshot) your Vault in order to recover from failure, or export your KMS key prior usage (option: `allow-plaintext-backup`).

    **If you loose the KMS key or recreate it, the kube-apiserver will not be able to decrypt any secrets.**


### Vault Policy
The following Vault Policy lists the API paths required for the `vault-kubernetes-kms` plugin:

!!! note
    This Policy assumes the transit engine is mounted at `transit` with a key named `kms`.
    In case your configuration differs, you will have to update the policy accordingly.

```hcl
# kms-policy.hcl
# lookup the current tokens ttl for token renewal, is also in Vaults default policy
path "auth/token/lookup-self" {
    capabilities = ["read"]
}

# encrypt any data using the transit key
path "transit/encrypt/kms" {
   capabilities = [ "update" ]
}

# decrypt any data using the transit key
path "transit/decrypt/kms" {
   capabilities = [ "update" ]
}

# get the transit keys key versions for KMS key rotation
path "transit/keys/kms" {
   capabilities = [ "read" ]
}
```

You can create the policy using `vault policy write kms ./kms-policy.hcl`.

### Vault Auth
`vault-kubernetes-kms` suppors Token & Approle Auth. Kubernetes Auth was removed (see [falcosuessgott/vault-kubernetes-kms#81](https://github.com/FalcoSuessgott/vault-kubernetes-kms/issues/81)), since a static pod cannot reference any other API objects, such as Service Account, which are required for Kubernetes Auth.

### Approle Auth

```bash
# Follow https://developer.hashicorp.com/vault/docs/auth/approle
# enable approle and create a role
$> vault auth enable approle
$> vault write auth/approle/role/kms token_num_uses=0 token_period=3600 token_policies=kms

# get the role ID from the output of
$> vault read auth/approle/role/kms/role-id

# get the secret ID from the output of
$> vault write -f auth/approle/role/kms/secret-id
```

### Token Auth
It is recommended, that the Vault token used for authentication is **an orphaned and periodic token**. Periodic tokens can be renewed within the period. An orphan token does not have a parent token and will not be revoked when the token that created it expires.

```bash
# get the token from the output
$> vault token create -orphan -policy="kms" -period=24h
```

## Deploying `vault-kubernetes-kms`
!!! info
    The plugin creates a Unix-Socket that is referenced in a `EncryptionConfiguration` manifest, which the `kube-apiserver` points to.

    **That means, that the `kube-apiserver` will not properly start if the plugin is not up & running. To ensure the plugin is running before the `kube-apiserver` it has to be deployed as a static Pod.** To do so, we use `priorityClassName: system-node-critical` in the plugins manifest, to mark the Pod as critical ([https://kubernetes.io/docs/tasks/administer-cluster/guaranteed-scheduling-critical-addon-pods/#marking-pod-as-critical](https://kubernetes.io/docs/tasks/administer-cluster/guaranteed-scheduling-critical-addon-pods/#marking-pod-as-critical)).


### CLI Args & Environment Variables
List of required and optional CLI args/env vars. **Furthermore, all of Vaults [Env Vars](https://developer.hashicorp.com/vault/docs/commands#environment-variables) are supported**:

**Vault Server**:

* **(Required)**: `-vault-address` (`VAULT_KMS_VAULT_ADDR`)
* **(Optional)**: `-vault-namespace` (`VAULT_KMS_VAULT_NAMESPACE`)

**Vault Transit Engine**:

* **(Optional)**: `-transit-mount` (`VAULT_KMS_TRANSIT_MOUNT`); default: `"transit"`
* **(Optional)**: `-transit-key` (`VAULT_KMS_TRANSIT_KEY`); default: `"kms"`


**If Vault Token Auth**:

* **(Required)**: `-auth-method="token"` (`VAULT_KMS_AUTH_METHOD`)
* **(Required)**: `-token` (`VAULT_KMS_VAULT_TOKEN`)

**If Vault Approle Auth**:

* **(Required)**: `-auth-method="approle"` (`VAULT_KMS_AUTH_METHOD`)
* **(Required)**: `-approle-role-id` (`VAULT_KMS_APPROLE_ROLE_ID`)
* **(Required)**: `-approle-secret-id` (`VAULT_KMS_APPROLE_SECRET_ID`)
* **(Optional)**: `-approle-mount` (`VAULT_KMS_APPROLE_MOUNT`); default: `"approle"`

**Optional**:

* **(Optional)**: `-socket` (`VAULT_KMS_SOCKET`); default: `unix:///opt/kms/vaultkms.socket"`
* **(Optional)**: `-debug` (`VAULT_KMS_DEBUG`)

### Example Vault Token Auth

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: vault-kubernetes-kms
  namespace: kube-system
spec:
  priorityClassName: system-node-critical
  hostNetwork: true
  containers:
    - name: vault-kubernetes-kms
      image: falcosuessgott/vault-kubernetes-kms:latest
      # either specify CLI Args or env vars (look above)
      command:
        - /vault-kubernetes-kms
        - -vault-address=https://vault.server.d
        - -auth-method=token
        - -token=hvs.ABC123
      volumeMounts:
        - name: kms
          mountPath: /opt/kms
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: "2"
          memory: 1Gi
  volumes:
    - name: kms
      hostPath:
        path: /opt/kms
```

### Example Vault Approle Auth

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: vault-kubernetes-kms
  namespace: kube-system
spec:
  priorityClassName: system-node-critical
  hostNetwork: true
  containers:
    - name: vault-kubernetes-kms
      image: falcosuessgott/vault-kubernetes-kms:latest
        # either specify CLI Args or env vars (look above)
      command:
        - /vault-kubernetes-kms
        - -vault-address=https://vault.server.d
        - -auth-method=approle
        - -approle-role-id=XXXX
        - -approle-secret-id=XXXX
      volumeMounts:
        - name: kms
          mountPath: /opt/kms
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: "2"
          memory: 1Gi
  volumes:
    - name: kms
      hostPath:
        path: /opt/kms
```

### Example TLS Configuration
It is recommended, to specify the CA cert that issued the vault server certificate. To do so, you would have to create a Kubernetes secret containing Vaults CA certificate PEM encoded.

Example:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: vault-kubernetes-kms
  namespace: kube-system
spec:
  priorityClassName: system-node-critical
  hostNetwork: true
  containers:
    - name: vault-kubernetes-kms
      image: falcosuessgott/vault-kubernetes-kms:latest
      # either specify CLI Args or env vars (look above)
      command:
        - /vault-kubernetes-kms
        - --vault-address=https://vault.server.de
        - -auth-method=token
        - -token=XXXX
      env:
        # add vaults CA file via env vars
        - name: VAULT_CACERT
          value: /opt/ca/ca.crt
      volumeMounts:
        # mount the hostpath volume to enable the kms socket to the node
        - name: kms
          mountPath: /opt/kms
        # mount the ca cert under /opt/ca/ca.crt
        - name: ca-cert
          mountPath: /opt/ca/ca.crt
          subPath: ca.crt
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: "2"
          memory: 1Gi
  volumes:
    - name: kms
      hostPath:
        path: /opt/kms
    - name: ca-cert
      secret:
        secretName: ca-cert # secret name containing the Vault CA certificate
        items:
          - key: ca.crt # key of the PEM encoded certificate
            path: ca.crt
```

After applying you check the pods logs for any errors:

```bash
$> kubectl logs -n kube-system vault-kubernetes-kms
{"level":"info","timestamp":"2024-01-31T13:18:24.852Z","caller":"cmd/main.go:111","message":"starting kms plugin","socket":"unix:///opt/vaultkms.socket","debug":false,"vault-address":"http://host.minikube.internal:8200","vault-namespace":"","vault-token":"","vault-k8s-mount":"kubernetes","vault-k8s-role":"kms","vault-transit-mount":"transit","vault-transit-key":"kms"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:144","message":"Successfully authenticated to vault"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:151","message":"Successfully created unix socket","socket":"/opt/vaultkms.socket"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:158","message":"Listening for connection"}
{"level":"info","timestamp":"2024-01-31T13:18:24.898Z","caller":"cmd/main.go:169","message":"Successfully registered kms plugin"}
```

## Enabling the encryption provider configuration for the `kube-apiserver`.
### Determine which KMS version to use
Since the `vault-kms-plugin` supports both KMS versions, you would have to determine, which KMS Plugin version works for your setup:

From the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin):

!!! note
    The version of Kubernetes that you need depends on which KMS API version you have selected. Kubernetes recommends using KMS v2.

    **If you selected KMS API v2, you should use Kubernetes v1.29** (if you are running a different version of Kubernetes that also supports the v2 KMS API, switch to the documentation for that version of Kubernetes).

    **If you selected KMS API v1 to support clusters prior to version v1.27 or if you have a legacy KMS plugin that only supports KMS v1, any supported Kubernetes version will work**. This API is deprecated as of Kubernetes v1.28. Kubernetes does not recommend the use of this API.

### Encryption Provider configuration
Copy the appropriate encryption provider configuration to your control plane nodes (e.g. `/opt/kms/encryption_provider_config.yml`):

#### KMS Plugin v1
```yaml
{!../scripts/encryption_provider_config_v1.yml!}
```


#### KMS Plugin v2
```yaml
{!../scripts/encryption_provider_config_v2.yml!}
```

### Modify the `kube-api-server` Manifest
Last but not least, you would have to enable the encryption provider config for the `kube-apiserver`.
This steps depends on wether your control plane components run as a systemd daemon or as static Pod on your control plane nodes (usually located at `/etc/kubernetes/manifests`).

**Either way, the following changes need to be done:**

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

Also you will have to mount the `/opt` directory, for accessing the socket, that is created by the plugin and the encryption provider config:

```yaml
# ....
volumeMounts:
    - name: kms
      mountPath: /opt/kms
volumes:
  - name: kms
    hostPath:
      path: /opt/kms
# ....
```

Once changes are made, the `kube-apiserver` will restart (if static pod) or you restart the SystemD Unit.

You then can use `watch` to monitor the pods:

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

You should now see in the plugin logs that encryption and decryption requests are coming:

```bash
$> kubectl logs -n kube-system vault-kubernetes-kms
{"level":"info","timestamp":"2024-01-31T13:31:29.159Z","caller":"kms/plugin.go:112","message":"encryption request","request_id":"f1eb6db8-390e-4bd4-8481-c56e46c1d685"}
```

## Verify Secret Encryption
Finally, create a secret to verify everything works correctly:

```bash
$> kubectl create secret generic secret-encrypted -n default --from-literal=key=value
secret/secret-encrypted created
```

**If the secret creation fails, something does not work!**

You could also check the etcd storage for the encrypted data:

```bash
# this works only on minikube, you would have to update the params according to your cluster config
$> kubectl -n kube-system exec etcd-minikube -- sh -c "ETCDCTL_API=3 etcdctl
    --endpoints=https://127.0.0.1:2379 \
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
```
