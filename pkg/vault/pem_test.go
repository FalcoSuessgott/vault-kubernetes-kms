package vault

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//nolint:funlen
func TestParseCombinedPEM(t *testing.T) {
	// Generate a real cert+key pair.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	certBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	t.Run("combined cert+key PEM", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "combined-*.pem")
		require.NoError(t, err)

		defer os.Remove(f.Name())

		// Write cert first, then key — matches kubelet-client-current.pem layout.
		_, err = f.Write(certBlock)
		require.NoError(t, err)
		_, err = f.Write(keyBlock)
		require.NoError(t, err)
		f.Close()

		gotCert, gotKey, err := parseCombinedPEM(f.Name())
		require.NoError(t, err)
		require.Equal(t, certBlock, gotCert, "cert PEM should match")
		require.Equal(t, keyBlock, gotKey, "key PEM should match")
	})

	t.Run("key before cert is also accepted", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "reversed-*.pem")
		require.NoError(t, err)

		defer os.Remove(f.Name())

		_, err = f.Write(keyBlock)
		require.NoError(t, err)
		_, err = f.Write(certBlock)
		require.NoError(t, err)
		f.Close()

		gotCert, gotKey, err := parseCombinedPEM(f.Name())
		require.NoError(t, err)
		require.Equal(t, certBlock, gotCert)
		require.Equal(t, keyBlock, gotKey)
	})

	t.Run("missing key returns error", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "cert-only-*.pem")
		require.NoError(t, err)

		defer os.Remove(f.Name())

		_, err = f.Write(certBlock)
		require.NoError(t, err)
		f.Close()

		_, _, err = parseCombinedPEM(f.Name())
		require.ErrorContains(t, err, "no PRIVATE KEY")
	})

	t.Run("missing cert returns error", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "key-only-*.pem")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.Write(keyBlock)
		require.NoError(t, err)
		f.Close()

		_, _, err = parseCombinedPEM(f.Name())
		require.ErrorContains(t, err, "no CERTIFICATE")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		_, _, err := parseCombinedPEM("/nonexistent/path/to/cert.pem")
		require.Error(t, err)
	})

	t.Run("ParseCombinedPEMFile writes host temp files and returns cleanup", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "combined-*.pem")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.Write(certBlock)
		require.NoError(t, err)
		_, err = f.Write(keyBlock)
		require.NoError(t, err)
		f.Close()

		certFile, keyFile, cleanup, err := ParseCombinedPEMFile(f.Name())
		require.NoError(t, err)

		// Temp files exist.
		_, statErr := os.Stat(certFile)
		require.NoError(t, statErr, "cert temp file should exist")

		_, statErr = os.Stat(keyFile)
		require.NoError(t, statErr, "key temp file should exist")

		// Cleanup removes them.
		cleanup()

		_, statErr = os.Stat(certFile)
		require.ErrorIs(t, statErr, os.ErrNotExist, "cert temp file should be removed after cleanup")

		_, statErr = os.Stat(keyFile)
		require.ErrorIs(t, statErr, os.ErrNotExist, "key temp file should be removed after cleanup")
	})
}
