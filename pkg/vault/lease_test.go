package vault

import (
	"context"
	"strings"
	"time"
)

func (s *VaultSuite) TestTokenRefresher() {
	// here we simply create a token with a TTL of 7sec and start the token refresher
	// after 20sec we check if the token is still valid
	s.Run("token refresher", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		token, err := s.tc.GetToken("default", "7s")
		s.Require().NoError(err, "token creation failed")

		vc, err := NewClient(
			WithVaultAddress(s.tc.URI),
			WithTokenAuth(token),
		)

		s.Require().NoError(err, "client")

		go vc.LeaseRefresher(ctx, 3*time.Second)

		time.Sleep(15 * time.Second)

		_, err = s.tc.RunCommand("vault token lookup " + token)
		s.Require().NoError(err, "token lookup")
	})
}

func (s *VaultSuite) TestTokenRefresherReauthenticatesOnLookupFailure() {
	s.Run("reauthenticates after token lookup failure", func() {
		const (
			mount = "approle-lease-test"
			role  = "lease-reauth-test"
		)

		_, err := s.tc.RunCommand("vault auth enable -path=" + mount + " approle")
		s.Require().NoError(err)

		_, err = s.tc.RunCommand(
			"vault write auth/" + mount + "/role/" + role +
				" token_policies=default token_ttl=1h token_max_ttl=4h",
		)
		s.Require().NoError(err)

		roleID, err := s.tc.RunCommand(
			"vault read -field=role_id auth/" + mount + "/role/" + role + "/role-id",
		)
		s.Require().NoError(err)

		secretID, err := s.tc.RunCommand(
			"vault write -f -field=secret_id auth/" + mount + "/role/" + role + "/secret-id",
		)
		s.Require().NoError(err)

		vc, err := NewClient(
			WithVaultAddress(s.tc.URI),
			WithAppRoleAuth(mount, strings.TrimSpace(roleID), strings.TrimSpace(secretID)),
		)
		s.Require().NoError(err)

		oldToken := vc.Client.Token()
		s.Require().NotEmpty(oldToken)

		_, err = s.tc.RunCommand("vault token revoke " + oldToken)
		s.Require().NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go vc.LeaseRefresher(ctx, 500*time.Millisecond)

		s.Eventually(func() bool {
			_, err := vc.Auth().Token().LookupSelf()
			return err == nil && vc.Client.Token() != oldToken
		}, 8*time.Second, 250*time.Millisecond)
	})
}
