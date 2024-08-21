package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "k8s.io/kms/apis/v2"
)

var socket = "/tmp/kms.socket"

//nolint:funlen,gocritic,cyclop
func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: cmd/v2_Client/main.go <string>")

		os.Exit(0)
	}

	conn, err := grpc.NewClient("unix:"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to initialize grpc client: %v", err)
	}

	defer conn.Close()

	client := pb.NewKeyManagementServiceClient(conn)
	ctx := context.Background()

	_, err = client.Status(ctx, &pb.StatusRequest{})
	if err != nil {
		log.Fatalf("Failed to get version: %v", err)
	}

	if len(os.Args) > 1 {
		input := strings.Join(os.Args[1:], " ")

		encRes, err := client.Encrypt(ctx, &pb.EncryptRequest{Plaintext: []byte(input)})
		if err != nil {
			log.Fatalf("Failed to encrypt: %v", err)
		}

		resp := base64.StdEncoding.EncodeToString(encRes.GetCiphertext())
		fmt.Printf("\"%s\" -> \"%s\"", input, resp)

		b, err := base64.StdEncoding.DecodeString(resp)
		if err != nil {
			log.Fatalf("Failed to decode: %v", err)
		}

		decResp, err := client.Decrypt(ctx, &pb.DecryptRequest{Ciphertext: b})
		if err != nil {
			log.Fatalf("Failed to encrypt: %v", err)
		}

		fmt.Printf(" -> \"%s\"\n", string(decResp.GetPlaintext()))

		os.Exit(0)
	}
}
