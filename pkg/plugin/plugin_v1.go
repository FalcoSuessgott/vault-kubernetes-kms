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

// KMSv1 is a KMS v1 wrapper.
type KMSv1 struct {
	pb.UnimplementedKeyManagementServiceServer

	plugin Plugin
}

// NewPluginV1 returns a kms wrapper.
func NewPluginV1(p Plugin) *KMSv1 {
	return &KMSv1{plugin: p}
}

// Version returns static plugin version metadata for the KMS v1 API.
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
func (v1 *KMSv1) Health(ctx context.Context) error {
	health := "health"

	enc, err := v1.encrypt(ctx, []byte(health), false)
	if err != nil {
		return err
	}

	dec, err := v1.decrypt(ctx, enc.GetCipher(), false)
	if err != nil {
		return err
	}

	if health != string(dec.GetPlain()) {
		zap.L().Info("v1 health status failed")

		return errors.New("v1 health check failed")
	}

	return nil
}

// Encrypt encrypts plaintext using Vault transit for the KMS v1 API.
// nolint: staticcheck
func (v1 *KMSv1) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	return v1.encrypt(ctx, request.GetPlain(), true)
}

// Decrypt decrypts ciphertext using Vault transit for the KMS v1 API.
// nolint: staticcheck
func (v1 *KMSv1) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	return v1.decrypt(ctx, request.GetCipher(), true)
}

// Register registers the KMS v1 gRPC service with the server.
// nolint: staticcheck
func (v1 *KMSv1) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, v1)
}

// nolint: staticcheck
func (v1 *KMSv1) encrypt(ctx context.Context, plain []byte, recordMetrics bool) (*pb.EncryptResponse, error) {
	var timer *prometheus.Timer
	if recordMetrics {
		timer = prometheus.NewTimer(metrics.EncryptionOperationDurationSeconds)
		defer timer.ObserveDuration()
	}

	resp, _, err := v1.plugin.Encrypt(ctx, plain)
	if err != nil {
		if recordMetrics {
			metrics.EncryptionErrorsTotal.Inc()
		}

		return nil, err
	}

	if recordMetrics {
		zap.L().Info("v1 encryption request")
	}

	return &pb.EncryptResponse{
		Cipher: resp,
	}, nil
}

// nolint: staticcheck
func (v1 *KMSv1) decrypt(ctx context.Context, cipher []byte, recordMetrics bool) (*pb.DecryptResponse, error) {
	var timer *prometheus.Timer
	if recordMetrics {
		timer = prometheus.NewTimer(metrics.DecryptionOperationDurationSeconds)
		defer timer.ObserveDuration()
	}

	resp, err := v1.plugin.Decrypt(ctx, cipher)
	if err != nil {
		if recordMetrics {
			metrics.DecryptionErrorsTotal.Inc()
		}

		return nil, err
	}

	if recordMetrics {
		zap.L().Info("v1 decryption request")
	}

	return &pb.DecryptResponse{
		Plain: resp,
	}, nil
}
