FROM golang:1.20 as builder 
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . . 
RUN go build -v -o /vault-kubernetes-kms cmd/main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:4a2c1a51ae5e10ec4758a0f981be3ce5d6ac55445828463fce8dff3a355e0b75
WORKDIR /
COPY --from=builder /vault-kubernetes-kms .

ENTRYPOINT ["/vault-kubernetes-kms"]