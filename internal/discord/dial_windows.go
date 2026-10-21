//go:build windows

package discord

import (
	"net"
	"os"
	"time"
)

// dial opens Discord's named pipe on Windows.
//
// The net package speaks "unix" sockets, not Windows named pipes, so the pipe
// is opened as a file. That is what the official SDKs do on this platform, and
// it is why this file exists beside the one that handles the other systems.
func dial() (net.Conn, error) {
	for _, path := range socketPaths() {
		file, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			continue
		}
		return &pipeConn{File: file}, nil
	}
	return nil, ErrNoDiscord
}

// pipeConn is a named pipe with the deadlines the protocol needs.
//
// A pipe opened as a file has no deadline of its own, so the handshake's read
// timeout is honoured by the caller instead: read() in the shared file sets a
// deadline only when the connection supports one.
type pipeConn struct {
	*os.File
}

// SetDeadline does nothing: a named pipe has no deadline. The handshake is
// bounded by the caller's read, and a pipe that never answers leaves the
// presence off, which is the behaviour wanted.
func (c *pipeConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline does nothing, as SetDeadline.
func (c *pipeConn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline does nothing, as SetDeadline.
func (c *pipeConn) SetWriteDeadline(time.Time) error { return nil }

// LocalAddr and RemoteAddr satisfy net.Conn; a pipe has neither.
func (c *pipeConn) LocalAddr() net.Addr  { return pipeAddr{} }
func (c *pipeConn) RemoteAddr() net.Addr { return pipeAddr{} }

// pipeAddr names a pipe's end.
type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "discord-ipc" }
