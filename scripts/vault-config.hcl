listener "tcp" {
  address                  = "0.0.0.0:8400"
  tls_cert_file            = "./scripts/output/vault.crt"
  tls_key_file             = "./scripts/output/vault.key"
  tls_disable              = false
  tls_disable_client_certs = false
}
