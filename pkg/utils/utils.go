package utils

import (
	"github.com/caarlos0/env/v6"
)

// ParseEnvs parses the environment variables and sets the options.
func ParseEnvs(prefix string, i interface{}) error {
	opts := env.Options{
		Prefix: prefix,
	}

	if err := env.Parse(i, opts); err != nil {
		return err
	}

	return nil
}
