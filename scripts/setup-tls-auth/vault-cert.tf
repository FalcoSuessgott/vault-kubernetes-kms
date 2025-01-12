resource "tls_private_key" "vault_priv_key" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "tls_cert_request" "vault_csr" {
  private_key_pem = tls_private_key.vault_priv_key.private_key_pem

  subject {
    common_name = "vault-kubernetes-kms-vault-cert"
  }

  dns_names    = ["localhost"]
  ip_addresses = ["127.0.0.1"]
}

# 4. sign the csr using the root ca
resource "tls_locally_signed_cert" "sign_vault_csr" {
  cert_request_pem      = tls_cert_request.vault_csr.cert_request_pem
  ca_private_key_pem    = tls_private_key.root_ca_priv_key.private_key_pem
  ca_cert_pem           = tls_self_signed_cert.root_cert.cert_pem
  validity_period_hours = 86000
  is_ca_certificate     = false

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
    "client_auth",
  ]
}

resource "local_file" "vault_cert" {
  filename = "${path.module}/../output/vault.crt"
  content  = tls_locally_signed_cert.sign_vault_csr.cert_pem
}

resource "local_file" "vault_key" {
  filename = "${path.module}/../output/vault.key"
  content  = tls_private_key.vault_priv_key.private_key_pem
}
