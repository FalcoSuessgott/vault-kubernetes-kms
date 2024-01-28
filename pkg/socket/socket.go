package socket

import (
	"fmt"
	"net"
	"strings"
)

type Socket struct {
	Network string
	Path    string
}

func NewSocket(str string) (*Socket, error) {
	socket := &Socket{}

	if strings.HasPrefix(strings.ToLower(str), "unix://") {
		s := strings.SplitN(str, "://", 2)
		if s[1] != "" {
			socket.Network, socket.Path = s[0], s[1]
		}
	}

	if socket.Path == "" {
		return nil, fmt.Errorf("path missing")
	}

	return socket, nil
}

func (s *Socket) Listen() (net.Listener, error) {
	return net.Listen(s.Network, s.Path)
}
