FROM golang:1.20 as builder 
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . . 
RUN go build -v -o /usr/local/bin/vault-kubernetes-kms cmd/main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:0d6ada5faececed5cd3f99baa08e4109934f2371c0d81b3bff38924fe1deea05
WORKDIR /
COPY --from=builder /usr/local/bin/vault-kubernetes-kms .

ENTRYPOINT ["vault-kubernetes-kms"]