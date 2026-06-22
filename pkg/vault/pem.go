package vault

import (
	"encoding/pem"
	"fmt"
	"os"
)

// ParseCombinedPEMFile reads a PEM file that contains both a CERTIFICATE block and a PRIVATE KEY
// block (e.g. kubelet's kubelet-client-current.pem) and writes each to a separate temp file.
// The caller must invoke the returned cleanup function when the temp files are no longer needed.
func ParseCombinedPEMFile(path string) (certFile, keyFile string, cleanup func(), err error) {
	certPEM, keyPEM, err := parseCombinedPEM(path)
	if err != nil {
		return "", "", func() {}, err
	}

	certF, err := os.CreateTemp("", "vault-cert-*.pem")
	if err != nil {
		return "", "", func() {}, fmt.Errorf("error creating temp cert file: %w", err)
	}

	if _, err = certF.Write(certPEM); err != nil {
		certF.Close()
		os.Remove(certF.Name())

		return "", "", func() {}, fmt.Errorf("error writing cert to temp file: %w", err)
	}

	certF.Close()

	keyF, err := os.CreateTemp("", "vault-key-*.pem")
	if err != nil {
		os.Remove(certF.Name())

		return "", "", func() {}, fmt.Errorf("error creating temp key file: %w", err)
	}

	if _, err = keyF.Write(keyPEM); err != nil {
		keyF.Close()
		os.Remove(certF.Name())
		os.Remove(keyF.Name())

		return "", "", func() {}, fmt.Errorf("error writing key to temp file: %w", err)
	}

	keyF.Close()

	return certF.Name(), keyF.Name(), func() {
		os.Remove(certF.Name())
		os.Remove(keyF.Name())
	}, nil
}

// parseCombinedPEM reads a PEM file and returns the CERTIFICATE and PRIVATE KEY blocks separately.
func parseCombinedPEM(path string) (certPEM []byte, keyPEM []byte, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read PEM file %s: %w", path, err)
	}

	rest := data

	for {
		var block *pem.Block

		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		switch block.Type {
		case "CERTIFICATE":
			certPEM = append(certPEM, pem.EncodeToMemory(block)...)
		case "EC PRIVATE KEY", "RSA PRIVATE KEY", "PRIVATE KEY":
			keyPEM = append(keyPEM, pem.EncodeToMemory(block)...)
		}
	}

	if certPEM == nil {
		return nil, nil, fmt.Errorf("no CERTIFICATE block found in %s", path)
	}

	if keyPEM == nil {
		return nil, nil, fmt.Errorf("no PRIVATE KEY block found in %s", path)
	}

	return certPEM, keyPEM, nil
}
