package vault

import (
	"context"
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
			context.Background(),
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
