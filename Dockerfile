FROM golang:1.22.2 as builder
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -o /vault-kubernetes-kms cmd/main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:6dcc833df2a475be1a3d7fc951de90ac91a2cb0be237c7578b88722e48f2e56f
WORKDIR /
COPY --from=builder /vault-kubernetes-kms .

ENTRYPOINT ["/vault-kubernetes-kms"]
