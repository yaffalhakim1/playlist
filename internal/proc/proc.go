// Package proc starts the programs the app leans on without a window.
//
// yt-dlp and ffmpeg are console programs. A GUI app that starts one the plain
// way makes Windows open a console window for it, so the user sees a black
// rectangle flash beside the player every time a track begins. Hide is the one
// line that stops that, and the no-op it is elsewhere.
package proc

import "os/exec"

// Hide keeps the window of a child process from opening.
//
// Call it after building the command and before Start. It changes nothing
// about what the child reads or writes: its standard streams still work, which
// is how the app reads the stream list and the decoded samples.
func Hide(cmd *exec.Cmd) { hide(cmd) }
