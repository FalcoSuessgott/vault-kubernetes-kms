package vault

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Encrypt takes any data and encrypts it using the specified vaults transit engine.
func (c *Client) TransitEncrypt(ctx context.Context, data []byte) ([]byte, string, error) {
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

	kv, err := c.TransitKeyVersion(ctx)
	if err != nil {
		return nil, "", err
	}

	return []byte(res), kv, nil
}

// Decrypt takes any encrypted data and decrypts it using the specified vaults transit engine.
func (c *Client) TransitDecrypt(ctx context.Context, data []byte) ([]byte, error) {
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

// GetKeyVersions returns the latest_version aka the timestamp the key version was created.
// https://developer.hashicorp.com/vault/api-docs/secret/transit#read-key
func (c *Client) TransitKeyVersion(ctx context.Context) (string, error) {
	p := fmt.Sprintf(transitKeyPath, c.TransitEngine, c.TransitKey)

	resp, err := c.Logical().ReadWithContext(ctx, p)
	if err != nil {
		return "", err
	}

	if resp == nil {
		return "", fmt.Errorf("could not read transit key: %s/%s. Check transit engine and key and permissions", c.TransitEngine, c.TransitKey)
	}

	kv, ok := resp.Data["latest_version"].(json.Number)
	if !ok {
		return "", fmt.Errorf("could not get latest_version of transit key: %s/%s", c.TransitEngine, c.TransitKey)
	}

	return kv.String(), nil
}
