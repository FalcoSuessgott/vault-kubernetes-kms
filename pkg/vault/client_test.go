package vault

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/stretchr/testify/suite"
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

func (s *VaultSuite) TestTokenRefresher() {
	s.Run("c", func() {
		s.T().Fail()

		token, err := s.tc.GetToken("default", "1h")
		s.Require().NoError(err, "token creation")

		vc, err := NewClient(
			WithVaultAddress(s.tc.URI),
			WithTokenAuth(token),
		)
		s.Require().NoError(err, "client")

		vc.TokenRefresher(context.Background(), 5*time.Second)

		fmt.Println(token)

		s.T().Fail()
	})
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
	if runtime.GOOS == "linux" {
		suite.Run(t, new(VaultSuite))
	}
}
