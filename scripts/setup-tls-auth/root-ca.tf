resource "tls_private_key" "root_ca_priv_key" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "tls_self_signed_cert" "root_cert" {
  private_key_pem = tls_private_key.root_ca_priv_key.private_key_pem

  subject {
    common_name = "vault-kubernetes-kms-root-ca"
  }

  validity_period_hours = 86000

  is_ca_certificate = true

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "cert_signing",
    "server_auth",
    "client_auth",
  ]
}

resource "local_file" "root_key" {
  filename = "${path.module}/../output/rootCA.key"
  content  = tls_private_key.root_ca_priv_key.private_key_pem
}

resource "local_file" "root_cert" {
  filename = "${path.module}/../output/rootCA.crt"
  content  = tls_self_signed_cert.root_cert.cert_pem
}
