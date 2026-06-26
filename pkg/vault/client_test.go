package vault

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const windowsOS = "windows"

type VaultSuite struct {
	suite.Suite

	tc    *testutils.TestContainer
	vault *Client
}

func (s *VaultSuite) TearDownSubTest() {
	err := s.tc.Terminate()
	if err != nil {
		log.Fatal(err)
	}
}

func (s *VaultSuite) SetupSubTest() {
	tc, err := testutils.StartTestContainer(
		"secrets enable transit",
		"write -f transit/keys/kms",
	)
	if err != nil {
		log.Fatal(err)
	}

	s.tc = tc

	vault, err := NewClient(
		WithVaultAddress(tc.URI),
		WithTokenAuth(tc.Token),
		WithTransit("transit", "kms"),
	)
	if err != nil {
		log.Fatal(err)
	}

	s.vault = vault
}

// nolint: funlen
func (s *VaultSuite) TestAuthMethods() {
	testCases := []struct {
		name    string
		prepCmd []string
		auth    func() (Option, error)
		err     bool
	}{
		{
			name: "basic approle auth",
			prepCmd: []string{
				"vault auth enable approle",
				"vault write auth/approle/role/kms token_ttl=1h",
			},
			auth: func() (Option, error) {
				roleID, secretID, err := s.tc.GetApproleCreds("approle", "kms")
				if err != nil {
					return nil, err
				}

				return WithAppRoleAuth("approle", roleID, secretID), nil
			},
		},
		{
			name: "invalid approle auth",
			err:  true,
			auth: func() (Option, error) {
				return WithAppRoleAuth("approle", "invalid", "invalid"), nil
			},
		},
		{
			name: "userpass auth",
			prepCmd: []string{
				"vault auth enable userpass",
				"vault write auth/userpass/users/kms-user password=kms-pass",
			},
			auth: func() (Option, error) {
				return WithUserPassAuth("userpass", "kms-user", "kms-pass"), nil
			},
		},
		{
			name: "invalid userpass auth",
			prepCmd: []string{
				"vault auth enable userpass",
				"vault write auth/userpass/users/kms-user password=kms-pass",
			},
			err: true,
			auth: func() (Option, error) {
				return WithUserPassAuth("userpass", "invalid", "invalid"), nil
			},
		},
		{
			name: "token auth",
			auth: func() (Option, error) {
				token, err := s.tc.GetToken("default", "1h")
				if err != nil {
					return nil, err
				}

				return WithTokenAuth(token), nil
			},
		},
		{
			name: "invalid token auth",
			auth: func() (Option, error) {
				return WithTokenAuth("invalidtoken"), nil
			},
			err: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// prep vault
			for _, cmd := range tc.prepCmd {
				_, _, err := s.tc.Container.Exec(s.T().Context(), strings.Split(cmd, " "))
				s.Require().NoError(err, tc.name)
			}

			// perform auth
			auth, err := tc.auth()
			s.Require().NoError(err, "auth "+tc.name)

			_, err = NewClient(
				WithVaultAddress(s.tc.URI),
				WithTokenAuth(s.tc.Token),
				auth,
			)

			// assert
			s.Require().Equal(tc.err, err != nil, tc.name)
		})
	}
}

func TestVaultSuite(t *testing.T) {
	// github actions doesn't offer the docker sock, which we require for testing
	if runtime.GOOS != windowsOS {
		suite.Run(t, new(VaultSuite))
	}
}

func TestWithJWTAuthErrors(t *testing.T) {
	t.Run("missing token file", func(t *testing.T) {
		opt := WithJWTAuth("jwt", "kms", "/nonexistent/token/path")
		c := &Client{Client: &api.Client{}}
		err := opt(c)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error reading jwt token file")
	})
}

// TestCertAuth tests the full cert auth flow against a real TLS-enabled Vault container.
// It generates ephemeral certs, starts Vault in non-dev TLS mode, configures cert auth,
// and verifies that the plugin can authenticate and perform transit encrypt/decrypt.
func TestCertAuth(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("docker sock unavailable on windows CI")
	}

	// 1. Generate ephemeral CA, server cert, client cert.
	certs, err := testutils.GenerateTestCerts()
	require.NoError(t, err, "generate test certs")

	// 2. Start TLS-enabled non-dev Vault container.
	tc, err := testutils.StartTLSTestContainer(t, certs)
	require.NoError(t, err, "start TLS vault container")

	defer func() { _ = tc.Terminate() }()

	// 3. Configure Vault: enable transit engine and create the KMS key.
	_, err = tc.ExecWithToken("vault secrets enable transit")
	require.NoError(t, err, "enable transit")

	_, err = tc.ExecWithToken("vault write -f transit/keys/kms")
	require.NoError(t, err, "create transit key")

	// 4. Enable cert auth and copy the CA cert into the container for the cert role.
	_, err = tc.ExecWithToken("vault auth enable cert")
	require.NoError(t, err, "enable cert auth")

	// Create a policy that allows the plugin's transit encrypt/decrypt operations.
	_, err = tc.ExecWithToken(
		`echo 'path "transit/*" { capabilities = ["create", "read", "update"] }' | vault policy write transit-pol -`,
	)
	require.NoError(t, err, "write transit policy")

	// Copy the CA cert into the container and write a cert role that trusts it.
	err = tc.Container.CopyFileToContainer(
		t.Context(),
		tc.CACertFile,
		"/tmp/vault-ca.crt",
		0o444,
	)
	require.NoError(t, err, "copy CA cert to container")

	_, err = tc.ExecWithToken(
		"vault write auth/cert/certs/kms certificate=@/tmp/vault-ca.crt policies=transit-pol",
	)
	require.NoError(t, err, "write cert role")

	// 5. Create a Vault client using cert auth and verify encrypt/decrypt works.
	// VAULT_CACERT must be set before NewClient so api.DefaultConfig() builds a
	// TLS transport that trusts Vault's self-signed test certificate.
	t.Setenv("VAULT_CACERT", tc.CACertFile)

	vc, err := NewClient(
		WithVaultAddress(tc.URI),
		WithTransit("transit", "kms"),
		WithCertAuth("cert", "kms", tc.ClientCertFile, tc.ClientKeyFile, tc.CACertFile),
	)
	require.NoError(t, err, "create vault client with cert auth")

	plaintext := []byte("hello-cert-auth")

	ciphertext, _, err := vc.Encrypt(t.Context(), plaintext)
	require.NoError(t, err, "encrypt")

	decrypted, err := vc.Decrypt(t.Context(), ciphertext)
	require.NoError(t, err, "decrypt")

	require.Equal(t, plaintext, decrypted, "decrypted plaintext must match original")
}

// TestJWTAuth tests the full JWT auth flow against a real dev-mode Vault container.
// It generates an ephemeral ES256 signing key, creates a signed JWT, configures Vault
// JWT auth with the matching static public key, and verifies transit encrypt/decrypt.
//
//nolint:funlen
func TestJWTAuth(t *testing.T) {
	if runtime.GOOS == windowsOS {
		t.Skip("docker sock unavailable on windows CI")
	}

	// 1. Generate ES256 signing key pair.
	privKey, pubKeyPEM, err := testutils.GenerateJWTSigningKey()
	require.NoError(t, err, "generate jwt signing key")

	// 2. Start dev-mode Vault with transit pre-enabled (no TLS needed for JWT auth).
	tc, err := testutils.StartTestContainer(
		"secrets enable transit",
		"write -f transit/keys/kms",
	)
	require.NoError(t, err, "start vault container")

	defer func() { _ = tc.Terminate() }()

	// 3. Sign a test JWT.
	jwtToken, err := testutils.SignTestJWT(privKey, "test-sa", "vault-kms")
	require.NoError(t, err, "sign test jwt")

	// 4. Write JWT to a temp file — WithJWTAuth reads the token from disk.
	tokenPath := filepath.Join(t.TempDir(), "jwt-token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(jwtToken), 0o600), "write jwt token file")

	// 5. Copy the public key PEM into the container; vault write can then use @/tmp/jwt-pub.pem.
	err = tc.Container.CopyToContainer(t.Context(), []byte(pubKeyPEM), "/tmp/jwt-pub.pem", 0o444)
	require.NoError(t, err, "copy public key to container")

	// 6. Enable JWT auth method.
	_, err = tc.ExecShell("vault auth enable jwt")
	require.NoError(t, err, "enable jwt auth")

	// 7. Configure JWT auth with the static public key (@ reads from file path in the container).
	_, err = tc.ExecShell("vault write auth/jwt/config jwt_validation_pubkeys=@/tmp/jwt-pub.pem")
	require.NoError(t, err, "configure jwt auth")

	// 8. Write transit policy (pipe requires ExecShell).
	_, err = tc.ExecShell(
		`printf 'path "transit/*" { capabilities = ["create","read","update"] }' | vault policy write transit-pol -`,
	)
	require.NoError(t, err, "write transit policy")

	// 9. Create a JWT role bound to our test subject and audience.
	_, err = tc.ExecShell(
		"vault write auth/jwt/role/kms role_type=jwt bound_audiences=vault-kms " +
			"user_claim=sub bound_subject=test-sa token_policies=transit-pol token_period=3600",
	)
	require.NoError(t, err, "write jwt role")

	// 10. Create vault client using JWT auth.
	vc, err := NewClient(
		WithVaultAddress(tc.URI),
		WithTransit("transit", "kms"),
		WithJWTAuth("jwt", "kms", tokenPath),
	)
	require.NoError(t, err, "create vault client with jwt auth")

	plaintext := []byte("hello-jwt-auth")

	ciphertext, _, err := vc.Encrypt(t.Context(), plaintext)
	require.NoError(t, err, "encrypt")

	decrypted, err := vc.Decrypt(t.Context(), ciphertext)
	require.NoError(t, err, "decrypt")

	require.Equal(t, plaintext, decrypted, "decrypted plaintext must match original")

	// 11. Token rotation: overwrite the token file and re-authenticate via AuthMethodFunc.
	// This proves that WithJWTAuth re-reads the file on each call, supporting K8s token rotation.
	t.Run("token rotation", func(t *testing.T) {
		newJWT, err := testutils.SignTestJWT(privKey, "test-sa", "vault-kms")
		require.NoError(t, err, "sign rotated jwt")

		require.NoError(t, os.WriteFile(tokenPath, []byte(newJWT), 0o600), "overwrite token file")
		require.NoError(t, vc.AuthMethodFunc(vc), "re-authenticate with rotated jwt")

		ciphertext, _, err := vc.Encrypt(t.Context(), plaintext)
		require.NoError(t, err, "encrypt after rotation")

		decrypted, err := vc.Decrypt(t.Context(), ciphertext)
		require.NoError(t, err, "decrypt after rotation")

		require.Equal(t, plaintext, decrypted, "decrypted must match after rotation")
	})
}
