FROM golang:1.23 AS builder
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -o /vault-kubernetes-kms main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:b033683de7de51d8cce5aa4b47c1b9906786f6256017ca8b17b2551947fcf6d8
WORKDIR /
COPY --from=builder /vault-kubernetes-kms .

ENTRYPOINT ["/vault-kubernetes-kms"]
