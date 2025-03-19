FROM golang:1.23 AS builder
WORKDIR /usr/src/vault-kubernetes-kms
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 go build -v -o /vault-kubernetes-kms main.go

# https://github.com/GoogleContainerTools/distroless/issues/1360#issuecomment-1646667145
FROM gcr.io/distroless/static-debian12@sha256:95ea148e8e9edd11cc7f639dc11825f38af86a14e5c7361753c741ceadef2167
WORKDIR /
COPY --from=builder /vault-kubernetes-kms .

ENTRYPOINT ["/vault-kubernetes-kms"]
