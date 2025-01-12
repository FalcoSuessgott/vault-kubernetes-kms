resource "tls_private_key" "tls_client_priv_key" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "tls_cert_request" "tls_client_csr" {
  private_key_pem = tls_private_key.tls_client_priv_key.private_key_pem

  subject {
    common_name = "vault-kubernetes-kms-tls-client"
  }

  dns_names    = ["localhost"]
  ip_addresses = ["127.0.0.1"]
}

resource "tls_locally_signed_cert" "sign_tls_client_csr" {
  cert_request_pem      = tls_cert_request.tls_client_csr.cert_request_pem
  ca_private_key_pem    = tls_private_key.tls_ca_priv_key.private_key_pem
  ca_cert_pem           = tls_locally_signed_cert.sign_tls_ca_csr.cert_pem
  validity_period_hours = 86000
  is_ca_certificate     = false

  allowed_uses = [
    "digital_signature",
    "client_auth",
  ]
}

resource "local_file" "tls_client_priv_key" {
  filename = "${path.module}/../output/tls-client-ca.key"
  content  = tls_private_key.tls_client_priv_key.private_key_pem
}

resource "local_file" "tls_client_cert" {
  filename = "${path.module}/../output/tls-client-ca.crt"
  content  = tls_locally_signed_cert.sign_tls_client_csr.cert_pem
}
