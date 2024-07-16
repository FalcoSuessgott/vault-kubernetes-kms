package vault

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Encrypt takes any data and encrypts it using the specified vaults transit engine.
func (c *Client) Encrypt(ctx context.Context, data []byte) ([]byte, string, error) {
	p := fmt.Sprintf(encryptDataPath, c.TransitEngine, c.TransitKey)

	opts := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(data),
	}

	resp, err := c.Logical().WriteWithContext(ctx, p, opts)
	if err != nil {
		return nil, "", err
	}

	res, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return nil, "", errors.New("invalid response")
	}

	keyVersions, err := c.GetKeyVersions()
	if err != nil {
		return nil, "", err
	}

	v, ok := keyVersions[resp.Data["key_version"].(json.Number).String()].(json.Number)
	if !ok {
		return nil, "", fmt.Errorf("did not find key_version for transit key: %s", c.TransitKey)
	}

	return []byte(res), v.String(), nil
}

// Decrypt takes any encrypted data and denrypts it using the specified vaults transit engine.
func (c *Client) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	p := fmt.Sprintf(decryptDataPath, c.TransitEngine, c.TransitKey)

	opts := map[string]interface{}{
		"ciphertext": string(data),
	}

	resp, err := c.Logical().WriteWithContext(ctx, p, opts)
	if err != nil {
		return nil, err
	}

	res, ok := resp.Data["plaintext"].(string)
	if !ok {
		return nil, errors.New("invalid response")
	}

	decoded, err := base64.StdEncoding.DecodeString(res)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

// GetKeyVersions returns the key_versions aka the timestamp the key version was created.
// https://developer.hashicorp.com/vault/api-docs/secret/transit#read-key
func (c *Client) GetKeyVersions() (map[string]interface{}, error) {
	p := fmt.Sprintf(transitKeyPath, c.TransitEngine, c.TransitKey)

	resp, err := c.Logical().Read(p)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, fmt.Errorf("could not get key versions of transit key: %s/%s. Check transit engine and key and permissions", c.TransitEngine, c.TransitKey)
	}

	keys, ok := resp.Data["keys"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not get key_versions of transit key: %s/%s", c.TransitEngine, c.TransitKey)
	}

	return keys, nil
}
