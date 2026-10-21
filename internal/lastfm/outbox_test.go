package lastfm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The outbox is what makes scrobbling survive the network: a play that
// qualified while offline waits on disk and is sent when it can be. Losing one
// to a restart, or sending one twice, are the two failures worth testing.

func play(title string, started time.Time) Scrobble {
	return Scrobble{
		Track:   Track{Artist: "Artist", Title: title, Album: "Album", Duration: 3 * time.Minute},
		Started: started,
	}
}

// TestAPlaySurvivesARestart checks the whole point of the outbox: it is written
// to disk, and a new outbox over the same file has it.
func TestAPlaySurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	started := time.Unix(1792585800, 0)

	first := OpenOutbox(path)
	first.Add(play("One Time", started))
	first.Wait()

	second := OpenOutbox(path)
	if got := second.Len(); got != 1 {
		t.Fatalf("the reopened outbox holds %d plays, want 1", got)
	}
	batch := second.Batch(time.Now())
	if len(batch) != 1 {
		t.Fatalf("the batch holds %d plays, want 1", len(batch))
	}
	got := batch[0]
	if got.Track.Title != "One Time" || got.Track.Artist != "Artist" {
		t.Errorf("the play came back as %q by %q", got.Track.Title, got.Track.Artist)
	}
	if !got.Started.Equal(started) {
		t.Errorf("the play began at %v, want %v", got.Started, started)
	}
	if got.Track.Duration != 3*time.Minute {
		t.Errorf("the length came back as %v", got.Track.Duration)
	}
}

// TestDeliveredPlaysLeaveTheOutbox checks that a play taken from a batch is
// gone once it is done, and stays gone across a restart.
func TestDeliveredPlaysLeaveTheOutbox(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	now := time.Now()
	o := OpenOutbox(path)
	o.Add(play("One", now))
	o.Add(play("Two", now))
	o.Wait()

	batch := o.Batch(now)
	if len(batch) != 2 {
		t.Fatalf("the batch holds %d plays, want 2", len(batch))
	}
	o.Done(len(batch))
	o.Wait()

	if got := o.Len(); got != 0 {
		t.Errorf("the outbox holds %d plays after delivery", got)
	}
	if got := OpenOutbox(path).Len(); got != 0 {
		t.Errorf("the reopened outbox holds %d delivered plays", got)
	}
}

// TestABatchIsCapped checks that a batch never exceeds what Last.fm takes in one
// call.
func TestABatchIsCapped(t *testing.T) {
	o := OpenOutbox("")
	now := time.Now()
	for i := 0; i < MaxBatch+10; i++ {
		o.Add(play("Track", now))
	}
	if got := len(o.Batch(now)); got != MaxBatch {
		t.Errorf("the batch holds %d plays, want at most %d", got, MaxBatch)
	}
}

// TestPlaysTooOldAreDropped checks the age rule: Last.fm ignores scrobbles from
// more than two weeks ago, so sending them is a wasted call and a permanent
// backlog.
func TestPlaysTooOldAreDropped(t *testing.T) {
	o := OpenOutbox("")
	now := time.Now()
	o.Add(play("Ancient", now.Add(-MaxAge-time.Hour)))
	o.Add(play("Fresh", now))

	batch := o.Batch(now)
	if len(batch) != 1 {
		t.Fatalf("the batch holds %d plays, want only the fresh one", len(batch))
	}
	if batch[0].Track.Title != "Fresh" {
		t.Errorf("the batch holds %q, want the fresh play", batch[0].Track.Title)
	}
	if got := o.Len(); got != 1 {
		t.Errorf("the outbox still holds %d plays", got)
	}
}

// TestTheOldestGoFirst checks the order of a batch: the plays nearest to
// expiring are sent first, so an outbox that is always full still delivers the
// oldest rather than dropping them.
func TestTheOldestGoFirst(t *testing.T) {
	o := OpenOutbox("")
	now := time.Now()
	// Added newest first, to prove the order comes from the time and not from
	// the order they arrived in.
	o.Add(play("Newest", now))
	o.Add(play("Oldest", now.Add(-time.Hour)))

	batch := o.Batch(now)
	if len(batch) != 2 {
		t.Fatalf("the batch holds %d plays", len(batch))
	}
	if batch[0].Track.Title != "Oldest" {
		t.Errorf("the batch starts with %q, want the oldest play first",
			batch[0].Track.Title)
	}
}

// TestTheOutboxIsBounded checks that a machine offline for months does not grow
// a file without end: the oldest are let go.
func TestTheOutboxIsBounded(t *testing.T) {
	o := OpenOutbox("")
	now := time.Now()
	for i := 0; i < MaxPending+50; i++ {
		o.Add(play("Track", now))
	}
	if got := o.Len(); got != MaxPending {
		t.Errorf("the outbox holds %d plays, want at most %d", got, MaxPending)
	}
}

// TestClearDropsEverything checks the case of disconnecting: the plays waiting
// for one account must not be sent as another.
func TestClearDropsEverything(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	o := OpenOutbox(path)
	o.Add(play("One", time.Now()))
	o.Wait()
	o.Clear()
	o.Wait()
	if got := o.Len(); got != 0 {
		t.Errorf("the outbox holds %d plays after clearing", got)
	}
	if got := OpenOutbox(path).Len(); got != 0 {
		t.Errorf("the reopened outbox holds %d plays after clearing", got)
	}
}

// TestAnUnreadableOutboxIsEmpty checks that a corrupt file does not stop the app:
// the plays are lost, which is regrettable, but nothing crashes and nothing is
// sent twice.
func TestAnUnreadableOutboxIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	if err := os.WriteFile(path, []byte("this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	o := OpenOutbox(path)
	if got := o.Len(); got != 0 {
		t.Errorf("a corrupt outbox holds %d plays", got)
	}
	// And it can be used again.
	o.Add(play("One", time.Now()))
	o.Wait()
	if got := OpenOutbox(path).Len(); got != 1 {
		t.Errorf("the outbox was not rewritten: it holds %d plays", got)
	}
}

// TestTheOutboxFileIsPrivate checks that the file holding what was listened to
// is not world-readable.
func TestTheOutboxFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	o := OpenOutbox(path)
	o.Add(play("One", time.Now()))
	o.Wait()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows reports 0666 for files it does not model permissions on, so the
	// check is only meaningful where a mode is real.
	if mode := info.Mode().Perm(); mode&0o077 != 0 && mode != 0o666 {
		t.Errorf("the outbox file is %v, want no access for others", mode)
	}
}
