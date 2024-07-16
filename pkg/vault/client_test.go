package vault

import (
	"context"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"testing"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/exec"
)

type VaultSuite struct {
	suite.Suite

	tc    *testutils.TestContainer
	vault *Client
}

func (s *VaultSuite) TearDownSubTest() {
	if err := s.tc.Terminate(); err != nil {
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
		name       string
		prepCmd    []string
		cmdOptions []exec.ProcessOption
		auth       func() (Option, error)
		err        bool
	}{
		{
			name: "basic approle auth",
			prepCmd: []string{
				"vault auth enable approle",
				"vault write auth/approle/role/kms token_ttl=1h",
			},
			auth: func() (Option, error) {
				_, r, err := s.tc.Container.Exec(context.Background(), []string{"vault", "read", "-field=role_id", "auth/approle/role/kms/role-id"})
				if err != nil {
					return nil, fmt.Errorf("error creating role_id: %w", err)
				}

				roleID, err := io.ReadAll(r)
				if err != nil {
					return nil, fmt.Errorf("error reading role_id: %w", err)
				}

				_, r, err = s.tc.Container.Exec(context.Background(), []string{"vault", "write", "-field=secret_id", "-force", "auth/approle/role/kms/secret-id"})
				if err != nil {
					return nil, fmt.Errorf("error creating secret_id: %w", err)
				}

				secretID, err := io.ReadAll(r)
				if err != nil {
					return nil, fmt.Errorf("error reading secret_id: %w", err)
				}

				// removing the first 8 bytes, which is the shell prompt
				return WitAppRoleAuth("approle", string(roleID[8:]), string(secretID[8:])), nil
			},
		},
		{
			name: "invalid approle auth",
			err:  true,
			auth: func() (Option, error) {
				return WitAppRoleAuth("approle", "invalid", "invalid"), nil
			},
		},
		{
			name: "token auth",
			auth: func() (Option, error) {
				_, r, err := s.tc.Container.Exec(context.Background(), []string{"vault", "token", "create", "-field=token"})
				if err != nil {
					return nil, fmt.Errorf("error creating role_id: %w", err)
				}

				token, err := io.ReadAll(r)
				if err != nil {
					return nil, fmt.Errorf("error reading role_id: %w", err)
				}

				// removing the first 8 bytes, which is the shell prompt
				return WithTokenAuth(string(token[8:])), nil
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
	if runtime.GOOS == "linux" {
		suite.Run(t, new(VaultSuite))
	}
}
