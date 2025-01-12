package vault

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/vault/api"
	"golang.org/x/net/context"
)

// Client Vault API wrapper.
type Client struct {
	*api.Client

	// move out of struct ...
	ctx context.Context

	token string

	appRoleMount    string
	appRoleID       string
	appRoleSecretID string

	authMethodFunc Option

	tokenRenewalSeconds int

	transitEngine string
	transitKey    string
}

// Option vault client connection option.
type Option func(client *Client) error

// NewClient returns a new vault client wrapper.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	cfg := api.DefaultConfig()

	// read all vault env vars
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	client := &Client{
		ctx:    ctx,
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
		c.transitEngine = mount
		c.transitKey = key

		return nil
	}
}

// WithTokenAuth sets the specified token.
func WithTokenAuth(token string) Option {
	return func(c *Client) error {
		c.token = token

		if token != "" {
			c.SetToken(token)
		}

		if c.authMethodFunc == nil {
			c.authMethodFunc = WithTokenAuth(token)
		}

		return nil
	}
}

// WithTokenAuth sets the specified token.
func WithTokenRenewalSeconds(seconds int) Option {
	return func(c *Client) error {
		c.tokenRenewalSeconds = seconds

		return nil
	}
}

// WitAppRoleAuth performs a approle auth login.
func WithAppRoleAuth(mount, roleID, secretID string) Option {
	return func(c *Client) error {
		c.appRoleID = roleID
		c.appRoleMount = mount
		c.appRoleSecretID = secretID

		opts := map[string]interface{}{
			"role_id":   roleID,
			"secret_id": secretID,
		}

		s, err := c.Logical().WriteWithContext(c.ctx, fmt.Sprintf(authLoginPath, mount), opts)
		if err != nil {
			return fmt.Errorf("error performing approle auth: %w", err)
		}

		c.SetToken(s.Auth.ClientToken)

		if c.authMethodFunc == nil {
			c.authMethodFunc = WithAppRoleAuth(mount, roleID, secretID)
		}

		return nil
	}
}

func WithTLSAuth(mount, role, key, cert, ca string) Option {
	return func(c *Client) error {
		caCert, err := os.ReadFile(ca)
		if err != nil {
			return fmt.Errorf("cannot read CA file %s: %w", ca, err)
		}

		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caCert)

		clientCert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return fmt.Errorf("cannot load key pair (key: %s, cert: %s): %w", key, cert, err)
		}

		tlsConfig := tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{clientCert},
		}

		transport := http.Transport{
			TLSClientConfig: &tlsConfig,
		}

		httpClient := http.Client{
			Transport: &transport,
		}

		opts := map[string]interface{}{
			"name": role,
		}

		payload, err := json.Marshal(opts)
		if err != nil {
			return fmt.Errorf("error marshaling payload: %w", err)
		}

		resp, err := httpClient.Post(fmt.Sprintf("%s/v1/"+authLoginPath, c.Address(), mount), "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("error performing tls auth: %w", err)
		}

		defer resp.Body.Close()

		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}

		data := make(map[string]interface{})
		if err := json.Unmarshal(out, &data); err != nil {
			return fmt.Errorf("error unmarshaling response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("error during tls auth: %s", resp.Status)
		}

		c.SetToken(data["auth"].(map[string]interface{})["client_token"].(string))

		if c.authMethodFunc == nil {
			c.authMethodFunc = WithTLSAuth(mount, role, key, cert, ca)
		}

		return nil
	}
}
