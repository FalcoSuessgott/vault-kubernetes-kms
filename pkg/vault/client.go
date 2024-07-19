package vault

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

// Client Vault API wrapper.
type Client struct {
	*api.Client

	Token string

	TokenRenewalInterval int

	AppRoleMount    string
	AppRoleID       string
	AppRoleSecretID string

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

		return nil
	}
}

// WitAppRoleAuth performs a approle auth login.
func WitAppRoleAuth(mount, roleID, secretID string) Option {
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

		return nil
	}
}

// WithTokenRenewal sets the specified namespace.
func WithTokenRenewal(d int) Option {
	return func(c *Client) error {
		c.TokenRenewalInterval = d

		return nil
	}
}

func (c *Client) TokenTTL() (int, error) {
	lookup, err := c.Auth().Token().LookupSelf()
	if err != nil {
		return 0, fmt.Errorf("error looking up token: %w", err)
	}

	return lookup.Data["ttl"].(int), nil
}

// TokenRenew renews the token by the specified interval.
func (c *Client) TokenRenew() error {
	_, err := c.Auth().Token().RenewSelf(c.TokenRenewalInterval)
	if err != nil {
		return fmt.Errorf("error renewing token: %w", err)
	}

	return nil
}
