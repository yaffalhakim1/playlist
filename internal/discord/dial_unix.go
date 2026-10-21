//go:build !windows

package discord

import (
	"net"
	"time"
)

// dial opens Discord's Unix socket, trying each place it may be listening.
func dial() (net.Conn, error) {
	for _, path := range socketPaths() {
		conn, err := net.DialTimeout("unix", path, time.Second)
		if err == nil {
			return conn, nil
		}
	}
	return nil, ErrNoDiscord
}
