package testutils

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/vault"
)

var (
	image = "hashicorp/vault:1.16.0"
	token = "root"
)

// TestContainer vault dev container wrapper.
type TestContainer struct {
	Container testcontainers.Container
	URI       string
	Token     string
}

// StartTestContainer Starts a fresh vault in development mode.
func StartTestContainer(commands ...string) (*TestContainer, error) {
	ctx := context.Background()

	vaultContainer, err := vault.Run(ctx,
		image,
		vault.WithToken(token),
		vault.WithInitCommand(commands...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	uri, err := vaultContainer.HttpHostAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("error returning container mapped port: %w", err)
	}

	return &TestContainer{
		Container: vaultContainer,
		URI:       uri,
		Token:     token,
	}, nil
}

func (v *TestContainer) GetApproleCreds(mount, role string) (string, string, error) {
	_, r, err := v.Container.Exec(context.Background(), []string{"vault", "read", "-field=role_id", fmt.Sprintf("auth/%s/role/%s/role-id", mount, role)})
	if err != nil {
		return "", "", fmt.Errorf("error creating role_id: %w", err)
	}

	roleID, err := io.ReadAll(r)
	if err != nil {
		return "", "", fmt.Errorf("error reading role_id: %w", err)
	}

	_, r, err = v.Container.Exec(context.Background(), []string{"vault", "write", "-field=secret_id", "-force", fmt.Sprintf("auth/%s/role/%s/secret-id", mount, role)})
	if err != nil {
		return "", "", fmt.Errorf("error creating secret_id: %w", err)
	}

	secretID, err := io.ReadAll(r)
	if err != nil {
		return "", "", fmt.Errorf("error reading secret_id: %w", err)
	}

	// removing the first 8 bytes, which is the shell prompt
	return string(roleID[8:]), string(secretID[8:]), nil
}

// nolint: perfsprint
func (v *TestContainer) GetToken(policy string, ttl string) (string, error) {
	return v.RunCommand("vault token create -field=token -policy=" + policy + " -ttl=" + ttl)
}

func (v *TestContainer) RunCommand(cmd string) (string, error) {
	log.Println("running command: ", cmd)

	_, r, err := v.Container.Exec(context.Background(), strings.Split(cmd, " "))
	if err != nil {
		return "", fmt.Errorf("error creating root token: %w", err)
	}

	rootToken, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("error reading root token: %w", err)
	}

	log.Println("output: ", string(rootToken[8:]))

	// removing the first 8 bytes, which is the shell prompt
	return string(rootToken[8:]), nil
}

// Terminate terminates the testcontainer.
func (v *TestContainer) Terminate() error {
	return v.Container.Terminate(context.Background())
}
