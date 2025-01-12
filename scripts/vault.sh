#!/usr/bin/env bash
set -x

command -v vault >/dev/null 2>&1 || { echo "vault is not installed.  Aborting." >&2; exit 1; }

# kill any remaining vault instances
kill $(pgrep -x vault) || true

# start developemnt vault
nohup vault server -dev \
   -dev-listen-address=0.0.0.0:8200 \
   -dev-root-token-id=root \
   -dev-tls \
   -dev-tls-cert-dir certs/ 2> /dev/null &

sleep 3

# auth to vault
export VAULT_ADDR="https://127.0.0.1:8200"
export VAULT_SKIP_VERIFY="true"
export VAULT_TOKEN="root"

# enable transit engine
vault secrets enable transit
vault write -f transit/keys/kms

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
