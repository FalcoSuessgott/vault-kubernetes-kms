package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/metrics"
	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// Client Vault API wrapper.
type Client struct {
	*api.Client

	Token string

	AppRoleMount    string
	AppRoleID       string
	AppRoleSecretID string

	AuthMethodFunc Option

	TokenRenewalSeconds int

	TransitEngine string
	TransitKey    string
}

// Option vault client connection option.
type Option func(*Client) error

// NewClient returns a new vault client wrapper.
func NewClient(opts ...Option) (*Client, error) {
	cfg := api.DefaultConfig()

	// read all vault env vars
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	client := &Client{Client: c}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	// perform a self lookup to verify the token
	_, err = c.Auth().Token().LookupSelf()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vault: %w", err)
	}

	return client, nil
}

// WithVaultAddress sets the specified address.
func WithVaultAddress(address string) Option {
	return func(c *Client) error {
		return c.SetAddress(address)
	}
}

// WithVaultNamespace sets the specified namespace.
func WithVaultNamespace(namespace string) Option {
	return func(c *Client) error {
		if namespace != "" {
			c.SetNamespace(namespace)
		}

		return nil
	}
}

// WithTransit sets transit parameters.
func WithTransit(mount, key string) Option {
	return func(c *Client) error {
		c.TransitEngine = mount
		c.TransitKey = key

		return nil
	}
}

// WithTokenAuth sets the specified token.
func WithTokenAuth(token string) Option {
	return func(c *Client) error {
		c.Token = token

		if token != "" {
			c.SetToken(token)
		}

		if c.AuthMethodFunc == nil {
			c.AuthMethodFunc = WithTokenAuth(token)
		}

		return nil
	}
}

// WithTokenAuth sets the specified token.
func WithTokenRenewalSeconds(seconds int) Option {
	return func(c *Client) error {
		c.TokenRenewalSeconds = seconds

		return nil
	}
}

// WitAppRoleAuth performs a approle auth login.
func WithAppRoleAuth(mount, roleID, secretID string) Option {
	return func(c *Client) error {
		c.AppRoleID = roleID
		c.AppRoleMount = mount
		c.AppRoleSecretID = secretID

		opts := map[string]interface{}{
			"role_id":   roleID,
			"secret_id": secretID,
		}

		s, err := c.Logical().Write(fmt.Sprintf(authLoginPath, mount), opts)
		if err != nil {
			return fmt.Errorf("error performing approle auth: %w", err)
		}

		c.SetToken(s.Auth.ClientToken)

		if c.AuthMethodFunc == nil {
			c.AuthMethodFunc = WithAppRoleAuth(mount, roleID, secretID)
		}

		return nil
	}
}

// TokenRefresher periodically checks the ttl of the current token and attempts to renew it if the ttl is less than half of the creation ttl.
// if the token renewal fails, a new login with the configured auth method is performed
// this func is supposed to run as a goroutine.
// nolint: funlen, gocognit, cyclop
func (c *Client) TokenRefresher(ctx context.Context, interval time.Duration) {
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
