package ffmpeg

import (
	"strings"
	"testing"
	"time"
)

// TestTheStreamIsAllowedToRecover checks the flags that keep a track playing.
//
// A stream from googlevideo drops its connection mid-song as a matter of
// course. Three things used to make that the end of the music:
//
//   - -xerror, which makes ffmpeg abort the whole process on the first
//     recoverable error rather than skipping the bad read;
//   - no reconnect, so a streamed HTTP source that went quiet was left to sit
//     on a dead socket while the place in the track stood still;
//   - nothing that told a clean end from a break.
func TestTheStreamIsAllowedToRecover(t *testing.T) {
	args := streamArgs("https://example.invalid/audio", 0)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "-xerror") {
		t.Error("-xerror aborts the process on a recoverable error: one dropped read ends the song")
	}
	for _, want := range []string{"-reconnect 1", "-reconnect_streamed 1", "-reconnect_on_network_error 1"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the stream is not told to reconnect: %q is missing", want)
		}
	}
}

// TestALocalFileIsNotGivenNetworkFlags checks the other side: a file on disk
// takes no reconnect options, and ffmpeg refuses them for some protocols.
func TestALocalFileIsNotGivenNetworkFlags(t *testing.T) {
	args := streamArgs(`C:\music\track.m4a`, 0)
	if joined := strings.Join(args, " "); strings.Contains(joined, "-reconnect") {
		t.Errorf("a local file was given network options: %s", joined)
	}
}

// TestTheOffsetComesBeforeTheInput checks that a seek is an input option, so
// ffmpeg starts reading at the offset rather than discarding what it read.
func TestTheOffsetComesBeforeTheInput(t *testing.T) {
	args := streamArgs("https://example.invalid/audio", 42*time.Second)
	seek, input := -1, -1
	for i, arg := range args {
		switch arg {
		case "-ss":
			seek = i
		case "-i":
			input = i
		}
	}
	if seek < 0 {
		t.Fatal("no offset was asked for")
	}
	if seek > input {
		t.Errorf("-ss comes after -i: ffmpeg would decode from the start and throw it away")
	}
}
