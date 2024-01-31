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

	c      *testutils.TestContainer
	client *Client
}

func (s *VaultSuite) TearDownSubTest() {
	if err := s.c.Terminate(); err != nil {
		log.Fatal(err)
	}
}

func (s *VaultSuite) SetupSubTest() {
	c, err := testutils.StartTestContainer()
	if err != nil {
		log.Fatal(err)
	}

	s.c = c

	client, err := NewClient(
		WithVaultAddress(c.URI),
		WithVaultToken(c.Token),
		WithTransit("transit", "kms"),
	)
	if err != nil {
		log.Fatal(err)
	}

	s.client = client
}

func TestVaultSuite(t *testing.T) {
	// github actions doesn't offer the docker sock, which we require for testing
	if runtime.GOOS == "linux" {
		suite.Run(t, new(VaultSuite))
	}
}
