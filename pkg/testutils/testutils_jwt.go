package testutils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"
)

// GenerateJWTSigningKey generates an ephemeral ECDSA P-256 key pair for JWT signing.
// Returns the private key and the PEM-encoded PKIX public key string suitable for
// Vault's jwt_validation_pubkeys configuration field.
func GenerateJWTSigningKey() (*ecdsa.PrivateKey, string, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("generate ecdsa key: %w", err)
	}

	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("marshal public key: %w", err)
	}

	pubKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyDER,
	}))

	return privKey, pubKeyPEM, nil
}

// es256SigSize is the byte length of an ES256 JWT signature (R || S, each 32 bytes for P-256).
const es256SigSize = 64

// SignTestJWT creates a minimal ES256-signed JWT suitable for Vault JWT auth testing.
// The token is valid for one hour from the time of signing.
func SignTestJWT(privKey *ecdsa.PrivateKey, subject, audience string) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "ES256",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)

	now := time.Now()

	payloadJSON, err := json.Marshal(map[string]any{
		"sub": subject,
		"aud": audience,
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal jwt payload: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + payload
	digest := sha256.Sum256([]byte(signingInput))

	r, s, err := ecdsa.Sign(rand.Reader, privKey, digest[:])
	if err != nil {
		return "", fmt.Errorf("ecdsa sign: %w", err)
	}

	// ES256 signature is R || S, each zero-padded to 32 bytes (P-256 curve order size).
	sig := make([]byte, es256SigSize)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
