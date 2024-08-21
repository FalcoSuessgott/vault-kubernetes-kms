package plugin

import (
	"context"
	"errors"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v1beta1"
)

// PluginV1 a kms plugin wrapper.
type PluginV1 struct {
	*vault.Client
}

// NewPluginV1 returns a kms wrapper.
func NewPluginV1(vc *vault.Client) *PluginV1 {
	return &PluginV1{vc}
}

// nolint: staticcheck
func (p *PluginV1) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        "v1beta1",
		RuntimeName:    "vault",
		RuntimeVersion: "0.0.1",
	}, nil
}

// Health sends a simple plaintext for encryption and then compares the decrypted value.
// nolint: staticcheck
func (p *PluginV1) Health() error {
	health := "health"

	enc, err := p.Encrypt(context.Background(), &pb.EncryptRequest{
		Plain: []byte(health),
	})
	if err != nil {
		return err
	}

	dec, err := p.Decrypt(context.Background(), &pb.DecryptRequest{
		Cipher: enc.GetCipher(),
	})
	if err != nil {
		return err
	}

	if health != string(dec.GetPlain()) {
		zap.L().Info("v1 health status failed")

		return errors.New("v1 health check failed")
	}

	return nil
}

// nolint: staticcheck
func (p *PluginV1) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	timer := prometheus.NewTimer(metrics.EncryptionOperationDurationSeconds)

	resp, _, err := p.Client.Encrypt(ctx, request.GetPlain())
	if err != nil {
		metrics.EncryptionErrorsTotal.Inc()

		return nil, err
	}

	zap.L().Info("v1 encryption request")

	timer.ObserveDuration()

	return &pb.EncryptResponse{
		Cipher: resp,
	}, nil
}

// nolint: staticcheck
func (p *PluginV1) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	timer := prometheus.NewTimer(metrics.DecryptionOperationDurationSeconds)

	resp, err := p.Client.Decrypt(ctx, request.GetCipher())
	if err != nil {
		metrics.DecryptionErrorsTotal.Inc()

		return nil, err
	}

	zap.L().Info("v1 decryption request")

	timer.ObserveDuration()

	return &pb.DecryptResponse{
		Plain: resp,
	}, nil
}

// nolint: staticcheck
func (p *PluginV1) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, p)
}
