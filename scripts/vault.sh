#!/usr/bin/bash

kill $(pgrep -x vault) || true 
nohup vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=root 2> /dev/null &
sleep 3

export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_SKIP_VERIFY="true"
export VAULT_TOKEN="root"

vault secrets enable transit
vault write -f transit/keys/kms