FROM alpine:3.19

COPY vault-kubernetes-kms /usr/bin/vault-kubernetes-kms

ENTRYPOINT ["/usr/bin/vault-kubernetes-kms"]