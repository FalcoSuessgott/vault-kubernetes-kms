package plugin

import (
	"context"
	"errors"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v1beta1"
)

// KMSv1 a kms plugin wrapper.
type KMSv1 struct {
	plugin Plugin
}

// NewPluginV1 returns a kms wrapper.
func NewPluginV1(p Plugin) *KMSv1 {
	return &KMSv1{p}
}

// nolint: staticcheck
func (v1 *KMSv1) Version(ctx context.Context, request *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        "v1beta1",
		RuntimeName:    "vault",
		RuntimeVersion: "0.0.1",
	}, nil
}

// Health sends a simple plaintext for encryption and then compares the decrypted value.
// nolint: staticcheck
func (v1 *KMSv1) Health() error {
	health := "health"

	enc, err := v1.Encrypt(context.Background(), &pb.EncryptRequest{
		Plain: []byte(health),
	})
	if err != nil {
		return err
	}

	dec, err := v1.Decrypt(context.Background(), &pb.DecryptRequest{
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
func (v1 *KMSv1) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	timer := prometheus.NewTimer(metrics.EncryptionOperationDurationSeconds)

	resp, _, err := v1.plugin.TransitEncrypt(ctx, request.GetPlain())
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
func (v1 *KMSv1) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	timer := prometheus.NewTimer(metrics.DecryptionOperationDurationSeconds)

	resp, err := v1.plugin.TransitDecrypt(ctx, request.GetCipher())
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
func (v1 *KMSv1) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, v1)
}
