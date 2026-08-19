package testutils

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spiffe/go-spiffe/v2/proto/spiffe/workload"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// JWTSVIDRequest records the identity and audiences requested from the fake
// SPIFFE Workload API.
type JWTSVIDRequest struct {
	Audience []string
	SPIFFEID string
}

// FakeJWTWorkloadAPI serves a configurable JWT-SVID and records each request.
type FakeJWTWorkloadAPI struct {
	workload.UnimplementedSpiffeWorkloadAPIServer

	mu       sync.Mutex
	token    string
	requests []JWTSVIDRequest
}

// NewFakeJWTWorkloadAPI returns a fake Workload API serving token.
func NewFakeJWTWorkloadAPI(token string) *FakeJWTWorkloadAPI {
	return &FakeJWTWorkloadAPI{token: token}
}

// FetchJWTSVID implements the SPIFFE Workload API JWT-SVID RPC.
func (f *FakeJWTWorkloadAPI) FetchJWTSVID(
	_ context.Context,
	req *workload.JWTSVIDRequest,
) (*workload.JWTSVIDResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, JWTSVIDRequest{
		Audience: append([]string(nil), req.GetAudience()...),
		SPIFFEID: req.GetSpiffeId(),
	})

	return &workload.JWTSVIDResponse{
		Svids: []*workload.JWTSVID{{
			SpiffeId: req.GetSpiffeId(),
			Svid:     f.token,
		}},
	}, nil
}

// SetToken changes the JWT-SVID returned by subsequent requests.
func (f *FakeJWTWorkloadAPI) SetToken(token string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.token = token
}

// RecordedRequests returns a snapshot of requests received by the fake API.
func (f *FakeJWTWorkloadAPI) RecordedRequests() []JWTSVIDRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	requests := make([]JWTSVIDRequest, len(f.requests))
	copy(requests, f.requests)

	return requests
}

// StartFakeJWTWorkloadAPI serves api on a temporary Unix domain socket.
func StartFakeJWTWorkloadAPI(t *testing.T, api *FakeJWTWorkloadAPI) string {
	t.Helper()

	// t.TempDir includes the test name and can exceed the macOS Unix-socket path limit.
	socketDir, err := os.MkdirTemp("", "spiffe-") //nolint:usetesting
	require.NoError(t, err, "create fake workload API socket directory")
	t.Cleanup(func() { _ = os.RemoveAll(socketDir) })

	socketPath := filepath.Join(socketDir, "workload-api.sock")

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "unix", socketPath)
	require.NoError(t, err, "listen on fake workload API socket")

	server := grpc.NewServer()
	workload.RegisterSpiffeWorkloadAPIServer(server, api)

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(server.Stop)

	return "unix://" + socketPath
}
