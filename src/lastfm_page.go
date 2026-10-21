package src

import (
	"fmt"
	"strings"

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
		case !status.Configured && !a.lastfmSaved:
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

// lastfmCredentials draws the two fields the application's key and secret go in,
// and the button that checks and keeps them.
//
// They are the application's, not the user's: Last.fm issues them per
// application, so a build that ships without them cannot sign a call at all.
//
// Once they are kept the fields are gone. The secret is a credential that is
// now written down in the settings; leaving it on screen would let a screenshot
// or a passer-by take it, and it is not needed again to scrobble.
func (a *app) lastfmCredentials(c *ui.Context) {
	t := c.Theme()
	if a.lastfmSaved {
		setting(c, "Key saved", "The key and secret are kept on this computer.", func() {
			if button(c, "Replace").Clicked() {
				a.lastfmSaved = false
				a.lastfmNotice = ""
				a.update(func() {})
			}
		})
		return
	}
	if a.lastfmSaving {
		setting(c, "Checking the key", "Asking Last.fm whether it accepts it.", func() {
			components.Spinner(c, components.SpinnerRing)
		})
		return
	}
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
	setting(c, "Save", "Checked with Last.fm before it is kept.", func() {
		if button(c, "Save").Clicked() {
			a.saveLastFMCredentials()
		}
	})
	if a.lastfmNotice != "" {
		hairline(c)
		setting(c, "Last.fm", a.lastfmNotice, nil)
	}
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
//
// The key is checked against Last.fm before it is kept, because a key that is
// wrong is indistinguishable from one that works until the first scrobble
// fails, and by then the mistake is a day old. The check is a token request: it
// needs nothing but the key and the secret, and it changes nothing about the
// account.
func (a *app) saveLastFMCredentials() {
	if a.scrobble == nil {
		return
	}
	key, secret := strings.TrimSpace(a.lastfmKey), strings.TrimSpace(a.lastfmSecret)
	switch {
	case key == "" || secret == "":
		a.lastfmNotice = "Both the key and the secret are needed."
		a.update(func() {})
		return
	case len(key) != 32 || len(secret) != 32:
		// Last.fm's keys are 32 characters of hex, and saying so saves the
		// user a round trip to find out they pasted the wrong line.
		a.lastfmNotice = "A Last.fm key and secret are 32 characters each. Check what was pasted."
		a.update(func() {})
		return
	}
	a.scrobble.configure(key, secret)
	a.lastfmSaving = true
	a.lastfmNotice = ""
	a.update(func() {})

	go func() {
		err := a.scrobble.verify()
		a.update(func() {
			a.lastfmSaving = false
			if err != nil {
				a.scrobble.configure("", "")
				a.lastfmNotice = "Last.fm did not accept that key: " + err.Error()
				return
			}
			if a.store != nil {
				a.store.SetLastFMCredentials(key, secret)
			}
			a.lastfmNotice = "Saved. Choose Connect to sign in."
			// The secret leaves the page once it is kept: it is written down
			// in the settings, and there is no reason to keep it in a field
			// that a screenshot can catch.
			a.lastfmKey, a.lastfmSecret = "", ""
			a.lastfmSaved = true
		})
	}()
}

// button draws a button of the ordinary kind, at the size a settings row wants.
func button(c *ui.Context, label string) ui.Element {
	return components.Button(c, label, components.ButtonSecondary)
}
