package socket

import (
	"errors"
	"net"
	"strings"
)

// Listen listens on a given path, which is a unix domain socket.
func Listen(str string) (net.Listener, error) {
	var network, path string

	if strings.HasPrefix(strings.ToLower(str), "unix://") {
		//nolint: mnd
		s := strings.SplitN(str, "://", 2)
		if s[1] != "" {
			network, path = s[0], s[1]
		}
	}

	if path == "" {
		return nil, errors.New("path missing")
	}

	return net.Listen(network, path)
}
