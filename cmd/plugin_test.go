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
			name: "token & k8s auth",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
				VaultToken:   "abc",
				VaultK8sRole: "abc",
			},
		},
		{
			name: "token & k8s auth",
			err:  true,
			opts: &Options{
				VaultAddress: "e2e",
			},
		},
		{
			name: "no vault address",
			err:  true,
			opts: &Options{
				VaultToken: "abc",
			},
		},
		{
			name: "k8s auth",
			opts: &Options{
				VaultAddress: "e2e",
				VaultK8sRole: "kms",
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
