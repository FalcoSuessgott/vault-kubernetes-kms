package vault

import (
	"fmt"
	"net/url"

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

	UserPassMount string
	Username      string
	Password      string

	CertAuthMount string
	CertRole      string
	CertFile      string
	CertKey       string

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

	// Wrap the existing transport (which carries any TLS config set by VAULT_CACERT env var
	// via api.DefaultConfig's ReadEnvironment) instead of replacing it wholesale.
	cfg.HttpClient = customHTTP.NewWithTransport(cfg.HttpClient.Transport)

	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	client := &Client{Client: c}

	for _, opt := range opts {
		err = opt(client)
		if err != nil {
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

// WithTokenRenewalSeconds sets the number of seconds used for token renewal.
func WithTokenRenewalSeconds(seconds int) Option {
	return func(c *Client) error {
		c.TokenRenewalSeconds = seconds

		return nil
	}
}

// WithAppRoleAuth performs an AppRole auth login.
func WithAppRoleAuth(mount, roleID, secretID string) Option {
	return func(c *Client) error {
		c.AppRoleID = roleID
		c.AppRoleMount = mount
		c.AppRoleSecretID = secretID

		opts := map[string]any{
			"role_id":   roleID,
			"secret_id": secretID,
		}

		s, err := c.Logical().Write(fmt.Sprintf(appRoleAuthLoginPath, mount), opts)
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

// WithUserPassAuth performs UserPass auth login.
func WithUserPassAuth(mount string, username string, password string) Option {
	return func(c *Client) error {
		c.UserPassMount = mount
		c.Username = username
		c.Password = password

		opts := map[string]any{
			"password": password,
		}

		s, err := c.Logical().Write(
			fmt.Sprintf(userPassAuthLoginPath, mount, url.PathEscape(username)),
			opts,
		)
		if err != nil {
			return fmt.Errorf("error performing userpass auth: %w", err)
		}

		c.SetToken(s.Auth.ClientToken)

		if c.AuthMethodFunc == nil {
			c.AuthMethodFunc = WithUserPassAuth(mount, username, password)
		}

		return nil
	}
}

// WithCertAuth performs Vault TLS Certificate auth login.
// The client certificate is presented during the TLS handshake; caFile should be the CA that
// signed the Vault server's TLS certificate (for the HTTPS connection). Vault verifies the
// client cert against cert roles configured with vault write auth/{mount}/certs/{name}.
func WithCertAuth(mount, role, certFile, keyFile, caFile string) Option {
	return func(c *Client) error {
		c.CertAuthMount = mount
		c.CertRole = role
		c.CertFile = certFile
		c.CertKey = keyFile

		// Build a temporary Vault SDK client configured with the client TLS certificate.
		// The client cert is presented during the TLS handshake, so it must be on the transport.
		tmpCfg := api.DefaultConfig()
		tmpCfg.Address = c.Address()

		if err := tmpCfg.ConfigureTLS(&api.TLSConfig{
			ClientCert: certFile,
			ClientKey:  keyFile,
			CACert:     caFile,
		}); err != nil {
			return fmt.Errorf("error configuring TLS for cert auth: %w", err)
		}

		tmpClient, err := api.NewClient(tmpCfg)
		if err != nil {
			return fmt.Errorf("error creating cert auth client: %w", err)
		}

		s, err := tmpClient.Logical().Write(
			fmt.Sprintf(certAuthLoginPath, mount),
			map[string]any{"name": role},
		)
		if err != nil {
			return fmt.Errorf("error performing cert auth: %w", err)
		}

		c.SetToken(s.Auth.ClientToken)

		if c.AuthMethodFunc == nil {
			c.AuthMethodFunc = WithCertAuth(mount, role, certFile, keyFile, caFile)
		}

		return nil
	}
}
