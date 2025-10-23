package vault

import (
	"fmt"

	customHTTP "github.com/FalcoSuessgott/vault-kubernetes-kms/pkg/http"
	"github.com/hashicorp/vault/api"
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
	cfg.HttpClient = customHTTP.New()

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
