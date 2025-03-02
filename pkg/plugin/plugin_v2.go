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
	TransitEncrypt(ctx context.Context, data []byte) ([]byte, string, error)
	TransitDecrypt(ctx context.Context, data []byte) ([]byte, error)
	TransitKeyVersion(ctx context.Context) (string, error)
}

// KMSv2 a kms plugin wrapper.
type KMSv2 struct {
	plugin Plugin
}

// PluginV2 returns a kms wrapper.
func NewPluginV2(p Plugin) *KMSv2 {
	return &KMSv2{p}
}

// Status performs a simple health check and returns ok if encryption / decryption was successful
// https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#developing-a-kms-plugin-gRPC-server-notes-kms-v2
func (v2 *KMSv2) Status(ctx context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	health := "ok"

	kv, err := v2.plugin.TransitKeyVersion(ctx)
	if err != nil {
		return nil, err
	}

	//nolint: contextcheck
	if err := v2.Health(); err != nil {
		health = "err"

		zap.L().Info(err.Error())
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
func (v2 *KMSv2) Health() error {
	health := "health"
	ctx := context.Background()

	start := time.Now().Unix()

	enc, err := v2.Encrypt(ctx, &pb.EncryptRequest{
		Plaintext: []byte(health),
		Uid:       strconv.FormatInt(start, 10),
	})
	if err != nil {
		return err
	}

	dec, err := v2.Decrypt(ctx, &pb.DecryptRequest{
		Ciphertext: enc.GetCiphertext(),
		Uid:        strconv.FormatInt(start, 10),
	})
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
	timer := prometheus.NewTimer(metrics.EncryptionOperationDurationSeconds)

	resp, id, err := v2.plugin.TransitEncrypt(ctx, request.GetPlaintext())
	if err != nil {
		metrics.EncryptionErrorsTotal.Inc()

		return nil, err
	}

	zap.L().Info("v2 encryption request", zap.String("request_id", request.GetUid()))

	timer.ObserveDuration()

	return &pb.EncryptResponse{
		Ciphertext: resp,
		KeyId:      id,
	}, nil
}

func (v2 *KMSv2) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	timer := prometheus.NewTimer(metrics.DecryptionOperationDurationSeconds)

	resp, err := v2.plugin.TransitDecrypt(ctx, request.GetCiphertext())
	if err != nil {
		metrics.DecryptionErrorsTotal.Inc()

		return nil, err
	}

	zap.L().Info("v2 decryption request", zap.String("request_id", request.GetUid()))

	timer.ObserveDuration()

	return &pb.DecryptResponse{
		Plaintext: resp,
	}, nil
}

func (v2 *KMSv2) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, v2)
}
