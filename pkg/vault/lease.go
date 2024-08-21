package vault

import (
	"context"
	"encoding/json"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"go.uber.org/zap"
)

// LeaseRefresher periodically checks the ttl of the current lease and attempts to renew it if the ttl is less than half of the creation ttl.
// if the token renewal fails, a new login with the configured auth method is performed
// this func is supposed to run as a goroutine.
// nolint: funlen, gocognit, cyclop
func (c *Client) LeaseRefresher(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			token, err := c.Auth().Token().LookupSelf()
			if err != nil {
				zap.L().Error("failed to lookup token", zap.Error(err))

				continue
			}

			creationTTL, ok := token.Data["creation_ttl"].(json.Number)
			if !ok {
				zap.L().Error("failed to assert creation_ttl type")

				continue
			}

			ttl, ok := token.Data["ttl"].(json.Number)
			if !ok {
				zap.L().Error("failed to assert ttl type")

				continue
			}

			creationTTLFloat, err := creationTTL.Float64()
			if err != nil {
				zap.L().Error("failed to parse creation_ttl", zap.Error(err))

				continue
			}

			ttlFloat, err := ttl.Float64()
			if err != nil {
				zap.L().Error("failed to parse ttl", zap.Error(err))

				continue
			}

			metrics.VaultTokenExpirySeconds.Set(ttlFloat)

			zap.L().Info("checking token renewal", zap.Float64("creation_ttl", creationTTLFloat), zap.Float64("ttl", ttlFloat))

			//nolint: nestif
			if ttlFloat < creationTTLFloat/2 {
				zap.L().Info("attempting token renewal", zap.Int("renewal_seconds", c.TokenRenewalSeconds))

				if _, err := c.Auth().Token().RenewSelf(c.TokenRenewalSeconds); err != nil {
					zap.L().Error("failed to renew token, performing new authentication", zap.Error(err))

					if err := c.AuthMethodFunc(c); err != nil {
						zap.L().Error("failed to authenticate", zap.Error(err))
					} else {
						zap.L().Info("successfully re-authenticated")
					}
				} else {
					zap.L().Info("successfully refreshed token")
				}

				metrics.VaultTokenRenewalTotal.Inc()
			} else {
				zap.L().Info("skipping token renewal")
			}

		case <-ctx.Done():
			zap.L().Info("token refresher shutting down")

			return
		}
	}
}
