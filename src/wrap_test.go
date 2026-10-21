package src

import (
	"testing"
	"time"
)

// A wrap is a set of facts about a period of listening. These tests check the
// facts, not the drawing: what a period adds up to is worth getting right before
// it is made to look good.

// playAt is a play of a track at a time.
func playAt(id, title, artist string, at time.Time, listened time.Duration) libraryPlay {
	return libraryPlay{TrackID: id, Title: title, Artist: artist, At: at, Listened: listened}
}

// aDay returns a time n days ago, at an hour.
func aDay(now time.Time, daysAgo, hour int) time.Time {
	day := now.AddDate(0, 0, -daysAgo)
	return time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, day.Location())
}

// TestAWrapOfNothingIsEmpty checks that no plays gives an empty wrap rather than
// a set of zeroes dressed as a story.
func TestAWrapOfNothingIsEmpty(t *testing.T) {
	w := buildWrap(nil, periodMonth, time.Now())
	if !w.Empty() {
		t.Error("a wrap of no plays is not empty")
	}
	if w.headline() != "Nothing played yet." {
		t.Errorf("the headline is %q", w.headline())
	}
}

// TestAWrapCountsThePeriod checks the totals.
func TestAWrapCountsThePeriod(t *testing.T) {
	now := time.Now()
	plays := []libraryPlay{
		playAt("a", "It Will Rain", "Bruno Mars", aDay(now, 1, 9), 3*time.Minute),
		playAt("b", "Roar", "Katy Perry", aDay(now, 1, 9), 3*time.Minute),
		playAt("a", "It Will Rain", "Bruno Mars", aDay(now, 2, 9), 3*time.Minute),
	}
	w := buildWrap(plays, periodMonth, now)

	if w.Plays != 3 {
		t.Errorf("the plays are %d, want 3", w.Plays)
	}
	if w.Tracks != 2 {
		t.Errorf("the tracks are %d, want 2", w.Tracks)
	}
	if w.Artists != 2 {
		t.Errorf("the artists are %d, want 2", w.Artists)
	}
	if want := 9 * time.Minute; w.Listened != want {
		t.Errorf("the listening time is %v, want %v", w.Listened, want)
	}
	if w.Days != 2 {
		t.Errorf("the days are %d, want 2", w.Days)
	}
}

// TestAWrapLeavesOutOlderPlays checks that the period is a period: a play from
// two months ago is not in the last four weeks.
func TestAWrapLeavesOutOlderPlays(t *testing.T) {
	now := time.Now()
	plays := []libraryPlay{
		playAt("old", "Old", "Someone", aDay(now, 60, 9), time.Minute),
		playAt("new", "New", "Someone", aDay(now, 2, 9), time.Minute),
	}

	month := buildWrap(plays, periodMonth, now)
	if month.Plays != 1 {
		t.Errorf("the month holds %d plays, want 1", month.Plays)
	}
	year := buildWrap(plays, periodYear, now)
	if year.Plays != 2 {
		t.Errorf("the year holds %d plays, want 2", year.Plays)
	}
	all := buildWrap(plays, periodAll, now)
	if all.Plays != 2 {
		t.Errorf("all time holds %d plays, want 2", all.Plays)
	}
}

// TestAWrapFindsTheTopTrackAndArtist checks the two facts worth leading with,
// and the tiebreak that keeps them stable.
func TestAWrapFindsTheTopTrackAndArtist(t *testing.T) {
	now := time.Now()
	plays := []libraryPlay{
		playAt("a", "It Will Rain", "Bruno Mars", aDay(now, 1, 9), time.Minute),
		playAt("a", "It Will Rain", "Bruno Mars", aDay(now, 2, 9), time.Minute),
		playAt("b", "Roar", "Katy Perry", aDay(now, 3, 9), time.Minute),
	}
	w := buildWrap(plays, periodMonth, now)

	if w.TopTrack != "It Will Rain" || w.TopTrackPlays != 2 {
		t.Errorf("the top track is %q (%d), want It Will Rain (2)", w.TopTrack, w.TopTrackPlays)
	}
	if w.TopArtist != "Bruno Mars" || w.TopArtistPlays != 2 {
		t.Errorf("the top artist is %q (%d), want Bruno Mars (2)", w.TopArtist, w.TopArtistPlays)
	}
}

// TestATieIsBrokenByName checks that two things played equally often come out in
// the same order every time: the page is drawn on every frame, so a tie broken
// by map order would flicker.
func TestATieIsBrokenByName(t *testing.T) {
	now := time.Now()
	plays := []libraryPlay{
		playAt("z", "Zebra", "Zed", aDay(now, 1, 9), time.Minute),
		playAt("a", "Apple", "Alice", aDay(now, 2, 9), time.Minute),
	}
	first := buildWrap(plays, periodMonth, now)
	for range 50 {
		w := buildWrap(plays, periodMonth, now)
		if w.TopTrack != first.TopTrack {
			t.Fatalf("the top track changed between frames: %q, then %q", first.TopTrack, w.TopTrack)
		}
		if w.TopArtist != first.TopArtist {
			t.Fatalf("the top artist changed between frames: %q, then %q", first.TopArtist, w.TopArtist)
		}
	}
	if first.TopTrack != "Apple" {
		t.Errorf("a tie came out as %q, want Apple (by name)", first.TopTrack)
	}
}

// TestAWrapFindsTheBusiestDayAndHour checks the shape of the listening: which
// day and which hour hold the most.
func TestAWrapFindsTheBusiestDayAndHour(t *testing.T) {
	now := time.Now()
	// Three plays on one day at 9, one on another day at 21.
	busy := aDay(now, 3, 9)
	plays := []libraryPlay{
		playAt("a", "A", "X", busy, time.Minute),
		playAt("b", "B", "X", busy, time.Minute),
		playAt("c", "C", "X", busy, time.Minute),
		playAt("d", "D", "X", aDay(now, 10, 21), time.Minute),
	}
	w := buildWrap(plays, periodMonth, now)

	if w.BusiestHour != 9 {
		t.Errorf("the busiest hour is %d, want 9", w.BusiestHour)
	}
	if w.BusiestHourPlays != 3 {
		t.Errorf("the busiest hour holds %d plays, want 3", w.BusiestHourPlays)
	}
	if w.BusiestDay != busy.Weekday() {
		t.Errorf("the busiest day is %v, want %v", w.BusiestDay, busy.Weekday())
	}
}

// TestAWrapFindsTheLongestStreak checks the run of days in a row, which is the
// one fact that rewards coming back.
func TestAWrapFindsTheLongestStreak(t *testing.T) {
	now := time.Now()
	plays := []libraryPlay{
		// Three in a row.
		playAt("a", "A", "X", aDay(now, 5, 9), time.Minute),
		playAt("a", "A", "X", aDay(now, 4, 9), time.Minute),
		playAt("a", "A", "X", aDay(now, 3, 9), time.Minute),
		// Then a gap, then one.
		playAt("a", "A", "X", aDay(now, 1, 9), time.Minute),
	}
	w := buildWrap(plays, periodMonth, now)

	if w.Streak != 3 {
		t.Errorf("the streak is %d, want 3", w.Streak)
	}
	if w.Days != 4 {
		t.Errorf("the days are %d, want 4", w.Days)
	}
}

// TestTheHeadlineSaysHowMuch checks the sentence a period leads with: hours when
// there are hours, minutes when there are only minutes, and a count when the
// lengths were never known.
func TestTheHeadlineSaysHowMuch(t *testing.T) {
	cases := []struct {
		name     string
		listened time.Duration
		plays    int
		want     string
	}{
		{"nothing known", 0, 7, "7 plays."},
		{"minutes", 25 * time.Minute, 3, "25 minutes of music."},
		{"one hour", time.Hour, 3, "1 hour of music."},
		{"hours and minutes", 2*time.Hour + 6*time.Minute, 9, "2 hours 6 minutes of music."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := wrap{Plays: tc.plays, Listened: tc.listened}
			if got := w.headline(); got != tc.want {
				t.Errorf("the headline is %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTheSupportingLineDoesNotRepeatTheTotal checks the line under the headline:
// it gives the shape of the period, not the number already said.
func TestTheSupportingLineDoesNotRepeatTheTotal(t *testing.T) {
	w := wrap{Plays: 40, Tracks: 12, Artists: 4, Streak: 6}
	got := w.supporting()
	if got != "12 tracks · 4 artists · 6 days in a row" {
		t.Errorf("the supporting line is %q", got)
	}
}

// TestTheHourReadsLikeATime checks that an hour of the day is written the way a
// person says it, not as a number.
func TestTheHourReadsLikeATime(t *testing.T) {
	cases := map[int]string{
		0:  "midnight",
		9:  "9 in the morning",
		12: "noon",
		15: "3 in the afternoon",
		21: "9 at night",
	}
	for hour, want := range cases {
		if got := hourWords(hour); got != want {
			t.Errorf("hourWords(%d) = %q, want %q", hour, got, want)
		}
	}
}

// TestThePeriodNamesMatchTheControl checks that the words the control shows are
// the ones the periods are built for.
func TestThePeriodNamesMatchTheControl(t *testing.T) {
	if len(periodNames) != 3 {
		t.Fatalf("there are %d period names, want 3", len(periodNames))
	}
	for i, p := range []period{periodMonth, periodYear, periodAll} {
		if p.since(time.Now()).IsZero() && p != periodAll {
			t.Errorf("%s has no start", periodNames[i])
		}
	}
}
