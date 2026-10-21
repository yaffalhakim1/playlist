# Documentation

| File | What it covers |
|---|---|
| [`bugs.md`](bugs.md) | every bug this app had: symptom, cause, fix, and the test that pins it. **Start here before changing playback or the release.** |
| [`performance.md`](performance.md) | what runs off the thread that draws, the buffers, the network timeouts, and what is *not* measured |
| [`playback.md`](playback.md) | how a track ends, and how a broken stream is recovered |
| [`mixes.md`](mixes.md) | starting a mix from a track: the id shape, and why it needs no account |
| [`lastfm.md`](lastfm.md) | scrobbling: the key, the threshold, the outbox, where the credential lives |
| [`discord.md`](discord.md) | the presence panel: both timestamps, the album line, what does not render |
| [`shipping.md`](shipping.md) | why a release is `mygo build` and never `go build` |

## Where the record lives

| | |
|---|---|
| **What changed, per version** | [`../CHANGELOG.md`](../CHANGELOG.md) |
| **Why a piece of code is the way it is** | the file's own comments, and these docs |
| **What a bug was** | [`bugs.md`](bugs.md) |

Every entry in `bugs.md` names a test. That is the point: a change that
reintroduces a bug fails somewhere, rather than being noticed weeks later by
someone listening to music.

## The rules these produced

Worth knowing before touching this code.

1. A method that holds `a.mu` may not call one that takes it again. Go's mutex is
   not reentrant, and the window freezes.
2. Do not decide the end of a track from `Playing()`. Use the place in the track.
3. A stream that breaks is not a track that finished. Recover it; do not skip.
4. Every spawn of a console program needs `proc.Hide`.
5. A release is `mygo build`, never `go build`.
6. A test may not reach the internet.
7. A fix needs a test that fails without it.
