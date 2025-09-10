FROM golang:1.25 AS builder
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -o /vault-kubernetes-kms main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:87bce11be0af225e4ca761c40babb06d6d559f5767fbf7dc3c47f0f1a466b92c
WORKDIR /
COPY --from=builder /vault-kubernetes-kms .

ENTRYPOINT ["/vault-kubernetes-kms"]
