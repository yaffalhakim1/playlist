// Command playlist is a native desktop player for YouTube playlists: paste a
// link to a playlist, and it plays the music of it.
//
// Everything lives in the src package; this file is only the entry point, so
// the root directory holds no logic of its own.
package main

import "youtube-playlist/src"

func main() {
	src.Run()
}
