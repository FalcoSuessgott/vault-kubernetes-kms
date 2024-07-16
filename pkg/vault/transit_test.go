package vault

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *VaultSuite) TestTransitEncryptDecrypt() {
	testCases := []struct {
		name string
		data []byte
		err  bool
	}{
		{
			name: "encrypt decrypt",
			data: []byte("simple string"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// encrypt data
			enc, _, err := s.vault.Encrypt(context.Background(), tc.data)
			require.NoError(s.Suite.T(), err, tc.name)

			// decrypt data
			dec, err := s.vault.Decrypt(context.Background(), enc)
			require.NoError(s.Suite.T(), err, tc.name)

			// data should match decrypted text
			assert.Equal(s.Suite.T(), tc.data, dec, tc.name)
		})
	}
}

func (s *VaultSuite) TestTransitKeyVersion() {
	testCases := []struct {
		name    string
		transit Option
		err     bool
	}{
		{
			name:    "should work",
			transit: WithTransit("transit", "kms"),
		},
		{
			name:    "should fail",
			transit: WithTransit("doesnot", "exist"),
			err:     true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			vault, err := NewClient(
				WithVaultAddress(s.tc.URI),
				WithVaultToken(s.tc.Token),
				tc.transit,
			)

			s.Suite.Require().NoError(err)

			_, err = vault.GetKeyVersions()

			s.Suite.Require().Equal(tc.err, err != nil)
		})
	}
}
