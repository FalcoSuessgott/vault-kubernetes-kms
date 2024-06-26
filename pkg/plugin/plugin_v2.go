package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	pb "k8s.io/kms/apis/v2"
)

// PluginV2 a kms plugin wrapper.
type PluginV2 struct {
	vc *vault.Client
}

// PluginV2 returns a kms wrapper.
func NewPluginV2(vc *vault.Client) *PluginV2 {
	p := &PluginV2{
		vc: vc,
	}

	return p
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
		zap.L().Info("Health status failed")

		return errors.New("health check failed")
	}

	return nil
}

// Status performs a simple health check and returns ok if encryption / decryption was successful
// https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#developing-a-kms-plugin-gRPC-server-notes-kms-v2
func (p *PluginV2) Status(_ context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	health := "ok"

	if err := p.vc.TokenRefresh(); err != nil {
		health = "err"

		zap.L().Info(err.Error())
	}

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
		health = "err"

		zap.L().Info(err.Error())
	}

	zap.L().Info("health status",
		zap.String("key_id", keys[len(keys)-1]),
		zap.String("healthz", health),
		zap.String("version", "v2"),
	)

	return &pb.StatusResponse{
		Version: "v2",
		Healthz: "ok",
		KeyId:   keys[len(keys)-1],
	}, nil
}

func (p *PluginV2) Encrypt(ctx context.Context, request *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	resp, id, err := p.vc.Encrypt(ctx, request.GetPlaintext())
	if err != nil {
		return nil, err
	}

	zap.L().Info("v2 encryption request", zap.String("request_id", request.GetUid()))

	return &pb.EncryptResponse{
		Ciphertext: resp,
		KeyId:      id,
	}, nil
}

func (p *PluginV2) Decrypt(ctx context.Context, request *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	resp, err := p.vc.Decrypt(ctx, request.GetCiphertext())
	if err != nil {
		return nil, err
	}

	zap.L().Info("v2 decryption request", zap.String("request_id", request.GetUid()))

	return &pb.DecryptResponse{
		Plaintext: resp,
	}, nil
}

func (p *PluginV2) Register(s *grpc.Server) {
	pb.RegisterKeyManagementServiceServer(s, p)
}
