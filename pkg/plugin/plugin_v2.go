package plugin

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v2"
)

type Plugin interface {
	Encrypt(ctx context.Context, data []byte) ([]byte, string, error)
	Decrypt(ctx context.Context, data []byte) ([]byte, error)
	GetKeyVersion(ctx context.Context) (string, error)
}

// KMSv2 is a KMS v2 wrapper.
type KMSv2 struct {
	pb.UnimplementedKeyManagementServiceServer

	plugin Plugin
}

// NewPluginV2 returns a KMS v2 wrapper.
func NewPluginV2(p Plugin) *KMSv2 {
	return &KMSv2{plugin: p}
}

// Status performs a simple health check and returns ok if encryption / decryption was successful
// https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#developing-a-kms-plugin-gRPC-server-notes-kms-v2
func (v2 *KMSv2) Status(ctx context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	health := "ok"

	kv, err := v2.plugin.GetKeyVersion(ctx)
	if err != nil {
		return nil, err
	}

	//nolint: contextcheck
	err = v2.Health(ctx)
	if err != nil {
		health = "err"

		zap.L().Info("v2 health check failed", zap.Error(err))
	}

	zap.L().Info("health status",
		zap.String("key_id", kv),
		zap.String("healthz", health),
		zap.String("version", "v2"),
	)

	return &pb.StatusResponse{
		Version: "v2",
		Healthz: health,
		KeyId:   kv,
	}, nil
}

// Health sends a simple plaintext for encryption and then compares the decrypted value.
func (v2 *KMSv2) Health(ctx context.Context) error {
	health := "health"

	start := time.Now().Unix()

	enc, err := v2.encrypt(ctx, []byte(health), strconv.FormatInt(start, 10), false)
	if err != nil {
		return err
	}

	dec, err := v2.decrypt(ctx, enc.GetCiphertext(), strconv.FormatInt(start, 10), false)
	if err != nil {
		return err
	}

	if health != string(dec.GetPlaintext()) {
		zap.L().Info("v2 health status failed")

		return errors.New("v2 health check failed")
	}

	return nil
}

func (v2 *KMSv2) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	return v2.encrypt(ctx, request.GetPlaintext(), request.GetUid(), true)
}

func (v2 *KMSv2) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	return v2.decrypt(ctx, request.GetCiphertext(), request.GetUid(), true)
}

func (v2 *KMSv2) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, v2)
}

func (v2 *KMSv2) encrypt(ctx context.Context, plain []byte, requestID string, recordMetrics bool) (*pb.EncryptResponse, error) {
	var timer *prometheus.Timer
	if recordMetrics {
		timer = prometheus.NewTimer(metrics.EncryptionOperationDurationSeconds)
		defer timer.ObserveDuration()
	}

	resp, id, err := v2.plugin.Encrypt(ctx, plain)
	if err != nil {
		if recordMetrics {
			metrics.EncryptionErrorsTotal.Inc()
		}

		return nil, err
	}

	if recordMetrics {
		zap.L().Info("v2 encryption request", zap.String("request_id", requestID))
	}

	return &pb.EncryptResponse{
		Ciphertext: resp,
		KeyId:      id,
	}, nil
}

func (v2 *KMSv2) decrypt(ctx context.Context, cipher []byte, requestID string, recordMetrics bool) (*pb.DecryptResponse, error) {
	var timer *prometheus.Timer
	if recordMetrics {
		timer = prometheus.NewTimer(metrics.DecryptionOperationDurationSeconds)
		defer timer.ObserveDuration()
	}

	resp, err := v2.plugin.Decrypt(ctx, cipher)
	if err != nil {
		if recordMetrics {
			metrics.DecryptionErrorsTotal.Inc()
		}

		return nil, err
	}

	if recordMetrics {
		zap.L().Info("v2 decryption request", zap.String("request_id", requestID))
	}

	return &pb.DecryptResponse{
		Plaintext: resp,
	}, nil
}
