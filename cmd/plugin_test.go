package cmd

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/stretchr/testify/require"
)

//nolint:cyclop,funlen,perfsprint
func TestNewPlugin(t *testing.T) {
	const (
		spiffeAudience = "vault-kms"
		spiffeSubject  = "spiffe://example.org/workload/vault-kubernetes-kms"
	)

	var fakeWorkloadAPI *testutils.FakeJWTWorkloadAPI

	testCases := []struct {
		name      string
		envVars   map[string]string
		args      []string
		vaultCmd  []string
		extraArgs func(t *testing.T, c *testutils.TestContainer) ([]string, error)
		assert    func(t *testing.T)
		err       bool
	}{
		{
			name: "token auth",
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/kms",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-auth-method=token",
				"-token=root",
				"-health-port=8081",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(_ *testing.T, c *testutils.TestContainer) ([]string, error) {
				return []string{fmt.Sprintf("-vault-address=%s", c.URI)}, nil
			},
		},
		{
			name: "approle auth",
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/kms",
				"auth enable approle",
				"write auth/approle/role/kms token_ttl=1h",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-auth-method=approle",
				"-health-port=8082",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(_ *testing.T, c *testutils.TestContainer) ([]string, error) {
				roleID, secretID, err := c.GetApproleCreds("approle", "kms")
				if err != nil {
					return nil, err
				}

				return []string{
					fmt.Sprintf("-vault-address=%s", c.URI),
					fmt.Sprintf("-approle-role-id=%s", roleID),
					fmt.Sprintf("-approle-secret-id=%s", secretID),
				}, nil
			},
		},
		{
			name: "mixed with env vars",
			envVars: map[string]string{
				"VAULT_KMS_TRANSIT_KEY":   "abc",
				"VAULT_KMS_TRANSIT_MOUNT": "transit",
				"VAULT_KMS_AUTH_METHOD":   "approle",
			},
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/abc",
				"auth enable -path=approle2 approle",
				"write auth/approle2/role/kms token_ttl=1h",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-approle-mount=approle2",
				"-health-port=8083",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(_ *testing.T, c *testutils.TestContainer) ([]string, error) {
				roleID, secretID, err := c.GetApproleCreds("approle2", "kms")
				if err != nil {
					return nil, err
				}

				return []string{
					fmt.Sprintf("-vault-address=%s", c.URI),
					fmt.Sprintf("-approle-role-id=%s", roleID),
					fmt.Sprintf("-approle-secret-id=%s", secretID),
				}, nil
			},
		},
		{
			name: "userpass auth",
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/kms",
				"auth enable userpass",
				"write auth/userpass/users/kms-user password=kms-pass",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-auth-method=userpass",
				"-health-port=8084",
				"-userpass-username=kms-user",
				"-userpass-password=kms-pass",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(_ *testing.T, c *testutils.TestContainer) ([]string, error) {
				return []string{
					fmt.Sprintf("-vault-address=%s", c.URI),
				}, nil
			},
		},
		{
			name: "spiffe jwt auth with env vars",
			envVars: map[string]string{
				"VAULT_KMS_AUTH_METHOD":         "jwt",
				"VAULT_KMS_JWT_ROLE":            "kms",
				"VAULT_KMS_JWT_TOKEN_SOURCE":    "spiffe",
				"VAULT_KMS_JWT_SPIFFE_AUDIENCE": spiffeAudience,
				"VAULT_KMS_JWT_SPIFFE_ID":       spiffeSubject,
			},
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/kms",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-health-port=8085",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(t *testing.T, c *testutils.TestContainer) ([]string, error) {
				t.Helper()

				privateKey, publicKeyPEM, err := testutils.GenerateJWTSigningKey()
				if err != nil {
					return nil, fmt.Errorf("generate jwt signing key: %w", err)
				}

				jwtToken, err := testutils.SignTestJWT(privateKey, spiffeSubject, spiffeAudience)
				if err != nil {
					return nil, fmt.Errorf("sign jwt-svid: %w", err)
				}

				fakeWorkloadAPI = testutils.NewFakeJWTWorkloadAPI(jwtToken)
				endpoint := testutils.StartFakeJWTWorkloadAPI(t, fakeWorkloadAPI)

				err = c.Container.CopyToContainer(t.Context(), []byte(publicKeyPEM), "/tmp/jwt-pub.pem", 0o444)
				if err != nil {
					return nil, fmt.Errorf("copy jwt public key: %w", err)
				}

				vaultCommands := []string{
					"vault auth enable jwt",
					"vault write auth/jwt/config jwt_validation_pubkeys=@/tmp/jwt-pub.pem",
					`printf 'path "transit/*" { capabilities = ["create","read","update"] }' | vault policy write transit-pol -`,
					fmt.Sprintf(
						"vault write auth/jwt/role/kms role_type=jwt bound_audiences=%s "+
							"user_claim=sub bound_subject=%s token_policies=transit-pol token_period=3600",
						spiffeAudience,
						spiffeSubject,
					),
				}

				for _, command := range vaultCommands {
					_, err = c.ExecShell(command)
					if err != nil {
						return nil, fmt.Errorf("configure vault jwt auth: %w", err)
					}
				}

				return []string{
					fmt.Sprintf("-vault-address=%s", c.URI),
					fmt.Sprintf("-jwt-spiffe-endpoint=%s", endpoint),
				}, nil
			},
			assert: func(t *testing.T) {
				t.Helper()

				requests := fakeWorkloadAPI.RecordedRequests()
				require.Len(t, requests, 1, "NewPlugin must fetch one JWT-SVID during Vault login")
				require.Equal(t, []string{spiffeAudience}, requests[0].Audience)
				require.Equal(t, spiffeSubject, requests[0].SPIFFEID)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vc, err := testutils.StartTestContainer(tc.vaultCmd...)
			require.NoError(t, err, "failed to start test container")

			//nolint: errcheck
			defer vc.Terminate()

			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			if tc.extraArgs != nil {
				extraArgs, err := tc.extraArgs(t, vc)
				require.NoError(t, err, "failed to perform extra args func: %w", err)

				tc.args = append(tc.args, extraArgs...)
			}

			os.Args = tc.args

			var wg sync.WaitGroup

			wg.Add(2)

			// invoke NewPlugin()
			go func() {
				defer wg.Done()

				err := NewPlugin("")
				if err != nil {
					log.Fatal(err)
				}
			}()

			// cancel after 5 seconds to avoid test timeout
			go func() {
				defer wg.Done()

				time.AfterFunc(5*time.Second, func() {
					_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				})
			}()

			wg.Wait()

			if tc.assert != nil {
				tc.assert(t)
			}
		})
	}
}

//nolint:funlen
func TestValidateFlags(t *testing.T) {
	testCases := []struct {
		name string
		opts *Options
		err  bool
	}{
		{
			name: "no vault address",
			err:  true,
			opts: &Options{
				Token: "abc",
			},
		},
		{
			name: "invalid auth method",
			err:  true,
			opts: &Options{
				AuthMethod: "invalid",
			},
		},
		{
			name: "token auth, but no token",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "token",
			},
		},
		{
			name: "approle auth, but no approle creds",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "approle",
			},
		},
		{
			name: "userpass auth, but no userpass creds",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "userpass",
			},
		},
		{
			name: "jwt auth, but no jwt role",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "jwt",
			},
		},
		{
			name: "jwt auth valid",
			err:  false,
			opts: &Options{ //nolint:gosec
				VaultAddress:         "e2e",
				AuthMethod:           "jwt",
				JWTRole:              "kms",
				JWTMount:             "jwt",
				JWTTokenSource:       jwtTokenSourceFile,
				JWTTokenPath:         "/var/run/secrets/kubernetes.io/serviceaccount/token",
				TransitKey:           "kms",
				TransitMount:         "transit",
				HealthPort:           "8080",
				TokenRefreshInterval: "60s",
			},
		},
		{
			name: "jwt auth with invalid token source",
			err:  true,
			opts: &Options{
				VaultAddress:   "e2e",
				AuthMethod:     "jwt",
				JWTRole:        "kms",
				JWTTokenSource: "invalid",
			},
		},
		{
			name: "jwt auth with file source but no token path",
			err:  true,
			opts: &Options{
				VaultAddress:   "e2e",
				AuthMethod:     "jwt",
				JWTRole:        "kms",
				JWTTokenSource: jwtTokenSourceFile,
			},
		},
		{
			name: "jwt auth with spiffe source is valid",
			err:  false,
			opts: &Options{
				VaultAddress:         "e2e",
				AuthMethod:           "jwt",
				JWTRole:              "kms",
				JWTMount:             "jwt",
				JWTTokenSource:       jwtTokenSourceSPIFFE,
				JWTSpiffeAudience:    "vault-kms",
				JWTSpiffeID:          "spiffe://example.org/workload/vault-kubernetes-kms",
				TransitKey:           "kms",
				TransitMount:         "transit",
				HealthPort:           "8080",
				TokenRefreshInterval: "60s",
			},
		},
		{
			name: "jwt auth with spiffe source but no audience",
			err:  true,
			opts: &Options{
				VaultAddress:   "e2e",
				AuthMethod:     "jwt",
				JWTRole:        "kms",
				JWTTokenSource: jwtTokenSourceSPIFFE,
				JWTSpiffeID:    "spiffe://example.org/workload/vault-kubernetes-kms",
			},
		},
		{
			name: "jwt auth with spiffe source but no spiffe id",
			err:  true,
			opts: &Options{
				VaultAddress:      "e2e",
				AuthMethod:        "jwt",
				JWTRole:           "kms",
				JWTTokenSource:    jwtTokenSourceSPIFFE,
				JWTSpiffeAudience: "vault-kms",
			},
		},
		{
			name: "jwt auth with malformed spiffe id",
			err:  true,
			opts: &Options{
				VaultAddress:      "e2e",
				AuthMethod:        "jwt",
				JWTRole:           "kms",
				JWTTokenSource:    jwtTokenSourceSPIFFE,
				JWTSpiffeAudience: "vault-kms",
				JWTSpiffeID:       "not-a-spiffe-id",
			},
		},

		{
			name: "all plugin versions disabled",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "token",
				Token:        "token",
				DisableV1:    true,
				DisableV2:    true,
			},
		},
		{
			name: "cert auth missing role",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "cert",
				CertFile:     "/tmp/cert.pem",
				CertKey:      "/tmp/key.pem",
			},
		},
		{
			name: "cert auth missing cert files",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "cert",
				CertAuthRole: "kms",
			},
		},
		{
			name: "cert auth with cert-pem is valid",
			err:  false,
			opts: &Options{
				VaultAddress:         "e2e",
				AuthMethod:           "cert",
				CertAuthMount:        "cert",
				CertAuthRole:         "kms",
				CertPEM:              "/tmp/combined.pem",
				TransitKey:           "kms",
				TransitMount:         "transit",
				HealthPort:           "8080",
				TokenRefreshInterval: "60s",
			},
		},
		{
			name: "cert auth with separate cert+key is valid",
			err:  false,
			opts: &Options{
				VaultAddress:         "e2e",
				AuthMethod:           "cert",
				CertAuthMount:        "cert",
				CertAuthRole:         "kms",
				CertFile:             "/tmp/cert.pem",
				CertKey:              "/tmp/key.pem",
				TransitKey:           "kms",
				TransitMount:         "transit",
				HealthPort:           "8080",
				TokenRefreshInterval: "60s",
			},
		},
	}

	for _, tc := range testCases {
		err := tc.opts.validateFlags()
		if tc.err {
			require.Error(t, err, tc.name)

			continue
		}

		require.NoError(t, err, tc.name)
	}
}
