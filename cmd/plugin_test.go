package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFlags(t *testing.T) {
	testCases := []struct {
		name string
		opts *Options
		err  bool
	}{
		{
			name: "no vault address",
			err:  true,
			opts: &Options{
				Token: "abc",
			},
		},
		{
			name: "invalid auth method",
			err:  true,
			opts: &Options{
				AuthMethod: "invalid",
			},
		},
		{
			name: "token auth, but no token",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "token",
			},
		},
		{
			name: "approle auth, but no approle creds",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "approle",
			},
		},
		{
			name: "k8s auth, but no k8s creds",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				AuthMethod:   "k8s",
			},
		},
	}

	for _, tc := range testCases {
		err := tc.opts.validateFlags()
		if tc.err {
			require.Error(t, err, tc.name)

			continue
		}

		require.NoError(t, err, tc.name)
	}
}
