package socket

import (
	"errors"
	"net"
	"strings"
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
func (s *Socket) Listen() (net.Listener, error) {
	return net.Listen(s.Network, s.Path)
}
