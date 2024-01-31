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
			// enable transit engine
			require.NoError(s.Suite.T(), s.client.EnableTransitEngine("transit"), tc.name)

			// create a transit key
			require.NoError(s.Suite.T(), s.client.CreateTransitKey("transit", "kms"), tc.name)

			// encrypt data
			enc, _, err := s.client.Encrypt(context.Background(), tc.data)
			require.NoError(s.Suite.T(), err, tc.name)

			// decrypt data
			dec, err := s.client.Decrypt(context.Background(), enc)
			require.NoError(s.Suite.T(), err, tc.name)

			// data should match decrypted text
			assert.Equal(s.Suite.T(), tc.data, dec, tc.name)
		})
	}
}
