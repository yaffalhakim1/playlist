package src

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// A wrap is what a period of listening adds up to: the one fact worth leading
// with, and the few that are worth reading under it.
//
// It is computed rather than drawn, so the numbers can be checked without a
// window. Every field is a fact about the plays given, and a period with no
// plays gives an empty wrap rather than a set of zeroes dressed up as a story.
type wrap struct {
	// Plays is how many tracks started in the period.
	Plays int
	// Listened is how long they played for.
	Listened time.Duration
	// Artists and Tracks are how many different ones.
	Artists int
	Tracks  int

	// TopArtist and TopTrack are the most played, with their counts.
	TopArtist      string
	TopArtistPlays int
	TopTrack       string
	TopTrackPlays  int

	// BusiestDay is the weekday that played most, and BusiestHour the hour of
	// the day, from 0 to 23.
	BusiestDay  time.Weekday
	BusiestHour int
	// BusiestHourPlays is how many plays that hour holds.
	BusiestHourPlays int

	// Streak is the longest run of days in a row with a play, and Days the
	// number of days in the period that had one.
	Streak int
	Days   int

	// First and Last are the first and last play in the period.
	First time.Time
	Last  time.Time

	// TopArtists is the ranking, most played first.
	TopArtists []played
}

// Empty tells whether the period holds nothing to say.
func (w wrap) Empty() bool { return w.Plays == 0 }

// period is how far back a wrap looks.
type period int

const (
	// periodMonth is the last four weeks, which is the shape of a month
	// without arguing about which month.
	periodMonth period = iota
	// periodYear is the last twelve months.
	periodYear
	// periodAll is everything the app remembers.
	periodAll
)

// periodNames are what the segmented control shows.
var periodNames = []string{"4 weeks", "A year", "All time"}

// since returns the start of a period, or the zero time for all of it.
func (p period) since(now time.Time) time.Time {
	switch p {
	case periodMonth:
		return now.AddDate(0, 0, -28)
	case periodYear:
		return now.AddDate(-1, 0, 0)
	default:
		return time.Time{}
	}
}

// buildWrap adds up the plays in a period.
//
// The ties are broken by name, as everywhere else in the app, so a wrap does
// not reorder itself between frames when two things were played equally often.
func buildWrap(plays []libraryPlay, p period, now time.Time) wrap {
	since := p.since(now)
	var in []libraryPlay
	for _, play := range plays {
		if since.IsZero() || !play.At.Before(since) {
			in = append(in, play)
		}
	}
	if len(in) == 0 {
		return wrap{}
	}

	w := wrap{Plays: len(in)}
	w.First, w.Last = in[0].At, in[0].At

	trackCounts := map[string]int{}
	trackNames := map[string]string{}
	artistCounts := map[string]int{}
	dayCounts := map[time.Weekday]int{}
	hourCounts := map[int]int{}
	days := map[string]bool{}

	for _, play := range in {
		w.Listened += play.Listened
		if play.At.Before(w.First) {
			w.First = play.At
		}
		if play.At.After(w.Last) {
			w.Last = play.At
		}

		trackCounts[play.TrackID]++
		trackNames[play.TrackID] = play.Title

		artist := play.Artist
		if artist == "" {
			artist = "Unknown artist"
		}
		artistCounts[artist]++

		dayCounts[play.At.Weekday()]++
		hourCounts[play.At.Hour()]++
		days[play.At.Format("2006-01-02")] = true
	}

	w.Artists = len(artistCounts)
	w.Tracks = len(trackCounts)
	w.Days = len(days)

	// The top track: most plays, then name, so the answer is stable.
	tracks := make([]played, 0, len(trackCounts))
	for id, count := range trackCounts {
		tracks = append(tracks, played{title: trackNames[id], count: count, id: id})
	}
	sortPlayed(tracks)
	w.TopTrack, w.TopTrackPlays = tracks[0].title, tracks[0].count

	// The top artists: the same order, over the whole ranking.
	artists := make([]played, 0, len(artistCounts))
	for name, count := range artistCounts {
		artists = append(artists, played{title: name, count: count, id: name})
	}
	sortPlayed(artists)
	w.TopArtist, w.TopArtistPlays = artists[0].title, artists[0].count
	w.TopArtists = artists
	if len(w.TopArtists) > 5 {
		w.TopArtists = w.TopArtists[:5]
	}

	// The busiest day and hour, by the same rule.
	bestDay, bestDayCount := time.Sunday, -1
	for day := time.Sunday; day <= time.Saturday; day++ {
		if count := dayCounts[day]; count > bestDayCount {
			bestDay, bestDayCount = day, count
		}
	}
	w.BusiestDay = bestDay

	bestHour, bestHourCount := 0, -1
	for hour := range 24 {
		if count := hourCounts[hour]; count > bestHourCount {
			bestHour, bestHourCount = hour, count
		}
	}
	w.BusiestHour, w.BusiestHourPlays = bestHour, bestHourCount

	w.Streak = longestStreak(days)
	return w
}

// sortPlayed puts the most played first, and breaks a tie by name and then by
// id, so the order is total.
func sortPlayed(list []played) {
	slices.SortFunc(list, func(a, b played) int {
		if a.count != b.count {
			return cmp.Compare(b.count, a.count)
		}
		if a.title != b.title {
			return cmp.Compare(a.title, b.title)
		}
		return cmp.Compare(a.id, b.id)
	})
}

// longestStreak returns the longest run of consecutive days in a set.
func longestStreak(days map[string]bool) int {
	if len(days) == 0 {
		return 0
	}
	keys := make([]string, 0, len(days))
	for day := range days {
		keys = append(keys, day)
	}
	slices.Sort(keys)

	best, run := 1, 1
	previous, err := time.Parse("2006-01-02", keys[0])
	if err != nil {
		return 1
	}
	for _, key := range keys[1:] {
		day, err := time.Parse("2006-01-02", key)
		if err != nil {
			continue
		}
		if day.Sub(previous) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > best {
			best = run
		}
		previous = day
	}
	return best
}

// headline is the one sentence a period leads with: how much was heard, in the
// plainest words that fit.
func (w wrap) headline() string {
	if w.Empty() {
		return "Nothing played yet."
	}
	switch {
	case w.Listened >= time.Hour:
		hours := int(w.Listened.Hours())
		minutes := int(w.Listened.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%d %s of music.", hours, plural(hours, "hour"))
		}
		return fmt.Sprintf("%d %s %d minutes of music.", hours, plural(hours, "hour"), minutes)
	case w.Listened > 0:
		return fmt.Sprintf("%d %s of music.", int(w.Listened.Minutes()), plural(int(w.Listened.Minutes()), "minute"))
	default:
		// The lengths were never known, so the count is the story.
		return fmt.Sprintf("%d %s.", w.Plays, plural(w.Plays, "play"))
	}
}

// supporting is the line under the headline: the shape of the period, without
// repeating the total.
func (w wrap) supporting() string {
	if w.Empty() {
		return "The page fills in as you listen."
	}
	parts := []string{
		fmt.Sprintf("%d %s", w.Tracks, plural(w.Tracks, "track")),
		fmt.Sprintf("%d %s", w.Artists, plural(w.Artists, "artist")),
	}
	if w.Streak > 1 {
		parts = append(parts, fmt.Sprintf("%d days in a row", w.Streak))
	}
	return strings.Join(parts, " · ")
}

// hourWords says an hour of the day the way a person would: "9 at night",
// "3 in the afternoon".
func hourWords(hour int) string {
	switch {
	case hour == 0:
		return "midnight"
	case hour == 12:
		return "noon"
	case hour < 12:
		return fmt.Sprintf("%d in the morning", hour)
	case hour < 18:
		return fmt.Sprintf("%d in the afternoon", hour-hint12)
	default:
		return fmt.Sprintf("%d at night", hour-hint12)
	}
}

// hint12 is how far past noon the afternoon and the evening run.
const hint12 = 12

// plural adds an s when there is more than one.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
