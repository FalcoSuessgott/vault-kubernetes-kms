package plugin

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v2"
)

// PluginV2 a kms plugin wrapper.
type PluginV2 struct {
	*vault.Client
}

// PluginV2 returns a kms wrapper.
func NewPluginV2(vc *vault.Client) *PluginV2 {
	return &PluginV2{vc}
}

// Status performs a simple health check and returns ok if encryption / decryption was successful
// https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#developing-a-kms-plugin-gRPC-server-notes-kms-v2
func (p *PluginV2) Status(ctx context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	health := "ok"

	kv, err := p.Client.GetKeyVersion(ctx)
	if err != nil {
		return nil, err
	}

	//nolint: contextcheck
	if err := p.Health(); err != nil {
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
		Healthz: "ok",
		KeyId:   kv,
	}, nil
}

// Health sends a simple plaintext for encryption and then compares the decrypted value.
func (p *PluginV2) Health() error {
	health := "health"

	start := time.Now().Unix()

	enc, err := p.Encrypt(context.Background(), &pb.EncryptRequest{
		Plaintext: []byte(health),
		Uid:       strconv.FormatInt(start, 10),
	})
	if err != nil {
		return err
	}

	dec, err := p.Decrypt(context.Background(), &pb.DecryptRequest{
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

func (p *PluginV2) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	timer := prometheus.NewTimer(metrics.EncryptionOperationDurationSeconds)

	resp, id, err := p.Client.Encrypt(ctx, request.GetPlaintext())
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

func (p *PluginV2) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	timer := prometheus.NewTimer(metrics.DecryptionOperationDurationSeconds)

	resp, err := p.Client.Decrypt(ctx, request.GetCiphertext())
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

func (p *PluginV2) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, p)
}
