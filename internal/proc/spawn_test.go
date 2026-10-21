package proc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryChildIsStartedHidden checks that the app asks for the console to be
// hidden at every place it starts a console program.
//
// The helper being correct is not enough: the bug was that nothing called it.
// yt-dlp and ffmpeg are both console programs, and each one that starts without
// this puts a window beside the player. This reads the source so a new call site
// cannot forget it.
func TestEveryChildIsStartedHidden(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	// Every place the app starts a program, found by the call that does it.
	var spawns, hidden int
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "build", "dist", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(src)
		for i, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, "exec.Command") {
				continue
			}
			spawns++
			// The call to proc.Hide follows within a few lines: a comment
			// usually sits between them.
			lines := strings.Split(text, "\n")
			end := min(i+6, len(lines))
			if strings.Contains(strings.Join(lines[i:end], "\n"), "proc.Hide") {
				hidden++
			} else {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s:%d starts a program without proc.Hide: Windows "+
					"would open a console window for it", rel, i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if spawns == 0 {
		t.Fatal("found no exec.Command calls: the search is wrong")
	}
	if hidden != spawns {
		t.Errorf("%d of %d spawn sites hide the console", hidden, spawns)
	}
}
