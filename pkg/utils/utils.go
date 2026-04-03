package utils

import (
	"github.com/caarlos0/env/v6"
)

// ParseEnvs parses the environment variables and sets the options.
func ParseEnvs(prefix string, i any) error {
	opts := env.Options{
		Prefix: prefix,
	}

	err := env.Parse(i, opts)
	if err != nil {
		return err
	}

	return nil
}
