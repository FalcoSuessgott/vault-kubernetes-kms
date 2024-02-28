FROM golang:1.20 as builder 
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . . 
RUN go build -v -o /usr/local/bin/vault-kubernetes-kms cmd/main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:508313ed9307b1efb249cac49a0c07ab68192f9044b64ab33625c30dbe59c3f2
WORKDIR /
COPY --from=builder /usr/local/bin/vault-kubernetes-kms .

ENTRYPOINT ["vault-kubernetes-kms"]