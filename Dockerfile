FROM golang:1.20 as builder 
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . . 
RUN go build -v -o /usr/local/bin/vault-kubernetes-kms cmd/main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:2eb0c793360d26aace9b88fb9cf1a4f680140e7fb7d68d86de1fe63dbc1a7660
WORKDIR /
COPY --from=builder /usr/local/bin/vault-kubernetes-kms .

ENTRYPOINT ["vault-kubernetes-kms"]