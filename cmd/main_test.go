package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFlags(t *testing.T) {
	testCases := []struct {
		name string
		opts *options
		err  bool
	}{
		{
			name: "token & k8s auth",
			err:  true,
			opts: &options{
				vaultAddress: "e2e",
				vaultToken:   "abc",
				vaultK8sRole: "abc",
			},
		},
		{
			name: "token & k8s auth",
			err:  true,
			opts: &options{
				vaultAddress: "e2e",
			},
		},
		{
			name: "no vault address",
			err:  true,
			opts: &options{
				vaultToken: "abc",
			},
		},
		{
			name: "k8s auth",
			opts: &options{
				vaultAddress: "e2e",
				vaultK8sRole: "kms",
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
