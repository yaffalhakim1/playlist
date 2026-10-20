// Package ffmpeg decodes the audio of a stream into samples for the player.
//
// ffmpeg does the decoding: it reads the stream itself and writes signed
// 16-bit PCM at 48 kHz in two channels, which is what MyGo's audio package
// plays.
package ffmpeg

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// sampleRate is the rate the audio plays at.
	sampleRate = 48000
	// channels is how many channels of audio there are.
	channels = 2
	// bytesPerSample is the size of one sample of 16-bit PCM.
	bytesPerSample = 2
	// readTimeout bounds one read of the stream by ffmpeg.
	readTimeout = 15 * time.Second
	// stopTimeout bounds the wait for ffmpeg to die.
	stopTimeout = 2 * time.Second
	// userAgent is the browser the web app pretends to be, for the streams
	// ffmpeg reads.
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)

// ErrToolMissing reports that ffmpeg is not installed.
type ErrToolMissing struct {
	Err error
}

func (e *ErrToolMissing) Error() string {
	return fmt.Sprintf("ffmpeg is not installed, so playback cannot work: %v", e.Err)
}

func (e *ErrToolMissing) Unwrap() error { return e.Err }

// LookTool finds ffmpeg, and says so when it is missing.
func LookTool() (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", &ErrToolMissing{Err: err}
	}
	return path, nil
}

// PCM decodes a stream of audio into raw samples, for MyGo's audio package.
//
// ffmpeg does the decoding: it reads the stream itself and writes signed
// 16-bit PCM at 48 kHz in two channels, which is what the player wants.
type PCM struct {
	url      string
	duration time.Duration
	ctx      context.Context

	mu    sync.Mutex
	cmd   *exec.Cmd
	out   io.ReadCloser
	wait  chan error
	raw   []byte // bytes read but not decoded yet
	frame int64  // frames decoded so far
	done  bool
	chunk []byte
}

// NewPCM starts ffmpeg on a stream and returns the decoder for it.
//
// The duration is the length of the track, which the player shows and seeks
// within; it may be zero when it is unknown.
func NewPCM(ctx context.Context, url string, duration time.Duration) (*PCM, error) {
	if _, err := LookTool(); err != nil {
		return nil, err
	}
	p := &PCM{url: url, duration: duration, ctx: ctx}
	if err := p.start(0); err != nil {
		return nil, err
	}
	return p, nil
}

// SampleRate returns the frames per second of the stream.
func (p *PCM) SampleRate() int { return sampleRate }

// Channels returns the samples per frame.
func (p *PCM) Channels() int { return channels }

// Len returns the frames from the start to the end, or -1 when the length of
// the track is unknown.
func (p *PCM) Len() int64 {
	if p.duration <= 0 {
		return -1
	}
	return int64(p.duration.Seconds() * sampleRate)
}

// Read decodes the next samples into buf.
func (p *PCM) Read(buf []float32) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(buf) == 0 {
		return 0, nil
	}
	frames := len(buf) / channels
	if frames == 0 {
		return 0, nil
	}
	want := frames * channels * bytesPerSample
	err := p.fill(want)

	n := p.decode(buf)
	p.frame += int64(n / channels)
	switch {
	case n > 0 && err != nil:
		return n, err
	case n > 0:
		return n, nil
	case err != nil:
		return 0, err
	default:
		return 0, io.EOF
	}
}

// SetPosition moves to a frame, counted from the start.
//
// ffmpeg cannot seek in the middle of a stream it is writing, so it is
// started again at the offset.
func (p *PCM) SetPosition(frame int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return errors.New("stream: the decoder is closed")
	}
	at := time.Duration(float64(frame) / sampleRate * float64(time.Second))
	p.stop()
	if err := p.start(at); err != nil {
		return err
	}
	p.frame = frame
	return nil
}

// Close stops ffmpeg.
func (p *PCM) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done = true
	p.stop()
	return nil
}

// fill reads bytes until there are enough of them, or the stream ends.
func (p *PCM) fill(want int) error {
	if p.chunk == nil {
		p.chunk = make([]byte, 32*1024)
	}
	for len(p.raw) < want {
		if p.out == nil {
			return io.EOF
		}
		n, err := p.out.Read(p.chunk)
		if n > 0 {
			p.raw = append(p.raw, p.chunk[:n]...)
		}
		if err != nil {
			p.out = nil
			if errors.Is(err, io.EOF) {
				return io.EOF
			}
			return err
		}
	}
	return nil
}

// decode turns the bytes read into samples.
func (p *PCM) decode(buf []float32) int {
	frames := len(p.raw) / (channels * bytesPerSample)
	if frames > len(buf)/channels {
		frames = len(buf) / channels
	}
	for i := 0; i < frames*channels; i++ {
		raw := int16(binary.LittleEndian.Uint16(p.raw[i*2:]))
		buf[i] = float32(raw) / 32768
	}
	used := frames * channels * bytesPerSample
	p.raw = p.raw[used:]
	return frames * channels
}

// start runs ffmpeg on the stream, at an offset.
func (p *PCM) start(at time.Duration) error {
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-xerror"}
	if isHTTP(p.url) {
		args = append(args,
			"-protocol_whitelist", "http,https,httpproxy,tcp,tls,crypto",
			"-rw_timeout", strconv.FormatInt(readTimeout.Microseconds(), 10),
			"-user_agent", userAgent,
		)
	}
	if at > 0 {
		args = append(args, "-ss", strconv.FormatFloat(at.Seconds(), 'f', 3, 64))
	}
	args = append(args,
		"-i", p.url,
		"-vn",
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ac", strconv.Itoa(channels),
		"-ar", strconv.Itoa(sampleRate),
		"pipe:1",
	)

	cmd := exec.CommandContext(p.ctx, "ffmpeg", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stream: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("stream: %w", err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	p.cmd = cmd
	p.out = out
	p.wait = waited
	return nil
}

// stop ends the run of ffmpeg.
//
// The wait is bounded: a process that does not die must not keep the player
// waiting.
func (p *PCM) stop() {
	if p.cmd != nil {
		_ = p.cmd.Process.Kill()
	}
	if p.out != nil {
		_ = p.out.Close()
		p.out = nil
	}
	if p.wait != nil {
		select {
		case <-p.wait:
		case <-time.After(stopTimeout):
		}
		p.wait = nil
	}
	p.cmd = nil
	p.raw = nil
}

// isHTTP tells an address that needs the network options of ffmpeg from a
// file on disk, which does not take them.
func isHTTP(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
