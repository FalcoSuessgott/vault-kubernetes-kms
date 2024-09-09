package socket

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSocket(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		err  bool
	}{
		{
			name: "basic",
			str:  "unix:///opt/vaultkms.socket",
		},
		{
			name: "invalid",
			str:  "/opt/vaultkms.socket",
			err:  true,
		},
	}

	for _, tc := range testCases {
		_, err := Listen(tc.str)

		require.Equal(t, tc.err, err != nil, fmt.Sprintf("%s: %v", tc.name, err))
	}
}
