# Integrations
Collection of snippets to automate & deploy the `vault-kubernetes-kms` plugin

## kubeadm
* [https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/#kubeadm-k8s-io-v1beta3-ClusterConfiguration](https://kubernetes.io/docs/reference/config-api/kubeadm-config.v1beta3/#kubeadm-k8s-io-v1beta3-ClusterConfiguration):

```yaml
kind: ClusterConfiguration
apiServer:
  extraArgs:
    encryption-provider-config: "/etc/kubernetes/encryption_provider_config_v2.yaml"
  extraVolumes:
    - name: encryption-config
      hostPath: "/etc/kubernetes/encryption_provider_config_v2.yaml"
      mountPath: "/etc/kubernetes/encryption_provider_config_v2.yaml"
      readOnly: true
      pathType: File
    - name: socket
      hostPath: "/opt/kms"
      mountPath: "/opt/kms"
```

## kind
* [https://kind.sigs.k8s.io/docs/user/configuration/](https://kind.sigs.k8s.io/docs/user/configuration/)

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  # mount encryption provider config available on all cp nodes
  - containerPath: /etc/kubernetes/encryption_provider_config_v2.yaml
    hostPath: scripts/encryption_provider_config_v2.yml
    readOnly: true
    propagation: None
  # vault-kubernetes-kms as a static Pod
  - containerPath: /etc/kubernetes/manifests/vault-kubernetes-kms.yaml
    hostPath: scripts/vault-kubernetes-kms.yml
    readOnly: true
    propagation: None
  # patch kube-apiserver
  kubeadmConfigPatches:
    - |
      kind: ClusterConfiguration
      apiServer:
        extraArgs:
          encryption-provider-config: "/etc/kubernetes/encryption_provider_config_v2.yaml"
        extraVolumes:
        - name: encryption-config
          hostPath: "/etc/kubernetes/encryption_provider_config_v2.yaml"
          mountPath: "/etc/kubernetes/encryption_provider_config_v2.yaml"
          readOnly: true
          pathType: File
        - name: socket
          hostPath: "/opt/kms"
          mountPath: "/opt/kms"
```

## kops
* [https://kops.sigs.k8s.io/manifests_and_customizing_via_api/](https://kops.sigs.k8s.io/manifests_and_customizing_via_api/)

```yaml
kind: Cluster
spec:
  # patch kube-apiserver
  encryptionConfig: true
  # mount encryption provider config available on all cp nodes
  fileAssets:
    - name: scripts/encryption_provider_config_v2.yml
      path: /etc/kubernetes/encryption_provider_config_v2.yaml
      roles:
        - Master
      content: |
        kind: EncryptionConfiguration
        apiVersion: apiserver.config.k8s.io/v1
        resources:
        - resources:
            - secrets
            providers:
            - kms:
                apiVersion: v2
                name: vault-kubernetes-kms
                endpoint: unix:///opt/kms/vaultkms.socket
            - identity: {}
    # vault-kubernetes-kms as a static Pod
    - name: scripts/vault-kubernetes-kms.yml
      path: /etc/kubernetes/manifests/vault-kubernetes-kms.yaml
      roles:
        - Master
      content: |
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
            imagePullPolicy: IfNotPresent
            command:
                - /vault-kubernetes-kms
                - -vault-address=http://172.17.0.1:8200
                - -auth-method=token
                - -token=root
            volumeMounts:
                # mount /opt/kms host directory
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
            # mount /opt/kms host directory
            - name: kms
            hostPath:
                path: /opt/kms
```

## k3s
* Place following file under `var/lib/rancher/k3s/server/manifests/encryption_provider_config_v2.yml` on the server node:

```yaml
{!../scripts/encryption_provider_config_v2.yml!}
```


* Place following file under `var/lib/rancher/k3s/server/manifests/vault-kubernetes-kms.yaml` on the server node, so it gets deployed by `kubelet`

```yaml
{!../scripts/vault-kubernetes-kms.yml!}
```

* bootstrap cluster by running:

```bash
$> k3s server '--kube-apiserver-arg="encryption-provider-config=/etc/kubernetes/encryption_provider_config_v2.yml"'
```

## k3d

```yaml
options:
  k3s:
    extraArgs:
      - arg: --kube-apiserver-arg="encryption-provider-config=/etc/kubernetes/encryption_provider_config_v2.yml"

```

## minikube
```
$>  minikube start \
  --vm-driver="docker" \
  --mount="true" \
  --mount-string="./assets:/etc/kubernetes/manifests" \
  --extra-config="apiserver.encryption-provider-config=/etc/kubernetes/manifests/encryption_provider_config_v2.yml"
```


* AKS
* EKS
* GKE