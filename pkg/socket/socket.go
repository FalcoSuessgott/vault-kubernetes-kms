package socket

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"go.uber.org/zap"
)

// Socket represents a unix socket.
type Socket struct {
	Network string
	Path    string
}

// NewSocket returns a new unix socket.
func NewSocket(str string) (*Socket, error) {
	socket := &Socket{}

	if strings.HasPrefix(strings.ToLower(str), "unix://") {
		//nolint: mnd
		s := strings.SplitN(str, "://", 2)
		if s[1] != "" {
			socket.Network, socket.Path = s[0], s[1]
		}
	}

	if socket.Path == "" {
		return nil, errors.New("path missing")
	}

	return socket, nil
}

// Listen listens on the current socket for connections.
func (s *Socket) Listen(force bool) (net.Listener, error) {
	// Remove the socket file if it already exists.
	if _, err := os.Stat(s.Path); err == nil {
		zap.L().Info("Socket already exists", zap.String("path", s.Path))

		if force {
			if err := os.Remove(s.Path); err != nil {
				return nil, fmt.Errorf("failed to remove unix socket: %w", err)
			}

			zap.L().Info("Socket overwrite is enabled. Successfully removed socket", zap.String("path", s.Path))
		}
	}

	return net.Listen(s.Network, s.Path)
}
