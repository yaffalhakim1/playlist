// Package lastfm speaks to Last.fm's scrobbling API: it says what is playing,
// and adds a play to the user's profile once enough of it has been heard.
//
// The API is a form POST to one endpoint, signed with an MD5 of the parameters
// and the application's shared secret. Nothing here reaches for an SDK: the
// whole protocol is a handful of fields and one hash.
//
// The secret is not confidential. Whatever a desktop app can sign with, its
// users can read, so Last.fm's signature only ties calls to an application. The
// user's own session key is what grants access to their account, and it never
// leaves the machine except to Last.fm.
package lastfm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// apiRoot is the one endpoint of the REST API. It is a variable so a test can
// point the whole package at a stub server.
var apiRoot = "https://ws.audioscrobbler.com/2.0/"

// requestTimeout bounds one call. Scrobbling happens in the background, so a
// slow service must not hold anything up.
const requestTimeout = 30 * time.Second

// ErrInvalidSession reports that the session key is no longer good: the user
// revoked the application, or the key was never valid.
//
// It is separate from other failures because the app has to act on it: the only
// cure is to ask the user to connect again.
var ErrInvalidSession = errors.New("last.fm: the session is no longer valid")

// errNotApproved reports that the user has not approved the token yet, which is
// the ordinary case while they are still on Last.fm's page.
var errNotApproved = errors.New("last.fm: the token is not approved yet")

// errTransient reports a failure that may pass if tried again.
type errTransient struct{ err error }

func (e errTransient) Error() string { return e.err.Error() }
func (e errTransient) Unwrap() error { return e.err }

// Track is what is sent about a track.
type Track struct {
	// Artist is the performer, which Last.fm requires.
	Artist string
	// Title is the name of the track, which Last.fm requires.
	Title string
	// Album is the record it is from, when known.
	Album string
	// Duration is its length, which Last.fm uses to decide whether a play
	// counts.
	Duration time.Duration
	// Number is its place on the album, when known.
	Number int
}

// Valid reports whether Last.fm can be told about this track: it needs a title
// and an artist, and nothing else.
func (t Track) Valid() bool {
	return strings.TrimSpace(t.Title) != "" && strings.TrimSpace(t.Artist) != ""
}

// Scrobble is a play to submit: what was played, and when it began.
type Scrobble struct {
	Track   Track
	Started time.Time
	// Chosen says the user picked the track, rather than the queue moving on
	// to it.
	Chosen bool
}

// Session is an account's permission to scrobble.
type Session struct {
	Name string
	Key  string
}

// Client is the application's connection to Last.fm, before any account.
type Client struct {
	Key    string
	Secret string
	// Root is the API address; empty is apiRoot. It is a field so a test can
	// point the client at a stub server.
	Root string

	http     *http.Client
	endpoint string
}

// New returns a client for the application's key and secret.
func New(key, secret string) *Client {
	return &Client{Key: key, Secret: secret, http: &http.Client{Timeout: requestTimeout}, endpoint: apiRoot}
}

// Configured reports whether the application has a key and a secret, without
// which no call can be signed.
func (c *Client) Configured() bool {
	return c != nil && strings.TrimSpace(c.Key) != "" && strings.TrimSpace(c.Secret) != ""
}

// sign returns the api_sig of a call: the parameters sorted by name, written as
// name followed by value, then the secret, hashed with MD5.
//
// This is the whole of Last.fm's authentication. The format and callback
// parameters are left out because they are not part of the signature.
func sign(params map[string]string, secret string) string {
	names := make([]string, 0, len(params))
	for name := range params {
		if name == "format" || name == "callback" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		b.WriteString(name)
		b.WriteString(params[name])
	}
	b.WriteString(secret)
	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// call posts a signed call and decodes the answer into out, which may be nil.
func (c *Client) call(ctx context.Context, method string, params map[string]string, out any) error {
	if !c.Configured() {
		return errors.New("last.fm: this build has no application key")
	}
	if params == nil {
		params = map[string]string{}
	}
	params["method"] = method
	params["api_key"] = c.Key
	params["api_sig"] = sign(params, c.Secret)
	params["format"] = "json"

	form := url.Values{}
	for name, value := range params {
		form.Set(name, value)
	}
	root := c.endpoint
	if c.Root != "" {
		root = c.Root
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, root, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("last.fm: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		// A network failure may pass on its own.
		return errTransient{fmt.Errorf("last.fm: %w", err)}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return errTransient{fmt.Errorf("last.fm: reading the answer: %w", err)}
	}
	if resp.StatusCode != http.StatusOK {
		return errTransient{fmt.Errorf("last.fm: %s: %s", resp.Status, firstLine(body))}
	}

	// The API answers with an error object even on a 200.
	var envelope struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != 0 {
		return apiError(envelope.Error, envelope.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("last.fm: reading the answer: %w", err)
	}
	return nil
}

// apiError turns the API's error number into a Go error.
func apiError(code int, message string) error {
	switch code {
	case 4, 9:
		// 4: authentication failed. 9: invalid session key.
		return fmt.Errorf("%w: %s", ErrInvalidSession, message)
	case 10, 13, 26:
		// A bad key or signature is a configuration fault, not a transient
		// one, so it is never retried.
		return fmt.Errorf("last.fm: check the API key and secret: %s", message)
	case 14:
		// The token is not approved yet: the user is still on Last.fm's page.
		return errNotApproved
	case 11, 16, 29:
		return errTransient{fmt.Errorf("last.fm is busy: %s", message)}
	default:
		return fmt.Errorf("last.fm: %s", message)
	}
}

// InvalidSession reports whether err says the session key is no good, which
// only connecting again cures.
func InvalidSession(err error) bool { return errors.Is(err, ErrInvalidSession) }

// NotApproved reports whether err says the token has not been approved yet.
func NotApproved(err error) bool { return errors.Is(err, errNotApproved) }

// Transient reports whether err may pass if the call is tried again.
func Transient(err error) bool {
	var t errTransient
	return errors.As(err, &t)
}

// firstLine trims a body to its first line, for an error message.
func firstLine(body []byte) string {
	text := strings.TrimSpace(string(body))
	if line, _, ok := strings.Cut(text, "\n"); ok {
		return line
	}
	return text
}

// UpdateNowPlaying tells Last.fm what is playing now.
//
// The service uses the same fields as a scrobble, minus the time the track
// began: the now-playing call has no timestamp.
func (c *Client) UpdateNowPlaying(ctx context.Context, session string, track Track) error {
	params := trackParams(track, -1)
	params["sk"] = session
	return c.call(ctx, "track.updateNowPlaying", params, nil)
}

// Scrobble adds plays to the user's profile, in batches of up to MaxBatch.
//
// Each carries the moment its track began, which is what Last.fm files it
// under.
func (c *Client) Scrobble(ctx context.Context, session string, plays []Scrobble) error {
	if len(plays) == 0 {
		return nil
	}
	if len(plays) > MaxBatch {
		return fmt.Errorf("last.fm: %d plays in one call, at most %d", len(plays), MaxBatch)
	}
	params := map[string]string{"sk": session}
	for i, play := range plays {
		// The array notation is part of the signature, and its names must be
		// sorted as text: artist[10] comes before artist[2].
		for name, value := range trackParams(play.Track, i) {
			params[name] = value
		}
		params["timestamp["+strconv.Itoa(i)+"]"] = strconv.FormatInt(play.Started.UTC().Unix(), 10)
		// chosenByUser tells Last.fm the listener picked the track, rather
		// than a queue or a recommendation moving on to it.
		chosen := "0"
		if play.Chosen {
			chosen = "1"
		}
		params["chosenByUser["+strconv.Itoa(i)+"]"] = chosen
	}
	var answer struct {
		Scrobbles struct {
			Accepted string `json:"accepted"`
			Ignored  string `json:"ignored"`
		} `json:"scrobbles"`
	}
	return c.call(ctx, "track.scrobble", params, &answer)
}

// trackParams are the fields a track is sent as, at a place in a batch.
//
// A single call has no index and so no brackets; a batch numbers every field,
// because Last.fm pairs the fields by that number.
func trackParams(track Track, index int) map[string]string {
	suffix := ""
	if index >= 0 {
		suffix = "[" + strconv.Itoa(index) + "]"
	}
	params := map[string]string{
		"artist" + suffix: track.Artist,
		"track" + suffix:  track.Title,
	}
	if track.Album != "" {
		params["album"+suffix] = track.Album
	}
	if track.Duration > 0 {
		// The API takes whole seconds.
		params["duration"+suffix] = strconv.Itoa(int(track.Duration.Seconds()))
	}
	if track.Number > 0 {
		params["trackNumber"+suffix] = strconv.Itoa(track.Number)
	}
	return params
}

// Token asks for a request token, which the user authorizes in a browser.
func (c *Client) Token(ctx context.Context) (string, error) {
	var answer struct {
		Token string `json:"token"`
	}
	if err := c.call(ctx, "auth.getToken", nil, &answer); err != nil {
		return "", err
	}
	if answer.Token == "" {
		return "", errors.New("last.fm: no token came back")
	}
	return answer.Token, nil
}

// AuthURL is the page the user visits to let the application use their account.
func (c *Client) AuthURL(token string) string {
	return "https://www.last.fm/api/auth/?api_key=" + url.QueryEscape(c.Key) +
		"&token=" + url.QueryEscape(token)
}

// Session exchanges an authorized token for a session key.
//
// The token is spent by a successful call, so a caller that loses the answer
// must ask for a new token.
func (c *Client) Session(ctx context.Context, token string) (Session, error) {
	var answer struct {
		Session struct {
			Name string `json:"name"`
			Key  string `json:"key"`
		} `json:"session"`
	}
	if err := c.call(ctx, "auth.getSession", map[string]string{"token": token}, &answer); err != nil {
		return Session{}, err
	}
	if answer.Session.Key == "" {
		return Session{}, errors.New("last.fm: no session key came back")
	}
	return Session{Name: answer.Session.Name, Key: answer.Session.Key}, nil
}
