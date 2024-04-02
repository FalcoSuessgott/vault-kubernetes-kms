package vault

import (
	"log"
	"runtime"
	"testing"

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
		WithVaultToken(tc.Token),
		WithTransit("transit", "kms"),
	)
	if err != nil {
		log.Fatal(err)
	}

	s.vault = vault
}

func TestVaultSuite(t *testing.T) {
	// github actions doesn't offer the docker sock, which we require for testing
	if runtime.GOOS == "linux" {
		suite.Run(t, new(VaultSuite))
	}
}
