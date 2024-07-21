package plugin

import (
	"context"
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/socket"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	v1beta1 "k8s.io/kms/apis/v1beta1"
	v2 "k8s.io/kms/apis/v2"
)

var socketPath = "/opt/vault/vaultkms.sock"

type PluginSuite struct {
	suite.Suite

	tc    *testutils.TestContainer
	vault *vault.Client
}

func TestVaultSuite(t *testing.T) {
	// github actions doesn't offer the docker sock, which we require for testing
	if runtime.GOOS == "linux" {
		suite.Run(t, new(PluginSuite))
	}
}

func (p *PluginSuite) SetupAllSuite() {
	// create unix socket
	_, err := socket.NewSocket(socketPath)
	if err != nil {
		p.T().Fatal("cannot create socket: %w", err)
	}
}

func (p *PluginSuite) SetupSubTest() {
	tc, err := testutils.StartTestContainer(
		"secrets enable transit",
		"write -f transit/keys/kms",
	)
	if err != nil {
		log.Fatal(err)
	}

	p.tc = tc

	vault, err := vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithTokenAuth(tc.Token),
		vault.WithTransit("transit", "kms"),
	)
	if err != nil {
		log.Fatal(err)
	}

	p.vault = vault
}

func (p *PluginSuite) TearDownSubTest() {
	if err := p.tc.Terminate(); err != nil {
		log.Fatal(err)
	}
}

// nolint: funlen, dupl
func (p *PluginSuite) TestPluginEncryptDecrypt() {
	testCases := []struct {
		name string
		data []byte
		v1   bool
		err  bool
	}{
		{
			name: "simple v2 encrypt decrypt",
			data: []byte("simple string"),
			v1:   false,
		},
		{
			name: "simple v1 encrypt decrypt",
			data: []byte("simple string"),
			v1:   true,
		},
	}

	for _, tc := range testCases {
		grpc := grpc.NewServer()

		p.Run(tc.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			p.T().Cleanup(cancel)

			// v1
			if tc.v1 {
				pluginV1 := NewPluginV1(p.vault)

				pluginV1.Register(grpc)

				//nolint: staticcheck
				vResp, err := pluginV1.Version(ctx, &v1beta1.VersionRequest{})
				p.Require().NoError(err, tc.name)

				//nolint: staticcheck
				p.Require().Equal(&v1beta1.VersionResponse{
					Version:        "v1beta1",
					RuntimeName:    "vault",
					RuntimeVersion: "0.0.1",
				}, vResp, tc.name)

				// encrypt
				//nolint: staticcheck
				encryptRequest := &v1beta1.EncryptRequest{
					Plain: tc.data,
				}

				resp, err := pluginV1.Encrypt(ctx, encryptRequest)
				p.Require().NoError(err, tc.name)

				// decrypt
				//nolint: staticcheck
				decryptRequest := &v1beta1.DecryptRequest{
					Cipher: resp.GetCipher(),
				}

				res, err := pluginV1.Decrypt(ctx, decryptRequest)
				p.Require().NoError(err, tc.name)

				// compare result
				p.Require().Equal(tc.data, res.GetPlain(), tc.name)
			} else {
				pluginV2 := NewPluginV2(p.vault)

				pluginV2.Register(grpc)

				vResp, err := pluginV2.Status(ctx, &v2.StatusRequest{})
				p.Require().NoError(err, tc.name)

				//nolint: staticcheck
				p.Require().Equal(&v2.StatusResponse{
					Version: "v2",
					Healthz: "ok",
					KeyId:   "1",
				}, vResp, tc.name)

				// encrypt
				encryptRequest := &v2.EncryptRequest{
					Plaintext: tc.data,
				}

				resp, err := pluginV2.Encrypt(ctx, encryptRequest)
				p.Require().NoError(err, tc.name)

				// decrypt
				decryptRequest := &v2.DecryptRequest{
					Ciphertext: resp.GetCiphertext(),
				}

				res, err := pluginV2.Decrypt(ctx, decryptRequest)
				p.Require().NoError(err, tc.name)

				// compare result
				p.Require().Equal(tc.data, res.GetPlaintext(), tc.name)
			}
		})
	}
}
