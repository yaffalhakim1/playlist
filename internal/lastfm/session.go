package lastfm

import (
	"errors"
	"os"
	"path/filepath"
)

// SessionStore keeps the account's session key in a file of its own, apart from
// the settings, readable only by the user.
//
// The key is a credential: it lets whoever holds it scrobble as this account
// until the user revokes the application. It is therefore not written among the
// settings, which are ordinary and may be copied about, and the file is created
// with the narrowest permissions the platform allows.
//
// The zero value keeps nothing: a store with no path is a store that has
// nowhere to write, which is what a test wants.
type SessionStore struct {
	Path string
}

// Load reads the session key, if one was saved.
func (s SessionStore) Load() (string, error) {
	if s.Path == "" {
		return "", nil
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Save writes the session key, replacing any earlier one.
func (s SessionStore) Save(key string) error {
	if s.Path == "" {
		return os.ErrNotExist
	}
	return writeFileAtomic(s.Path, []byte(key), 0o600)
}

// Delete forgets the session key.
func (s SessionStore) Delete() error {
	if s.Path == "" {
		return nil
	}
	if err := os.Remove(s.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// writeFileAtomic writes data to path so that a crash leaves either the old
// file or the new one, never half of each: it writes a temporary file beside
// the target, syncs it, and renames it over the target.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp) // harmless once the rename has happened
	if err := f.Chmod(perm); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
