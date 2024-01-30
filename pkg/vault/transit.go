package vault

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	encryptDataPath = "%s/encrypt/%s"
	decryptDataPath = "%s/decrypt/%s"

	mountEnginePath = "sys/mounts/%s"
	transitKeyPath  = "%s/keys/%s"
)

func (c *Client) Encrypt(data []byte) ([]byte, string, error) {
	p := fmt.Sprintf(encryptDataPath, c.TransitEngine, c.TransitKey)

	opts := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(data),
	}

	resp, err := c.Logical().Write(p, opts)
	if err != nil {
		return nil, "", err
	}

	res, ok := resp.Data["ciphertext"].(string)
	if !ok {
		return nil, "", fmt.Errorf("invalid response")
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

func (c *Client) Decrypt(data []byte) ([]byte, error) {
	p := fmt.Sprintf(decryptDataPath, c.TransitEngine, c.TransitKey)

	opts := map[string]interface{}{
		"ciphertext": string(data),
	}

	resp, err := c.Logical().Write(p, opts)
	if err != nil {
		return nil, err
	}

	res, ok := resp.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response")
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

	keys, ok := resp.Data["keys"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not get key_versions of transit key: %s/%s", c.TransitEngine, c.TransitKey)
	}

	return keys, nil
}

// EnableTransitEngine enables a transit engine under the given path.
func (c *Client) EnableTransitEngine(path string) error {
	options := map[string]interface{}{
		"type": "transit",
		"options": map[string]interface{}{
			"path": path,
		},
	}

	_, err := c.Logical().Write(fmt.Sprintf(mountEnginePath, path), options)
	if err != nil {
		return err
	}

	return nil
}

// CreateTransitKey enables a transit engine under the given path.
func (c *Client) CreateTransitKey(path, key string) error {
	p := fmt.Sprintf(transitKeyPath, path, key)

	_, err := c.Logical().Write(p, nil)
	if err != nil {
		return err
	}

	return nil
}
