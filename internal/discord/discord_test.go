package discord

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"
)

// The protocol is small enough to test without Discord: a fake client on a
// socket answers the handshake and records what the app sends it.

// fakeClient is a stand-in for Discord, listening on a socket.
type fakeClient struct {
	listener net.Listener
	// frames is what the app sent, in order.
	frames chan frame
}

type frame struct {
	op   uint32
	body map[string]any
}

// newFakeClient starts a client that answers a handshake and records frames.
func newFakeClient(t *testing.T, refuse bool) *fakeClient {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/discord-ipc-0"
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("unix sockets are not available here: %v", err)
	}
	f := &fakeClient{listener: listener, frames: make(chan frame, 32)}
	go f.serve(refuse)
	t.Cleanup(func() { listener.Close() })
	return f
}

// serve answers one connection.
func (f *fakeClient) serve(refuse bool) {
	conn, err := f.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		op, body, err := readFrame(conn)
		if err != nil {
			return
		}
		var parsed map[string]any
		_ = json.Unmarshal(body, &parsed)
		f.frames <- frame{op: op, body: parsed}

		switch op {
		case opHandshake:
			if refuse {
				writeFrame(conn, opClose, map[string]any{"code": 4000, "message": "Invalid Client ID"})
				return
			}
			writeFrame(conn, opFrame, map[string]any{"cmd": "DISPATCH", "evt": "READY"})
		case opClose:
			return
		}
	}
}

// readFrame takes one frame.
func readFrame(conn net.Conn) (uint32, []byte, error) {
	var header [8]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return 0, nil, err
	}
	length := binary.LittleEndian.Uint32(header[4:8])
	body := make([]byte, length)
	if _, err := io.ReadFull(conn, body); err != nil {
		return 0, nil, err
	}
	return binary.LittleEndian.Uint32(header[0:4]), body, nil
}

// writeFrame sends one frame.
func writeFrame(conn net.Conn, op uint32, payload any) {
	body, _ := json.Marshal(payload)
	frame := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(frame[0:4], op)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(body)))
	copy(frame[8:], body)
	conn.Write(frame)
}

// TestConnectWithoutAClientID checks the cheapest failure: an app with no
// application id does not try to reach Discord at all.
func TestConnectWithoutAClientID(t *testing.T) {
	client, err := Connect("")
	if err != ErrNoClientID {
		t.Errorf("Connect with no id gave %v, want ErrNoClientID", err)
	}
	if client != nil {
		t.Error("Connect with no id returned a client")
	}
}

// TestSetActivitySendsTheTrack checks the payload: the track is the first line,
// the artist the second, and the pictures are the artwork.
func TestSetActivitySendsTheTrack(t *testing.T) {
	fake := newFakeClient(t, false)
	client := &Client{conn: mustDial(t, fake), id: "123"}
	if err := client.handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	// The handshake's own frame is the first one.
	<-fake.frames

	start := time.Now().Unix()
	err := client.SetActivity(Activity{
		Details:    "It Will Rain",
		State:      "Bruno Mars",
		Timestamps: &Timestamps{Start: start, End: start + 222},
		Assets:     &Assets{LargeImage: "https://example.test/cover.jpg", LargeText: "Ballads"},
		Type:       2,
	})
	if err != nil {
		t.Fatalf("SetActivity: %v", err)
	}

	select {
	case got := <-fake.frames:
		if got.op != opFrame {
			t.Fatalf("the frame is opcode %d, want %d", got.op, opFrame)
		}
		if cmd, _ := got.body["cmd"].(string); cmd != "SET_ACTIVITY" {
			t.Errorf("the command is %q, want SET_ACTIVITY", cmd)
		}
		args, ok := got.body["args"].(map[string]any)
		if !ok {
			t.Fatalf("the frame has no args: %v", got.body)
		}
		activity, ok := args["activity"].(map[string]any)
		if !ok {
			t.Fatalf("the args have no activity: %v", args)
		}
		if activity["details"] != "It Will Rain" {
			t.Errorf("the track is %v, want It Will Rain", activity["details"])
		}
		if activity["state"] != "Bruno Mars" {
			t.Errorf("the artist is %v, want Bruno Mars", activity["state"])
		}
		if activity["type"] != float64(2) {
			t.Errorf("the type is %v, want 2 (listening)", activity["type"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no frame was sent")
	}
}

// TestClearSendsAnEmptyActivity checks that clearing the panel sends an activity
// with nothing in it, which is how Discord is told to take it down.
func TestClearSendsAnEmptyActivity(t *testing.T) {
	fake := newFakeClient(t, false)
	client := &Client{conn: mustDial(t, fake), id: "123"}
	if err := client.handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	<-fake.frames

	if err := client.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	select {
	case got := <-fake.frames:
		args := got.body["args"].(map[string]any)
		activity, ok := args["activity"].(map[string]any)
		if !ok {
			t.Fatalf("clearing sent no activity field: %v", args)
		}
		if len(activity) != 0 {
			t.Errorf("clearing sent %v, want an empty activity", activity)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("clearing sent no frame")
	}
}

// TestARefusedHandshakeIsAnError checks that a wrong client id comes back as an
// error rather than a connection that silently does nothing.
func TestARefusedHandshakeIsAnError(t *testing.T) {
	fake := newFakeClient(t, true)
	client := &Client{conn: mustDial(t, fake), id: "wrong"}
	err := client.handshake()
	if err == nil {
		t.Fatal("a refused handshake was not an error")
	}
}

// TestAFrameTooLargeIsRefused checks the guard: a pipe that answers with a
// nonsense length must not make the app allocate it.
func TestAFrameTooLargeIsRefused(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		var header [8]byte
		binary.LittleEndian.PutUint32(header[0:4], opFrame)
		binary.LittleEndian.PutUint32(header[4:8], maxFrame+1)
		server.Write(header[:])
	}()

	c := &Client{conn: client}
	if _, _, err := c.read(); err == nil {
		t.Error("a frame larger than the limit was read")
	}
}

// TestClosingTwiceIsHarmless checks that a client closed twice does not panic:
// the app closes on quit, and a failed connection may already be closed.
func TestClosingTwiceIsHarmless(t *testing.T) {
	fake := newFakeClient(t, false)
	client := &Client{conn: mustDial(t, fake), id: "123"}
	if err := client.handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	<-fake.frames

	if err := client.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

// mustDial connects to a fake client's socket.
func mustDial(t *testing.T, f *fakeClient) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("unix", f.listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial the fake client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
