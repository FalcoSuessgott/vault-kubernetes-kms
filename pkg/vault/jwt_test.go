package vault

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/spiffe/go-spiffe/v2/proto/spiffe/workload"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type jwtSVIDRequest struct {
	audience []string
	spiffeID string
}

type fakeJWTWorkloadAPI struct {
	workload.UnimplementedSpiffeWorkloadAPIServer

	mu       sync.Mutex
	token    string
	requests []jwtSVIDRequest
}

func (f *fakeJWTWorkloadAPI) FetchJWTSVID(
	_ context.Context,
	req *workload.JWTSVIDRequest,
) (*workload.JWTSVIDResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, jwtSVIDRequest{
		audience: append([]string(nil), req.GetAudience()...),
		spiffeID: req.GetSpiffeId(),
	})

	return &workload.JWTSVIDResponse{
		Svids: []*workload.JWTSVID{{
			SpiffeId: req.GetSpiffeId(),
			Svid:     f.token,
		}},
	}, nil
}

func (f *fakeJWTWorkloadAPI) setToken(token string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.token = token
}

func (f *fakeJWTWorkloadAPI) recordedRequests() []jwtSVIDRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	requests := make([]jwtSVIDRequest, len(f.requests))
	copy(requests, f.requests)

	return requests
}

func startFakeJWTWorkloadAPI(t *testing.T, api *fakeJWTWorkloadAPI) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "workload-api.sock")

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "unix", socketPath)
	require.NoError(t, err, "listen on fake workload API socket")

	server := grpc.NewServer()
	workload.RegisterSpiffeWorkloadAPIServer(server, api)

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
	})

	return "unix://" + socketPath
}

func TestFileJWTTokenSourceRereadsToken(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "jwt-token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("first-token\n"), 0o600))

	source := fileJWTTokenSource{path: tokenPath}
	token, err := source.Token(t.Context())
	require.NoError(t, err)
	require.Equal(t, "first-token", token)

	require.NoError(t, os.WriteFile(tokenPath, []byte("second-token\n"), 0o600))
	token, err = source.Token(t.Context())
	require.NoError(t, err)
	require.Equal(t, "second-token", token)
}

func TestSPIFFEJWTTokenSourceBoundsFetch(t *testing.T) {
	subject, err := spiffeid.FromString("spiffe://example.org/workload/vault-kubernetes-kms")
	require.NoError(t, err)

	fetchErr := errors.New("workload API unavailable")
	source := spiffeJWTTokenSource{
		audience: "vault-kms",
		subject:  subject,
		fetch: func(ctx context.Context, params jwtsvid.Params, _ ...workloadapi.ClientOption) (*jwtsvid.SVID, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "SPIFFE fetch must have a deadline")
			require.LessOrEqual(t, time.Until(deadline), spiffeJWTTokenFetchTimeout)
			require.Equal(t, "vault-kms", params.Audience)
			require.Equal(t, subject, params.Subject)

			return nil, fetchErr
		},
	}

	_, err = source.Token(t.Context())
	require.ErrorIs(t, err, fetchErr)
}

// TestSPIFFEJWTAuth exercises the Workload API request, Vault JWT login,
// Transit operations, and JWT-SVID rotation against a real Vault server.
//
//nolint:funlen
func TestSPIFFEJWTAuth(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("unix socket and docker sock unavailable on windows CI")
	}

	const (
		audience = "vault-kms"
		subject  = "spiffe://example.org/workload/vault-kubernetes-kms"
	)

	privateKey, publicKeyPEM, err := testutils.GenerateJWTSigningKey()
	require.NoError(t, err, "generate jwt signing key")

	jwtToken, err := testutils.SignTestJWT(privateKey, subject, audience)
	require.NoError(t, err, "sign initial jwt-svid")

	fakeWorkloadAPI := &fakeJWTWorkloadAPI{token: jwtToken}
	endpoint := startFakeJWTWorkloadAPI(t, fakeWorkloadAPI)
	t.Setenv(workloadapi.SocketEnv, endpoint)

	tc, err := testutils.StartTestContainer(
		"secrets enable transit",
		"write -f transit/keys/kms",
	)
	require.NoError(t, err, "start vault container")
	t.Cleanup(func() { _ = tc.Terminate() })

	err = tc.Container.CopyToContainer(t.Context(), []byte(publicKeyPEM), "/tmp/jwt-pub.pem", 0o444)
	require.NoError(t, err, "copy public key to container")

	_, err = tc.ExecShell("vault auth enable jwt")
	require.NoError(t, err, "enable jwt auth")

	_, err = tc.ExecShell("vault write auth/jwt/config jwt_validation_pubkeys=@/tmp/jwt-pub.pem")
	require.NoError(t, err, "configure jwt auth")

	_, err = tc.ExecShell(
		`printf 'path "transit/*" { capabilities = ["create","read","update"] }' | vault policy write transit-pol -`,
	)
	require.NoError(t, err, "write transit policy")

	_, err = tc.ExecShell(fmt.Sprintf(
		"vault write auth/jwt/role/kms role_type=jwt bound_audiences=%s "+
			"user_claim=sub bound_subject=%s token_policies=transit-pol token_period=3600",
		audience,
		subject,
	))
	require.NoError(t, err, "write jwt role")

	vc, err := NewClient(
		WithVaultAddress(tc.URI),
		WithTransit("transit", "kms"),
		WithSPIFFEJWTAuth("jwt", "kms", "", audience, subject),
	)
	require.NoError(t, err, "create vault client with spiffe jwt auth")

	requests := fakeWorkloadAPI.recordedRequests()
	require.Len(t, requests, 1)
	require.Equal(t, []string{audience}, requests[0].audience)
	require.Equal(t, subject, requests[0].spiffeID)

	plaintext := []byte("hello-spiffe-jwt-auth")
	ciphertext, _, err := vc.Encrypt(t.Context(), plaintext)
	require.NoError(t, err, "encrypt")

	decrypted, err := vc.Decrypt(t.Context(), ciphertext)
	require.NoError(t, err, "decrypt")
	require.Equal(t, plaintext, decrypted)

	firstVaultToken := vc.Client.Token()
	rotatedJWT, err := testutils.SignTestJWT(privateKey, subject, audience)
	require.NoError(t, err, "sign rotated jwt-svid")
	fakeWorkloadAPI.setToken(rotatedJWT)

	_, err = tc.ExecShell("vault token revoke " + firstVaultToken)
	require.NoError(t, err, "revoke initial vault token")
	require.NoError(t, vc.AuthMethodFunc(vc), "re-authenticate with rotated jwt-svid")
	require.NotEqual(t, firstVaultToken, vc.Client.Token(), "reauthentication must install a fresh vault token")

	requests = fakeWorkloadAPI.recordedRequests()
	require.Len(t, requests, 2, "each vault login must fetch a fresh jwt-svid")

	ciphertext, _, err = vc.Encrypt(t.Context(), plaintext)
	require.NoError(t, err, "encrypt after reauthentication")
	decrypted, err = vc.Decrypt(t.Context(), ciphertext)
	require.NoError(t, err, "decrypt after reauthentication")
	require.Equal(t, plaintext, decrypted)
}
