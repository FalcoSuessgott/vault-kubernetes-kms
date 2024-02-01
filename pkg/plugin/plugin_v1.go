package plugin

import (
	"context"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v1beta1"
)

// PluginV1 a kms plugin wrapper.
type PluginV1 struct {
	vc *vault.Client
}

// NewPluginV1 returns a kms wrapper.
func NewPluginV1(vc *vault.Client) *PluginV1 {
	p := &PluginV1{
		vc: vc,
	}

	return p
}

// nolint: staticcheck
func (p *PluginV1) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        "v1beta1",
		RuntimeName:    "vault",
		RuntimeVersion: "0.0.1",
	}, nil
}

// nolint: staticcheck
func (p *PluginV1) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	resp, _, err := p.vc.Encrypt(ctx, request.GetPlain())
	if err != nil {
		return nil, err
	}

	zap.L().Info("encryption request")

	return &pb.EncryptResponse{
		Cipher: resp,
	}, nil
}

// nolint: staticcheck
func (p *PluginV1) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	resp, err := p.vc.Decrypt(ctx, request.GetCipher())
	if err != nil {
		return nil, err
	}

	zap.L().Info("decryption request")

	return &pb.DecryptResponse{
		Plain: resp,
	}, nil
}

// nolint: staticcheck
func (p *PluginV1) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, p)
}
