package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEnvs(t *testing.T) {
	type test struct {
		Test string `env:"TEST"`
	}

	o := &test{}
	exp := "test"

	t.Setenv("test_TEST", exp)

	require.NoError(t, ParseEnvs("test_", o))
	require.Equal(t, exp, o.Test)
}
