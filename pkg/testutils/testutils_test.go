package testutils

import (
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/stretchr/testify/require"
)

func TestVaultConnection(t *testing.T) {
	// create vault
	tc, err := StartTestContainer()
	require.NoError(t, err, "start")

	// create token
	token, err := tc.GetToken("default", "1h")
	require.NoError(t, err, "token creation")

	tokenVault, err := vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithTokenAuth(token),
	)
	require.NoError(t, err, "token login")

	// check unsealed / init
	health, err := tokenVault.Client.Sys().Health()
	require.NoError(t, err, "health")
	require.True(t, health.Initialized, "initialized")
	require.False(t, health.Sealed, "unsealed")

	// teardown
	require.NoError(t, tc.Terminate(), "terminate")
}

func TestVaultTokenAuth(t *testing.T) {
	// create vault
	tc, err := StartTestContainer("auth enable approle",
		"write auth/approle/role/kms token_ttl=1h",
	)
	require.NoError(t, err, "start")

	// create token
	token, err := tc.GetToken("default", "1h")
	require.NoError(t, err, "token creation")

	_, err = vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithTokenAuth(token),
		vault.WithTransit("transit", "kms"),
	)
	require.NoError(t, err, "token login")
	require.NoError(t, tc.Terminate(), "terminate")
}

func TestVaultAppRoleAuth(t *testing.T) {
	// create vault
	tc, err := StartTestContainer("auth enable approle",
		"write auth/approle/role/kms token_ttl=1h",
	)
	require.NoError(t, err, "start")

	roleID, secretID, err := tc.GetApproleCreds("approle", "kms")
	require.NoError(t, err, "approle creation")

	_, err = vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithAppRoleAuth("approle", roleID, secretID),
	)

	require.NoError(t, err, "health")
	require.NoError(t, tc.Terminate(), "terminate")
}

func TestVaultUserPassAuth(t *testing.T) {
	// create vault
	tc, err := StartTestContainer("auth enable userpass",
		"write auth/userpass/users/kms-user password=kms-pass",
	)
	require.NoError(t, err, "start")

	_, err = vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithUserPassAuth("userpass", "kms-user", "kms-pass"),
	)

	require.NoError(t, err, "health")
	require.NoError(t, tc.Terminate(), "terminate")
}
