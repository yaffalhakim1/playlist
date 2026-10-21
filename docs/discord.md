# Discord presence

The app can show what is playing on your Discord profile: the track, the artist,
and a counter of how long it has played.

![Where the switch lives](docs/screenshots/look.png)

## Turning it on

The switch is in **Look → Sharing → Discord**. It is off by default.

**A build with no application id cannot turn it on.** The panel is drawn by
Discord using the name and pictures of a *Discord application*, so one has to
exist before the presence can show anything. `src/presence.go` holds the id:

```go
const discordClientID = ""
```

To enable it:

1. Open the [Discord Developer Portal](https://discord.com/developers/applications)
   and create an application. Its name is what the panel will say.
2. Copy the **Application ID** from the General Information page.
3. Put it in `discordClientID`, and build.

Nothing else is needed: no bot, no token, no server. Rich Presence is drawn by
the Discord client on the user's own machine.

## Pictures

The large image is the track's artwork, which is a URL. Discord shows a URL as
the large image only for applications that have uploaded an asset under that
name, so as it stands the panel shows the track and the artist with no picture.

Two ways to change that, in order of effort:

- Upload the app's own mark as an asset named `playlist` in the developer portal,
  and use `mp:external/<url>` for the artwork. Discord renders external images
  that way for some builds, not all.
- Keep the text only. The panel reads correctly without a picture.

## What is sent

| Field | Value |
|---|---|
| `details` | the track's title |
| `state` | the artist, then the album on a second line |
| `timestamps.start` | when the track began |
| `timestamps.end` | when it ends, which is what draws the progress bar |
| `type` | `2`, which is Listening |
| `assets.large_image` | the artwork's URL |

**The progress bar needs both ends.** With `start` alone Discord shows a
counting timer and no bar; it can only draw the elapsed-against-total bar when
it knows where the track finishes.

**The album is a second line inside `state`.** The protocol has two text fields,
not three, but Discord draws a newline inside `state` as its own line. Line
breaks inside an artist or album name are flattened first, so a title cannot
push the panel into three lines.

A paused track sends the same fields without the timestamps, so the counter
stops.

## Why it says "Listening to Axolotl"

Discord writes that line itself: `Listening to <application name>`. The
application is called Axolotl in the developer portal, so that is the word.
Rename the application there to change it; no rebuild is needed.

The same is true of the small music icon beside it, which Discord draws for
activity type `2`. It is not an asset the app sends.

## When it fails

Discord is often not running. Every failure is logged once and the integration
stays off; the music never stops because of it. The presence tries again only
when the switch is turned off and on, so a machine without Discord does not open
a socket on every track.

## How it works

Discord listens on a named pipe on Windows (`\\.\pipe\discord-ipc-0` to `-9`)
and a Unix socket elsewhere. The protocol is a handshake with the application id,
then a frame per change, each framed by a little-endian opcode and length:

```
[opcode: uint32][length: uint32][json body]
```

`internal/discord` implements it with the standard library only. No SDK, no
dependency, and no network: the connection is to the desktop's own Discord.

## Tests

```powershell
go test ./internal/discord/
```

A fake client on a socket answers the handshake and records what the app sends,
so the protocol is tested without Discord running.
