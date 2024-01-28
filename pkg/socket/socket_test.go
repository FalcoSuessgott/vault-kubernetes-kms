package socket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSocket(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		exp  *Socket
		err  bool
	}{
		{
			name: "basic",
			str:  "unix:///opt/vaultkms.socket",
			exp: &Socket{
				Network: "unix",
				Path:    "/opt/vaultkms.socket",
			},
		},
		{
			name: "invalid",
			str:  "/opt/vaultkms.socket",
			err:  true,
		},
	}

	for _, tc := range testCases {
		s, err := NewSocket(tc.str)

		if tc.err {
			require.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)

			assert.Equal(t, tc.exp, s, tc.name)
		}
	}
}
