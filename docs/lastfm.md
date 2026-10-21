# Last.fm

The app can send what you listen to your Last.fm profile: the track that is
playing now, and a scrobble once enough of it has been heard.

Nothing is sent until you ask for it. Scrobbling is off until an account is
connected, and connecting is a browser step you take yourself.

## Setting it up

Last.fm issues an **API key and a shared secret per application**, not per user.
A release build cannot carry one, because the key belongs to whoever publishes
the app, so you make your own:

1. Go to [last.fm/api/account/create](https://www.last.fm/api/account/create)
   and fill it in. The name and description are what you see when you approve
   the app; the callback URL can be left empty, because a desktop app does not
   use one.
2. Copy the **API key** and the **shared secret** it gives you.
3. In the app, open **Look**, find the **Last.fm** card, and paste both into
   the fields. Choose **Save**.
4. Choose **Connect**. Your browser opens Last.fm's own page, where you approve
   the app. You type your password there, on Last.fm's site, and the app never
   sees it.
5. The card then says **Connected as** your account name.

To stop: **Disconnect** forgets the account and deletes the key. **Scrobble
plays** pauses without disconnecting.

## When a play counts

Last.fm's rule, which the app follows:

- a track of **30 seconds or less** is never scrobbled;
- otherwise, once you have heard **half the track, or four minutes**, whichever
  comes first.

That is measured from the positions the player reports, not from the clock. A
pause, a buffering stall, or dragging the position forward does **not** count
towards it, so skipping to the end of a track does not scrobble it.

The time a play is filed under is when the track **began**, not when it
qualified. That is why a play can appear on your profile a minute or two before
the app sends it.

## When it cannot be sent

A play that qualified while the network was down is written to disk and sent
when it can be. Nothing is lost to a restart, a crash, or a night offline.

- Plays are sent oldest first, in batches of up to 50.
- A play more than **13 days** old is dropped: Last.fm ignores it.
- At most **2000** plays are kept, oldest dropped first.
- A batch Last.fm rejects is retried one play at a time, so a single bad play is
  dropped rather than holding the rest back.

The card shows how many plays are waiting.

## When Last.fm refuses the session

The card offers **Reconnect**. That happens when the session was revoked, or
the key file was deleted. Plays keep waiting on disk meanwhile, and are sent
after you reconnect.

## Where things are kept

| What | Where | Why there |
|---|---|---|
| API key and secret | the app's `settings.json` | not a credential: it identifies the app, not you |
| Account name, pause choice | `settings.json` | ordinary settings |
| **Session key** | `lastfm-session` beside the settings, readable only by you | it is a credential: it scrobbles as you until revoked |
| Plays waiting | `lastfm-pending.json` | survives a restart |

The session key is deliberately not in the settings, which are ordinary and may
be copied or backed up. **Disconnect** deletes it, and drops the plays that were
waiting for it, so they cannot be sent as whoever connects next.

## What is sent

| Call | When | Fields |
|---|---|---|
| `track.updateNowPlaying` | a track starts playing | artist, title, album, length |
| `track.scrobble` | the track passes the threshold | the same, plus when it began and whether you chose it |

Every call is signed: the parameters are sorted by name, written as name then
value, the secret is appended, and the whole is hashed with MD5. That is
Last.fm's own scheme, and `sign` in `internal/lastfm/lastfm.go` implements it.

The secret is not confidential in any meaningful sense: whatever a desktop app
can sign with, its users can read. The signature ties calls to an application;
your session key is what grants access to your account.

## The code

| Path | Responsibility |
|---|---|
| `internal/lastfm/lastfm.go` | the signed client: now playing, scrobbling, the sign-in |
| `internal/lastfm/listen.go` | the counting rule: when a play qualifies |
| `internal/lastfm/outbox.go` | the plays waiting, on disk |
| `internal/lastfm/session.go` | the session key, kept apart from the settings |
| `src/scrobble.go` | what the app does with them |
| `src/lastfm_page.go` | the Last.fm card in Look |

## The tests

```powershell
go test ./internal/lastfm/     # the client, the rule, the outbox
go test ./src/ -run 'TestAPlay|TestPausing|TestDisconnecting|TestTheSession'
```

The signature is checked against the worked example in Last.fm's own
authentication specification, so a change that breaks it fails a test rather
than a scrobble.
