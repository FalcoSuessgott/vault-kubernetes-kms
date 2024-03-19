#!/usr/bin/bash
set -x
# kill any remaining vault instances
kill $(pgrep -x vault) || true

# start developemnt vault
nohup vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=root 2> /dev/null &
sleep 3

# auth to vault
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_SKIP_VERIFY="true"
export VAULT_TOKEN="root"

# enable transit engine
vault secrets enable transit
vault write -f transit/keys/kms

# create sa, secret and crb
#kubectl apply -f scripts/rbac.yml

# # enable k8s auth on kubernetes
# token=$(kubectl get secret -n kube-system vault-auth -o go-template='{{ .data.token }}' | base64 --decode)
# ca_cert=$(kubectl get cm kube-root-ca.crt -o jsonpath="{['data']['ca\.crt']}")

# # enabel k8s auth on vault
# vault auth enable kubernetes
# vault write auth/kubernetes/config \
#     token_reviewer_jwt="${token}" \
#     kubernetes_host="https://127.0.0.1:8443" \
#     kubernetes_ca_cert="${ca_cert}"

# vault write auth/kubernetes/role/kms bound_service_account_names=default bound_service_account_namespaces=kube-system policies=kms ttl=24h

# write vault policy
vault policy write kms - <<EOF
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
