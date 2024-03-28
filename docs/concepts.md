# Concepts
Read the official [Kubernetes KMS docs](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/) for more details.

## Encryption Request
```mermaid
%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#326ce5', 'primaryTextColor': '#fff', 'textColor': '#000'}}}%%
sequenceDiagram
    participant etcd
    participant kubeapiserver
    participant kmsplugin
    participant externalkms
    kubeapiserver->>kmsplugin: encrypt request
    alt using key hierarchy
        kmsplugin->>kmsplugin: encrypt DEK with local KEK
        kmsplugin->>externalkms: encrypt local KEK with remote KEK
        externalkms->>kmsplugin: encrypted local KEK
        kmsplugin->>kmsplugin: cache encrypted local KEK
        kmsplugin->>kubeapiserver: return encrypt response <br/> {"ciphertext": "<encrypted DEK>", key_id: "<remote KEK ID>", <br/> "annotations": {"kms.kubernetes.io/local-kek": "<encrypted local KEK>"}}
    else not using key hierarchy
        %% current behavior
        kmsplugin->>externalkms: encrypt DEK with remote KEK
        externalkms->>kmsplugin: encrypted DEK
        kmsplugin->>kubeapiserver: return encrypt response <br/> {"ciphertext": "<encrypted DEK>", key_id: "<remote KEK ID>", "annotations": {}}
    end
    kubeapiserver->>etcd: store encrypt response and encrypted DEK
```


## Decryption Request
```mermaid
%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#326ce5', 'primaryTextColor': '#fff', 'textColor': '#000'}}}%%
sequenceDiagram
    participant kubeapiserver
    participant kmsplugin
    participant externalkms
    %% if local KEK in annotations, then using hierarchy
    alt encrypted local KEK is in annotations
      kubeapiserver->>kmsplugin: decrypt request <br/> {"ciphertext": "<encrypted DEK>", key_id: "<key_id gotten as part of EncryptResponse>", <br/> "annotations": {"kms.kubernetes.io/local-kek": "<encrypted local KEK>"}}
        alt encrypted local KEK in cache
            kmsplugin->>kmsplugin: decrypt DEK with local KEK
        else encrypted local KEK not in cache
            kmsplugin->>externalkms: decrypt local KEK with remote KEK
            externalkms->>kmsplugin: decrypted local KEK
            kmsplugin->>kmsplugin: decrypt DEK with local KEK
            kmsplugin->>kmsplugin: cache decrypted local KEK
        end
        kmsplugin->>kubeapiserver: return decrypt response <br/> {"plaintext": "<decrypted DEK>", key_id: "<remote KEK ID>", <br/> "annotations": {"kms.kubernetes.io/local-kek": "<encrypted local KEK>"}}
    else encrypted local KEK is not in annotations
        kubeapiserver->>kmsplugin: decrypt request <br/> {"ciphertext": "<encrypted DEK>", key_id: "<key_id gotten as part of EncryptResponse>", <br/> "annotations": {}}
        kmsplugin->>externalkms: decrypt DEK with remote KEK (same behavior as today)
        externalkms->>kmsplugin: decrypted DEK
        kmsplugin->>kubeapiserver: return decrypt response <br/> {"plaintext": "<decrypted DEK>", key_id: "<remote KEK ID>", <br/> "annotations": {}}
    end
```
