package src

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
	"github.com/oneo-me/onevra/source/components"
)

// statsDays is how many days the chart shows.
const statsDays = 14

// played is a track or an artist and how often it played.
type played struct {
	title string
	count int
	// id breaks a tie between two of the same name, so the order is total and
	// never depends on how a map happened to iterate.
	id string
}

// statsChart draws the plays of every day, on the accent, in a well of its own
// so the bars read against a quieter face than the card.
func (a *app) statsChart(c *ui.Context, values []float64, labels []string) {
	s := surface(c)
	t := c.Theme()
	plays := 0.0
	for _, value := range values {
		plays += value
	}
	group(c, "Plays per day", func() {
		caption(c, fmt.Sprintf("The last %d days", statsDays))
	}, func() {
		ui.Box(c).FillWidth().Height(t.Space(46)).Padding(t.Space(4)).
			Radius(radiusControl(t)).Background(s.sunken()).Children(func() {
			if plays == 0 {
				// A chart of nothing has no shape to read: it says what would
				// fill it instead of drawing an axis with no bars on it.
				ui.Box(c).Fill().Center().Children(func() {
					components.Text(c, "Play something, and the days fill in here.").
						FontSize(t.Rem(remBody)).TextColor(s.muted()).TextAlign(ui.Center)
				})
				return
			}
			x := chartBand(labels)
			// Plays are whole things: the scale marks whole numbers, so it
			// never asks the eye to read half a play.
			y := chartLinear(0, highest(values))
			drawPlot(c, s, x, y, values, labels)
		})
	})
}

// statsMostPlayed draws the tracks that played most, as a list of names with the
// count at the right.
func (a *app) statsMostPlayed(c *ui.Context, plays []libraryPlay) {
	s := surface(c)
	t := c.Theme()
	most := mostPlayed(plays, 6)
	if len(most) == 0 {
		return
	}
	group(c, "Most played", nil, func() {
		for i, item := range most {
			if i > 0 {
				hairline(c)
			}
			ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.Center).
				Padding(t.Space(1), 0).Children(func() {
				ui.Box(c).Size(t.Space(7), t.Space(7)).Shrink(0).Radius(radiusPill(t)).
					Background(s.sunken()).Center().Children(func() {
					components.Text(c, fmt.Sprintf("%d", i+1)).FontSize(t.Rem(remCaption)).
						FontWeight(500).TextColor(s.muted()).FontFeatures("tnum")
				})
				components.Text(c, item.title).FontSize(t.Rem(remBody)).
					FontWeight(500).MaxLines(1).Ellipsis("…").Grow(1)
				caption(c, playsWord(item.count))
			})
		}
	})
}

// mostPlayed returns the tracks played most, the first first.
//
// The order is total and stable: by count, most first, then by name, then by id.
// Without that tiebreak the order would follow Go's map iteration, which is
// randomized on every pass, and the page would shuffle itself while the user
// looked at it.
func mostPlayed(plays []libraryPlay, limit int) []played {
	counts := map[string]int{}
	names := map[string]string{}
	for _, play := range plays {
		counts[play.TrackID]++
		names[play.TrackID] = play.Title
	}
	list := make([]played, 0, len(counts))
	for id, count := range counts {
		list = append(list, played{title: names[id], count: count, id: id})
	}
	sortPlayed(list)
	if len(list) > limit {
		list = list[:limit]
	}
	return list
}

// playsWord says how often a track played, in words: one play is not "1 plays".
func playsWord(count int) string {
	if count == 1 {
		return "1 play"
	}
	return fmt.Sprintf("%d plays", count)
}

// playsPerDay counts the plays of every day, oldest first.
func playsPerDay(plays []libraryPlay, days int) ([]float64, []string) {
	now := time.Now()
	values := make([]float64, days)
	labels := make([]string, days)
	for i := range days {
		day := now.AddDate(0, 0, -(days - 1 - i))
		labels[i] = day.Format("Jan 2")
	}
	for _, play := range plays {
		for i := range days {
			day := now.AddDate(0, 0, -(days - 1 - i))
			if play.At.Year() == day.Year() && play.At.YearDay() == day.YearDay() {
				values[i]++
				break
			}
		}
	}
	return values, labels
}

// highest returns the largest of the values, and at least one.
func highest(values []float64) float64 {
	top := 0.0
	for _, value := range values {
		if value > top {
			top = value
		}
	}
	if top == 0 {
		return 1
	}
	return top
}

// countArtists returns how many different artists were played.
func countArtists(plays []libraryPlay) int {
	seen := map[string]bool{}
	for _, play := range plays {
		artist := play.Artist
		if artist == "" {
			artist = play.TrackID
		}
		seen[artist] = true
	}
	return len(seen)
}

// listeningTime returns how long the music played, in words.
func listeningTime(plays []libraryPlay) string {
	var total time.Duration
	for _, play := range plays {
		total += play.Listened
	}
	hours := int(total.Hours())
	minutes := int(total.Minutes()) % 60
	if hours == 0 {
		return fmt.Sprintf("%d min", minutes)
	}
	return fmt.Sprintf("%d h %d min", hours, minutes)
}

// chartBand and chartLinear are the chart's scales, kept here so the plot code
// reads without the chart package in every call.
func chartBand(labels []string) chart.Band  { return chart.NewBand(labels...).Padding(0.42, 0.12) }
func chartLinear(lo, hi float64) chart.Linear {
	y := chart.NewLinear(lo, hi).Nice(4)
	y.Format = func(v float64) string { return fmt.Sprintf("%.0f", v) }
	return y
}

// drawPlot draws the grid and the bars.
func drawPlot(c *ui.Context, s surfaces, x chart.Band, y chart.Linear, values []float64, labels []string) {
	t := c.Theme()
	chart.Plot(c, chart.Axes{
		Left:   y.Ticks(4),
		Bottom: x.Ticks(),
	}, func(area ui.Element) {
		// The lines behind the bars are the grid, in the faintest line the
		// theme has, so the bars stay the subject.
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range y.Ticks(4) {
				line := chart.Y(r, tick.At)
				p.Line(r.X, line, r.X+r.W, line, 1, s.hairline())
			}
		})
		for i, value := range values {
			if value == 0 {
				continue
			}
			chart.Rect(c, x.Band(i), y.Span(0, value)).
				Background(s.accent()).Radius(t.Space(0.75))
		}
	}).Height(t.Space(38)).Gap(t.Space(2))
}
