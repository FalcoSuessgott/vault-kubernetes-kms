package testutils

import (
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/vault"
	"github.com/stretchr/testify/require"
)

func TestVaultConnection(t *testing.T) {
	// create vault
	tc, err := StartTestContainer("secrets enable transit",
		"write -f transit/keys/kms",
		"auth enable approle",
		"write auth/approle/role/kms token_ttl=1h",
	)
	require.NoError(t, err, "start")

	// create token
	token, err := tc.GetToken("default")
	require.NoError(t, err, "token creation")

	tokenVault, err := vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithTokenAuth(token),
		vault.WithTransit("transit", "kms"),
	)
	require.NoError(t, err, "token login")

	// check unsealed / init
	health, err := tokenVault.Client.Sys().Health()
	require.NoError(t, err, "health")
	require.True(t, health.Initialized, "initialized")
	require.False(t, health.Sealed, "unsealed")

	// test approle
	roleID, secretID, err := tc.GetApproleCreds("approle", "kms")
	require.NoError(t, err, "approle creation")

	_, err = vault.NewClient(
		vault.WithVaultAddress(tc.URI),
		vault.WithTokenAuth(tc.Token),
		vault.WitAppRoleAuth("approle", roleID, secretID),
		vault.WithTransit("transit", "kms"),
	)

	require.NoError(t, err, "health")

	// teardown
	require.NoError(t, tc.Terminate(), "terminate")
}
