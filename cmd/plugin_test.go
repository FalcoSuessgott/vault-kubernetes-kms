package cmd

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// nolint: perfsprint, funlen
func TestNewPlugin(t *testing.T) {
	testCases := []struct {
		name      string
		envVars   map[string]string
		args      []string
		vaultCmd  []string
		extraArgs func(c *testutils.TestContainer) ([]string, error)
		err       bool
	}{
		{
			name: "token auth",
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/kms",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-auth-method=token",
				"-token=root",
				"-health-port=8081",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(c *testutils.TestContainer) ([]string, error) {
				return []string{fmt.Sprintf("-vault-address=%s", c.URI)}, nil
			},
		},
		{
			name: "approle auth",
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/kms",
				"auth enable approle",
				"write auth/approle/role/kms token_ttl=1h",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-auth-method=approle",
				"-health-port=8082",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(c *testutils.TestContainer) ([]string, error) {
				roleID, secretID, err := c.GetApproleCreds("approle", "kms")
				if err != nil {
					return nil, err
				}

				return []string{
					fmt.Sprintf("-vault-address=%s", c.URI),
					fmt.Sprintf("-approle-role-id=%s", roleID),
					fmt.Sprintf("-approle-secret-id=%s", secretID),
				}, nil
			},
		},
		{
			name: "mixed with env vars",
			envVars: map[string]string{
				"VAULT_KMS_TRANSIT_KEY":   "abc",
				"VAULT_KMS_TRANSIT_MOUNT": "transit",
				"VAULT_KMS_AUTH_METHOD":   "approle",
			},
			vaultCmd: []string{
				"secrets enable transit",
				"write -f transit/keys/abc",
				"auth enable -path=approle2 approle",
				"write auth/approle2/role/kms token_ttl=1h",
			},
			args: []string{
				"vault-kubernetes-kms",
				"-approle-mount=approle2",
				"-health-port=8083",
				fmt.Sprintf("-socket=unix:///%s/vaultkms.socket", t.TempDir()),
			},
			extraArgs: func(c *testutils.TestContainer) ([]string, error) {
				roleID, secretID, err := c.GetApproleCreds("approle2", "kms")
				if err != nil {
					return nil, err
				}

				return []string{
					fmt.Sprintf("-vault-address=%s", c.URI),
					fmt.Sprintf("-approle-role-id=%s", roleID),
					fmt.Sprintf("-approle-secret-id=%s", secretID),
				}, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vc, err := testutils.StartTestContainer(tc.vaultCmd...)
			require.NoError(t, err, "failed to start test container")

			//nolint: errcheck
			defer vc.Terminate()

			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			if tc.extraArgs != nil {
				extraArgs, err := tc.extraArgs(vc)
				require.NoError(t, err, "failed to perform extra args func: %w", err)

				tc.args = append(tc.args, extraArgs...)
			}

			os.Args = tc.args

			var wg sync.WaitGroup

			wg.Add(2)

			// invoke NewPlugin()
			go func() {
				defer wg.Done()

				if err := NewPlugin(""); err != nil {
					log.Fatal(err)
				}
			}()

			// cancel after 5 seconds to avoid test timeout
			go func() {
				defer wg.Done()

				time.AfterFunc(5*time.Second, func() {
					_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				})
			}()

			wg.Wait()
		})
	}
}

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
