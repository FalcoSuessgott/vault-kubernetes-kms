package vault

import (
	"context"
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
				roleID, secretID, err := s.tc.GetApproleCreds("approle", "kms")
				if err != nil {
					return nil, err
				}

				return WitAppRoleAuth("approle", roleID, secretID), nil
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
				token, err := s.tc.GetToken("default")
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
	if runtime.GOOS == "linux" {
		suite.Run(t, new(VaultSuite))
	}
}
