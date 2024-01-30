package kms

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v2"
)

type Plugin struct {
	vc *vault.Client
}

func NewKMSPlugin(vc *vault.Client) *Plugin {
	p := &Plugin{
		vc: vc,
	}

	return p
}

// Health sends a simple plaintext for encryption and then compares the decrypted value.
func (p *Plugin) Health() error {
	health := "health"

	enc, err := p.Encrypt(context.Background(), &pb.EncryptRequest{Plaintext: []byte(health)})
	if err != nil {
		return err
	}

	dec, err := p.Decrypt(context.Background(), &pb.DecryptRequest{Ciphertext: enc.GetCiphertext()})
	if err != nil {
		return err
	}

	if health != string(dec.GetPlaintext()) {
		zap.L().Info("Health status failed")

		return fmt.Errorf("health check failed")
	}

	return nil
}

// Status performs a simple health check and returns ok if encryption / decryption was successful
// https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#developing-a-kms-plugin-gRPC-server-notes-kms-v2
func (p *Plugin) Status(_ context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	keyVersions, err := p.vc.GetKeyVersions()
	if err != nil {
		return nil, err
	}

	keys := []string{}
	for _, k := range keyVersions {
		//nolint: forcetypeassert
		keys = append(keys, k.(json.Number).String())
	}

	sort.Strings(keys)

	//nolint: contextcheck
	if err := p.Health(); err != nil {
		return nil, err
	}

	zap.L().Info("KMS Plugin Status",
		zap.String("key_id", keys[len(keys)-1]),
		zap.String("healthz", "ok"),
		zap.String("version", "v2"),
	)

	return &pb.StatusResponse{
		Version: "v2",
		Healthz: "ok",
		KeyId:   keys[len(keys)-1],
	}, nil
}

func (p *Plugin) Encrypt(_ context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	resp, id, err := p.vc.Encrypt(request.GetPlaintext())
	if err != nil {
		return nil, err
	}

	zap.L().Info("encryption request",
		zap.String("rquest_id", request.GetUid()),
		zap.String("plaintext", string(request.GetPlaintext())),
		zap.String("ciphertext", string(resp)),
		zap.String("key_id", id),
	)

	return &pb.EncryptResponse{
		Ciphertext: resp,
		KeyId:      id,
	}, nil
}

func (p *Plugin) Decrypt(_ context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	resp, err := p.vc.Decrypt(request.GetCiphertext())
	if err != nil {
		return nil, err
	}

	zap.L().Info("decryption request",
		zap.String("request_id", request.GetUid()),
		zap.String("ciphertext", string(request.GetCiphertext())),
		zap.String("plaintext", string(resp)),
	)

	return &pb.DecryptResponse{
		Plaintext: resp,
	}, nil
}

func (p *Plugin) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, p)
}
