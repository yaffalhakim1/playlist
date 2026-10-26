package src

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// statsPeriod is the period the page shows, kept on the app so it survives a
// frame. It is an index into periodNames.
//
// It is not saved: a wrap is a thing you look at, not a setting, and reopening
// the app on "the last four weeks" is a better guess than reopening it on
// whatever was chosen last.
var statsPeriodIndex int

// statsPage draws what was listened to, as one poster rather than a dashboard.
//
// A page of listening is a story, so it leads with the one fact worth leading
// with — how much music, in plain words — and puts the few facts that earn their
// place under it. The chart stays, because a shape says something a total
// cannot, but it is the supporting evidence rather than the subject.
func (a *app) statsPage(c *ui.Context) {
	t := c.Theme()
	plays := a.plays()
	p := period(statsPeriodIndex)
	now := time.Now()
	w := buildWrap(plays, p, now)

	a.pageFrame(c, true, func() {
		ui.Column(c).FillWidth().Gap(t.Space(6)).Children(func() {
			ui.Row(c).FillWidth().Gap(t.Space(4)).AlignItems(ui.End).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(1)).Children(func() {
					pageName(c, "Your listening")
					caption(c, w.supporting())
				})
				ui.Segmented(c, &statsPeriodIndex, periodNames...).
					Width(t.Space(64)).Label("Period")
			})

			if w.Empty() {
				a.statsEmpty(c, p)
				return
			}

			a.statsPoster(c, w)
			a.statsFacts(c, w)
			values, labels := playsPerDay(plays, statsDays)
			a.statsChart(c, values, labels)
			a.statsMostPlayed(c, plays)
		})
	})
}

// statsPoster is the one big statement: how much was heard, and the shape of it
// underneath.
func (a *app) statsPoster(c *ui.Context, w wrap) {
	t := c.Theme()
	card(c, func() {
		ui.Column(c).FillWidth().Gap(t.Space(3)).Children(func() {
			ui.Text(c, w.headline()).
				FontSize(t.Rem(remDisplay)).FontWeight(600).
				LetterSpacing(trackDisplay(t)).LineHeight(leadDisplay).
				MaxLines(3)
			ui.Row(c).Gap(t.Space(6)).Wrap().Children(func() {
				a.posterFact(c, "Top artist", w.TopArtist, w.TopArtistPlays)
				a.posterFact(c, "Top track", w.TopTrack, w.TopTrackPlays)
			})
			// A strip of the period, one bar per day, so the poster carries its
			// own shape instead of pointing at the chart below it.
			a.posterStrip(c, w)
		})
	})
}

// posterFact is one of the two facts that sit under the headline.
func (a *app) posterFact(c *ui.Context, label, value string, count int) {
	t := c.Theme()
	ui.Column(c).Grow(1).MinWidth(t.Space(40)).Gap(t.Space(0.5)).Children(func() {
		caption(c, label)
		ui.Text(c, value).FontSize(t.Rem(remHeadline)).FontWeight(600).
			LineHeight(leadHeadline).MaxLines(1).Ellipsis("…")
		caption(c, playsWord(count))
	})
}

// posterStrip draws the plays of the period as a row of thin bars: the shape of
// the listening, in the poster rather than in a chart with axes.
func (a *app) posterStrip(c *ui.Context, w wrap) {
	s := surface(c)
	t := c.Theme()
	// The bars are the last four weeks whatever the period, so the strip is a
	// glance rather than a second chart.
	values, _ := playsPerDay(a.plays(), statsDays)
	top := highest(values)
	ui.Row(c).FillWidth().Height(t.Space(9)).Gap(t.Space(0.75)).
		AlignItems(ui.End).Children(func() {
		for _, value := range values {
			height := t.Space(1)
			if top > 0 && value > 0 {
				height = t.Space(1) + t.Space(8)*float32(value/top)
			}
			bar := ui.Box(c).Grow(1).Height(height).Radius(t.Space(0.5))
			if value > 0 {
				bar.Background(s.text())
			} else {
				bar.Background(s.hairline())
			}
		}
	})
}

// statsFacts is the row of three facts that earn their place: the streak, the
// busiest hour, and how many days were listened to at all.
func (a *app) statsFacts(c *ui.Context, w wrap) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(t.Space(4)).Children(func() {
		a.factTile(c, "In a row", fmt.Sprintf("%d", w.Streak), plural(w.Streak, "day"))
		a.factTile(c, "Most often", hourWords(w.BusiestHour), "of the day")
		a.factTile(c, "Days listened", fmt.Sprintf("%d", w.Days), "in this period")
	})
}

// factTile is one small fact: a number or a phrase, and what it counts.
func (a *app) factTile(c *ui.Context, label, value, note string) {
	s := surface(c)
	t := c.Theme()
	ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.5)).Padding(t.Space(4)).
		Radius(radiusControl(t)).Background(s.card()).Children(func() {
		caption(c, label)
		ui.Text(c, value).FontSize(t.Rem(remHeadline)).FontWeight(600).
			LineHeight(leadHeadline).MaxLines(1).Ellipsis("…")
		caption(c, note)
	})
}

// statsEmpty says what a period with nothing in it needs, and offers the way to
// a period that has something.
func (a *app) statsEmpty(c *ui.Context, p period) {
	s := surface(c)
	t := c.Theme()
	card(c, func() {
		ui.Column(c).FillWidth().Gap(t.Space(2)).Padding(t.Space(4), 0).
			Center().Children(func() {
			ui.Text(c, "Nothing in this period.").FontSize(t.Rem(remTitle)).
				FontWeight(600)
			ui.Text(c, "Listen for a while and it fills in.").
				FontSize(t.Rem(remBody)).TextColor(s.muted())
			if p != periodAll {
				if b := plainButton(c, "Show all time"); b.Clicked() {
					statsPeriodIndex = int(periodAll)
				}
			}
		})
	})
}
