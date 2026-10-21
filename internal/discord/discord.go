// Package discord shows what is playing in Discord, through its local socket.
//
// Discord Rich Presence is a small protocol over a named pipe on Windows and a
// Unix socket elsewhere: a handshake with the application's client id, then a
// frame whenever the presence changes. There is no SDK and no network call; the
// desktop's own Discord client draws the panel.
//
// The package is written to fail quietly. Discord is often not running, the
// application id may not be set, and neither is a reason for the music to stop:
// every call here returns an error the caller may log and ignore.
package discord

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// The opcodes of the protocol.
const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
	opPing      = 3
	opPong      = 4
)

// maxFrame bounds a reply, so a wrong pipe cannot make the app allocate without
// end.
const maxFrame = 1 << 20

// readTimeout bounds the wait for Discord to answer a handshake.
const readTimeout = 5 * time.Second

// Client is a connection to the desktop's Discord client.
type Client struct {
	mu   sync.Mutex
	conn net.Conn
	id   string
}

// ErrNoDiscord reports that no Discord client is listening, which is the normal
// state on a machine that does not run it.
var ErrNoDiscord = errors.New("discord: no client is running")

// ErrNoClientID reports that the application has no Discord application id.
var ErrNoClientID = errors.New("discord: no application id is configured")

// Connect finds the running Discord client and shakes hands with it.
//
// It looks for the first socket Discord listens on, as the official SDKs do:
// discord-ipc-0 through 9, and a snap's socket on Linux.
func Connect(clientID string) (*Client, error) {
	if clientID == "" {
		return nil, ErrNoClientID
	}
	conn, err := dial()
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn, id: clientID}
	if err := c.handshake(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// socketPaths returns where Discord may be listening, in the order to try.
func socketPaths() []string {
	if runtime.GOOS == "windows" {
		var paths []string
		for i := range 10 {
			paths = append(paths, fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i))
		}
		return paths
	}
	var paths []string
	for _, base := range []string{
		os.Getenv("XDG_RUNTIME_DIR"),
		os.Getenv("TMPDIR"),
		os.Getenv("TMP"),
		os.Getenv("TEMP"),
		"/tmp",
	} {
		if base == "" {
			continue
		}
		for i := range 10 {
			paths = append(paths, filepath.Join(base, fmt.Sprintf("discord-ipc-%d", i)))
		}
	}
	// The sandboxed build ships its own socket directory.
	if dir := os.Getenv("SNAP"); dir != "" {
		for i := range 10 {
			paths = append(paths, filepath.Join(dir, "common", ".config", "discord",
				fmt.Sprintf("discord-ipc-%d", i)))
		}
	}
	return paths
}

// handshake sends the client id and waits for Discord to accept it.
func (c *Client) handshake() error {
	if err := c.write(opHandshake, map[string]any{"v": 1, "client_id": c.id}); err != nil {
		return err
	}
	// Discord answers with the ready frame, or with a close frame carrying the
	// reason. Either way the connection is settled by the time this returns.
	_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	op, body, err := c.read()
	if err != nil {
		return fmt.Errorf("discord: the handshake was not answered: %w", err)
	}
	if op == opClose {
		return fmt.Errorf("discord: the handshake was refused: %s", body)
	}
	return nil
}

// SetActivity replaces what the panel shows.
//
// An empty activity clears the panel. The payload is the one the protocol
// documents: activity.details is the track, activity.state is the artist, and
// the timestamps make Discord count the time played.
func (c *Client) SetActivity(activity Activity) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return ErrNoDiscord
	}
	return c.write(opFrame, map[string]any{
		"cmd": "SET_ACTIVITY",
		"args": map[string]any{
			"pid":      os.Getpid(),
			"activity": activity,
		},
		"nonce": fmt.Sprintf("%d", time.Now().UnixNano()),
	})
}

// Clear removes the panel.
func (c *Client) Clear() error { return c.SetActivity(Activity{}) }

// Close ends the connection. The panel goes with it, as it does when the app
// quits.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	_ = c.write(opClose, map[string]any{})
	err := c.conn.Close()
	c.conn = nil
	return err
}

// write sends one frame: a little-endian opcode and length, then the JSON body.
func (c *Client) write(op uint32, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	frame := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(frame[0:4], op)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(body)))
	copy(frame[8:], body)
	_, err = c.conn.Write(frame)
	return err
}

// read takes one frame.
func (c *Client) read() (op uint32, body []byte, err error) {
	var header [8]byte
	if _, err := io.ReadFull(c.conn, header[:]); err != nil {
		return 0, nil, err
	}
	op = binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])
	if length > maxFrame {
		return 0, nil, fmt.Errorf("discord: a frame of %d bytes is not a reply", length)
	}
	body = make([]byte, length)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return 0, nil, err
	}
	return op, body, nil
}

// Activity is what the panel shows, in the shape the protocol takes.
type Activity struct {
	// Details is the first line, which is the track.
	Details string `json:"details,omitempty"`
	// State is the second line, which is the artist.
	State string `json:"state,omitempty"`
	// Timestamps make Discord count the time played.
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	// Assets are the pictures: the large one is the artwork.
	Assets *Assets `json:"assets,omitempty"`
	// Type is how Discord frames the presence. Listening is 2.
	Type int `json:"type,omitempty"`
}

// Timestamps is when the track started, so Discord counts forward from it.
type Timestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// Assets is the pictures on the panel.
type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

