package src

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

// The Last.fm group of the Look page.
//
// Nothing is sent to Last.fm before the user asks for it: the application's key
// has to be entered, and an account connected, and both are the user's choice.
// Connecting opens Last.fm's own page in the browser, so the app never sees the
// password.

// lastfmGroup draws the Last.fm card: the application key, the account, and the
// switch that pauses scrobbling.
func (a *app) lastfmGroup(c *ui.Context) {
	if a.scrobble == nil {
		return
	}
	status := a.scrobble.statusNow()

	group(c, "Last.fm", nil, func() {
		switch {
		case !status.Configured:
			setting(c, "Scrobbling",
				"Send the tracks you listen to, and what is playing now, to your Last.fm profile. "+
					"Last.fm issues an API key per application, so paste your own below.",
				nil)
			hairline(c)
			a.lastfmCredentials(c)

		case status.Busy:
			setting(c, "Waiting for approval",
				"Approve Playlist on the Last.fm page that opened in your browser. "+
					"It connects as soon as you do, and never sees your password.",
				func() {
					if button(c, "Cancel").Clicked() {
						a.scrobble.cancelConnect()
					}
				})

		case status.User == "":
			setting(c, "Scrobbling",
				"Send the tracks you listen to, and what is playing now, to your Last.fm profile.",
				func() {
					if button(c, "Connect").Clicked() {
						a.connectLastFM()
					}
				})
			hairline(c)
			a.lastfmCredentials(c)

		default:
			description := "Plays are sent after you have heard half the track, or four minutes."
			if status.Pending > 0 {
				description = fmt.Sprintf("%s %d waiting to be sent.", description, status.Pending)
			}
			setting(c, "Connected as "+status.User, description, func() {
				if button(c, "Disconnect").Clicked() {
					a.disconnectLastFM()
				}
			})
			hairline(c)
			setting(c, "Scrobble plays",
				"Turn off to pause scrobbling without disconnecting.", func() {
					a.switchSetting(c, &a.switchLastFM, "Scrobble plays")
				})
			if status.Reconnect {
				hairline(c)
				setting(c, "Connect again",
					"Last.fm no longer accepts this connection. Plays wait on this computer until you reconnect.",
					func() {
						if button(c, "Reconnect").Clicked() {
							a.connectLastFM()
						}
					})
			}
		}
		if status.Notice != "" {
			hairline(c)
			setting(c, "Last.fm", status.Notice, nil)
		}
	})

	// A switch flips the value it is handed as it is built, so the field is
	// read back once the frame has drawn.
	if status.Configured && status.User != "" && a.switchLastFM != !status.Paused {
		a.scrobble.setPaused(!a.switchLastFM)
	}
}

// lastfmCredentials draws the two fields the application's key and secret go in.
//
// They are the application's, not the user's: Last.fm issues them per
// application, so a build that ships without them cannot sign a call at all.
func (a *app) lastfmCredentials(c *ui.Context) {
	t := c.Theme()
	setting(c, "API key", "From last.fm/api/account/create.", func() {
		ui.Box(c).Width(t.Space(78)).Children(func() {
			components.TextBox(c, &a.lastfmKey, components.TextBoxOptions{})
		})
	})
	hairline(c)
	setting(c, "Shared secret", "The secret Last.fm shows beside the key.", func() {
		ui.Box(c).Width(t.Space(78)).Children(func() {
			components.TextBox(c, &a.lastfmSecret, components.TextBoxOptions{}).Password()
		})
	})
	hairline(c)
	setting(c, "Save", "The key and secret are kept in the app's settings on this computer.", func() {
		if button(c, "Save").Clicked() {
			a.saveLastFMCredentials()
		}
	})
}

// connectLastFM starts the browser sign-in.
func (a *app) connectLastFM() {
	if a.scrobble == nil {
		return
	}
	a.scrobble.connect(mygo.Shell.OpenExternal)
	a.update(func() {})
}

// disconnectLastFM forgets the account.
func (a *app) disconnectLastFM() {
	if a.scrobble == nil {
		return
	}
	a.scrobble.disconnect()
	a.update(func() {})
}

// saveLastFMCredentials keeps the application's key and secret.
func (a *app) saveLastFMCredentials() {
	if a.scrobble == nil {
		return
	}
	a.scrobble.configure(a.lastfmKey, a.lastfmSecret)
	if a.store != nil {
		a.store.SetLastFMCredentials(a.lastfmKey, a.lastfmSecret)
	}
	a.update(func() {})
}

// button draws a button of the ordinary kind, at the size a settings row wants.
func button(c *ui.Context, label string) ui.Element {
	return components.Button(c, label, components.ButtonSecondary)
}
