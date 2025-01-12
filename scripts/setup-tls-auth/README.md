`terraform` code to setup TLS certs for Vaults TLS auth method.

Once applied, this code will create (in `../output/`):

- a RootCA (key: `rootCA.key`, cert: `rootCA.crt`)
- a Vault Server Cert (key: `vault.key`, cert: `vault.crt`)
- an Intermediate CA (key: `tls-ca.key`, cert: `tls-ca.crt`)
- a Client certificate (key: `tls-client-ca.key`, cert: `tls-client-ca.key`)

The chain of trust is:

- `RootCA` signs `Vault Server Cert`
- `RootCA` -> `Intermediate CA` -> `Client cert`

# Usage
```bash
terraform init
terraform apply
```

# Verify
```bash
# verify root CA -> vault server cert
openssl verify -CAfile rootCA.crt vault.crt

# verify Root CA -> Intermediate CA -> TLS Client Cert
openssl verify -CAfile rootCA.crt -untrusted tls-ca.crt tls-client-ca.crt
```
