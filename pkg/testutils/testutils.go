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

const dockerMuxHeaderSize = 8

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

// GetToken creates a token with the supplied policy and TTL.
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

// ExecShell runs a command via "sh -c" inside the container.
// Use when the command requires shell features such as pipes or redirection.
func (v *TestContainer) ExecShell(cmd string) (string, error) {
	log.Println("running shell command: ", cmd)

	_, r, err := v.Container.Exec(context.Background(), []string{
		"sh", "-c",
		"export VAULT_TOKEN=" + v.Token + " && " + cmd,
	})
	if err != nil {
		return "", fmt.Errorf("error executing shell command: %w", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("error reading shell output: %w", err)
	}

	if len(out) > dockerMuxHeaderSize {
		out = out[dockerMuxHeaderSize:]
	}

	log.Println("output: ", string(out))

	return string(out), nil
}

// Terminate terminates the testcontainer.
func (v *TestContainer) Terminate() error {
	return v.Container.Terminate(context.Background())
}
