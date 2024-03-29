package vault

import (
	"fmt"
	"os"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// Client Vault API wrapper.
type Client struct {
	*api.Client

	TransitEngine string
	TransitKey    string
}

// Option vault client connection option.
type Option func(*Client) error

// NewClient returns a new vault client wrapper.
func NewClient(opts ...Option) (*Client, error) {
	// read all vault env vars
	c, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	client := &Client{
		Client: c,
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// WithVaultAddress sets the specified address.
func WithVaultAddress(address string) Option {
	return func(c *Client) error {
		return c.SetAddress(address)
	}
}

// WithVaultToken sets the specified token.
func WithVaultToken(token string) Option {
	return func(c *Client) error {
		if token != "" {
			c.SetToken(token)
		}

		return nil
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

// WithK8sAuth performs a k8s auth login.
func WithK8sAuth(mount, role string) Option {
	return func(c *Client) error {
		if role == "" {
			return nil
		}

		jwt, err := os.ReadFile(serviceAccountTokenLocation)
		if err != nil {
			return err
		}

		opts := map[string]interface{}{
			"role": role,
			"jwt":  string(jwt),
		}

		s, err := c.Logical().Write(fmt.Sprintf(k8sLoginPath, mount), opts)
		if err != nil {
			return fmt.Errorf("error performing k8s auth: %w", err)
		}

		c.SetToken(s.Auth.ClientToken)

		return nil
	}
}

// TokenRefresh renews the token for 24h.
func (c *Client) TokenRefresh() error {
	token, err := c.Auth().Token().RenewSelf(tokenRefreshIntervall)
	if err != nil {
		return fmt.Errorf("error renewing token: %w", err)
	}

	c.SetToken(token.Auth.ClientToken)

	zap.L().Info("successfully refreshed token")

	return nil
}
