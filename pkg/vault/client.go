package vault

import (
	"os"

	"github.com/hashicorp/vault/api"
)

// Client Vault API wrapper.
type Client struct {
	*api.Client

	Address string
	Token   string

	TransitEngine string
	TransitKey    string
}

// NewClient returns a new vault client wrapper.
func NewClient(addr, token, transitEngine, transitKey string) (*Client, error) {
	// read all vault env vars
	c, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	// set address
	if err = c.SetAddress(addr); err != nil {
		return nil, err
	}

	// set token
	c.SetToken(token)

	// and namespace
	if vaultNamespace, ok := os.LookupEnv("VAULT_NAMESPACE"); ok {
		c.SetNamespace(vaultNamespace)
	}

	return &Client{
		Client:        c,
		Address:       addr,
		Token:         token,
		TransitEngine: transitEngine,
		TransitKey:    transitKey,
	}, nil
}
