package testutils

import (
	"context"
	"fmt"
	"io"
)

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
