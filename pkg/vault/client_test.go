package vault

import (
	"context"
	"log"
	"runtime"
	"strings"
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

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
				_, _, err := s.tc.Container.Exec(context.Background(), strings.Split(cmd, " "))
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
	if runtime.GOOS != "windows" {
		suite.Run(t, new(VaultSuite))
	}
}

// TestCertAuth tests the full cert auth flow against a real TLS-enabled Vault container.
// It generates ephemeral certs, starts Vault in non-dev TLS mode, configures cert auth,
// and verifies that the plugin can authenticate and perform transit encrypt/decrypt.
func TestCertAuth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("docker sock unavailable on windows CI")
	}

	// 1. Generate ephemeral CA, server cert, client cert.
	certs, err := testutils.GenerateTestCerts()
	require.NoError(t, err, "generate test certs")

	// 2. Start TLS-enabled non-dev Vault container.
	tc, err := testutils.StartTLSTestContainer(certs)
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
		context.Background(),
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

	ciphertext, _, err := vc.Encrypt(context.Background(), plaintext)
	require.NoError(t, err, "encrypt")

	decrypted, err := vc.Decrypt(context.Background(), ciphertext)
	require.NoError(t, err, "decrypt")

	require.Equal(t, plaintext, decrypted, "decrypted plaintext must match original")
}
