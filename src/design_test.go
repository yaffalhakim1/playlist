package src

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The tests in this file are the design system's own guard rails: they do not
// check that a page draws, they check that it is drawn from the scale. A page
// that reaches past the scale is how a window starts looking assembled rather
// than designed, so each rule here fails loudly instead of being noticed late.

// TestTypeScaleIsOrdered checks that the sizes, the tracking and the leading
// form one scale: sizes grow, tracking turns negative as they do, and leading
// tightens as they grow.
func TestTypeScaleIsOrdered(t *testing.T) {
	sizes := []struct {
		name string
		rem  float32
	}{
		{"remCaption", remCaption},
		{"remBody", remBody},
		{"remTitle", remTitle},
		{"remHeadline", remHeadline},
		{"remLead", remLead},
		{"remDisplay", remDisplay},
	}
	for i := 1; i < len(sizes); i++ {
		if sizes[i].rem <= sizes[i-1].rem {
			t.Errorf("%s (%.2f) is not larger than %s (%.2f): the scale has a step out of order",
				sizes[i].name, sizes[i].rem, sizes[i-1].name, sizes[i-1].rem)
		}
	}

	leadings := []struct {
		name string
		mult float32
	}{
		{"leadDisplay", leadDisplay},
		{"leadHeadline", leadHeadline},
		{"leadTitle", leadTitle},
		{"leadBody", leadBody},
	}
	for i := 1; i < len(leadings); i++ {
		if leadings[i].mult <= leadings[i-1].mult {
			t.Errorf("%s (%.2f) is not looser than %s (%.2f): leading must follow size the other way",
				leadings[i].name, leadings[i].mult, leadings[i-1].name, leadings[i-1].mult)
		}
	}
	if leadings[0].mult < 1 {
		t.Errorf("leadDisplay (%.2f) is tighter than the glyphs are tall", leadings[0].mult)
	}
	// Body text wants a comfortable line, not the tightest one.
	if leadBody < 1.4 {
		t.Errorf("leadBody is %.2f: body text under 1.4 is hard to read in lines", leadBody)
	}
}

// TestNoTextIsTooSmall checks the floor of the scale: no role renders under 12
// DIPs, which is the smallest a desktop reads at arm's length.
func TestNoTextIsTooSmall(t *testing.T) {
	theme := themeInFrame(t)
	// The base the theme resolves at, after the desktop's own text scale.
	base := theme.FontSize
	if base <= 0 {
		t.Fatalf("the theme has no font size")
	}
	roles := []struct {
		name string
		rem  float32
	}{
		{"remCaption", remCaption},
		{"remBody", remBody},
		{"remTitle", remTitle},
		{"remHeadline", remHeadline},
		{"remLead", remLead},
		{"remDisplay", remDisplay},
	}
	for _, role := range roles {
		px := role.rem * base
		if px < 12 {
			t.Errorf("%s renders at %.1f DIPs, under the 12 DIP floor", role.name, px)
		}
	}
}

// TestHierarchyStepsAreWide checks that the steps which carry the hierarchy are
// wide enough to read as steps: a title, a group name, a lead and a page name
// each have to look different from the one below, or a page reads flat.
//
// The caption is not in the chain: it is metadata, and every system keeps it
// just under the body (Apple's SF runs 11, 12, 13, 15, 17). Spacing it a whole
// step away would make a track length look like a heading.
func TestHierarchyStepsAreWide(t *testing.T) {
	steps := []struct {
		smaller, larger string
		small, large    float32
	}{
		{"remBody", "remTitle", remBody, remTitle},
		{"remTitle", "remHeadline", remTitle, remHeadline},
		{"remHeadline", "remLead", remHeadline, remLead},
		{"remLead", "remDisplay", remLead, remDisplay},
	}
	for _, step := range steps {
		if ratio := step.large / step.small; ratio < 1.15 {
			t.Errorf("%s to %s is only %.2f:1, which reads as the same size",
				step.smaller, step.larger, ratio)
		}
	}
	// The caption stays close to the body, and under it.
	if remCaption >= remBody {
		t.Errorf("the caption (%.2f) is not smaller than the body (%.2f)", remCaption, remBody)
	}
	if ratio := remBody / remCaption; ratio > 1.25 {
		t.Errorf("the caption is %.2f:1 under the body: metadata should sit close to the text it labels",
			ratio)
	}
}

// TestTrackingFollowsSize checks the rule the eye notices first: large text is
// tracked tight, so a heading does not look loose.
func TestTrackingFollowsSize(t *testing.T) {
	theme := themeInFrame(t)
	display, headline := trackDisplay(theme), trackHeadline(theme)
	if display >= 0 {
		t.Errorf("a display is tracked %+.3f: large text wants negative tracking", display)
	}
	if headline >= 0 {
		t.Errorf("a headline is tracked %+.3f: it wants negative tracking too", headline)
	}
	if display >= headline {
		t.Errorf("tracking does not tighten as type grows: display %+.3f, headline %+.3f",
			display, headline)
	}
}

// TestRadiiAreMultiplesOfTheTheme checks that every corner is a share of the
// theme's radius, so a window that rounds more rounds all of it.
func TestRadiiAreMultiplesOfTheTheme(t *testing.T) {
	theme := themeInFrame(t)
	if radiusCard(theme) <= radiusControl(theme) {
		t.Errorf("a card (%.1f) is not rounder than a control (%.1f)",
			radiusCard(theme), radiusControl(theme))
	}
	if radiusPill(theme) <= radiusCard(theme) {
		t.Errorf("a pill (%.1f) is not rounder than a card (%.1f)",
			radiusPill(theme), radiusCard(theme))
	}
}

// themeInFrame returns the theme the app installs, as a frame sees it: the
// app's own theme function, run against a real context, not a copy.
func themeInFrame(t *testing.T) *ui.Theme {
	t.Helper()
	a := &app{art: newArtwork()}
	var theme *ui.Theme
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(a.theme(c))
		theme = c.Theme()
	}, 40, 40)
	tester.Frame()
	if theme == nil {
		t.Fatal("the frame installed no theme")
	}
	return theme
}

// TestPagesUseTheTypeScale parses the pages and fails on a size written by
// hand: every t.Rem argument is a role, never a number.
//
// This is the rule the redesign rests on. A page that writes t.Rem(1.55) gets
// a size that no other page shares, and the scale quietly stops being one.
func TestPagesUseTheTypeScale(t *testing.T) {
	for _, file := range sourceFiles(t) {
		if strings.HasSuffix(file, "theme.go") || strings.HasSuffix(file, "assets.go") {
			// The theme defines the scale; it is where the numbers live.
			continue
		}
		walkCalls(t, file, "Rem", func(t *testing.T, call *ast.CallExpr, where string) {
			if len(call.Args) != 1 {
				t.Errorf("%s: Rem takes one argument", where)
				return
			}
			name, ok := call.Args[0].(*ast.Ident)
			if !ok || !strings.HasPrefix(name.Name, "rem") {
				t.Errorf("%s: t.Rem(%s) is a size written by hand; use a rem role from theme.go",
					where, exprText(call.Args[0]))
			}
		})
	}
}

// TestPagesUseTheThemeColours parses the pages and fails on a colour written
// by hand: every face is a role of the theme.
//
// The exceptions are the theme, which defines the paper and its ink, the colour
// reader, which reads a cover, and the Appearance page, whose whole job is to show
// colours: the tones it offers, and a miniature of the appearance the window is
// not currently in.
func TestPagesUseTheThemeColours(t *testing.T) {
	allowed := map[string]bool{
		"theme.go":           true, // the paper and its ink are defined here
		"color.go":           true, // a cover's hue is read here
		"page_appearance.go": true, // the tones and the previews are drawn here
	}
	for _, file := range sourceFiles(t) {
		if allowed[filepath.Base(file)] {
			continue
		}
		for _, name := range []string{"Hex", "RGB", "RGBA"} {
			walkCalls(t, file, name, func(t *testing.T, call *ast.CallExpr, where string) {
				if isUIPackage(t, call) {
					t.Errorf("%s: ui.%s(%s) is a colour written by hand; take a role from surfaces.go",
						where, name, exprText(call.Args[0]))
				}
			})
		}
	}
}

// TestNoMethodLocksTwiceInOnePath is a structural guard against the class of
// bug that froze the window when a track started: a method that holds the
// app's lock calling another that takes it again.
//
// Go's mutex is not reentrant, so the second Lock never returns and the thread
// that draws waits on itself. A function that locks must call the *Locked form
// of anything that reads the state, or hand what it already knows to the
// callee.
//
// Only the function's own statements count: a lock taken inside a closure that
// runs later is a different moment, and a call passed to a function that runs
// it later is not a call yet.
func TestNoMethodLocksTwiceInOnePath(t *testing.T) {
	// The methods that take the app's lock. A function that already locks may
	// not call one of these at its own level.
	lockers := []string{
		"currentIndex", "currentTrack", "times", "times2", "durationOf",
		"isPlaying", "isShuffle", "repeatMode", "volumeLevel", "plays",
		"toggleShuffle", "setShuffle", "setRepeat", "cycleRepeat",
		"volumeBy", "update", "refresh",
	}

	for _, file := range sourceFiles(t) {
		for name, body := range functionBodies(t, file) {
			if !locksItself(body) {
				continue
			}
			for _, call := range appCalls(body) {
				if contains(lockers, call) {
					t.Errorf("%s holds the app's lock and calls a.%s(), which locks it again: the window would freeze",
						name, call)
				}
			}
		}
	}
}

// locksItself tells whether a function body takes the app's lock at its own
// level, rather than inside a closure it builds.
func locksItself(body *ast.BlockStmt) bool {
	found := false
	walkOwn(body, func(node ast.Node) {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isSelectorCall(call, "Lock") {
			return
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if inner, ok := selector.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "mu" {
				found = true
			}
		}
	})
	return found
}

// appCalls returns the names of the app's methods a function calls at its own
// level.
func appCalls(body *ast.BlockStmt) []string {
	var names []string
	walkOwn(body, func(node ast.Node) {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == "a" {
			names = append(names, selector.Sel.Name)
		}
	})
	return names
}

// walkOwn walks a function body, skipping the bodies of closures: what a
// closure does happens when it is called, not when it is built.
func walkOwn(body *ast.BlockStmt, visit func(ast.Node)) {
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		visit(node)
		return true
	})
}

// isSelectorCall tells whether a call names a method.
func isSelectorCall(call *ast.CallExpr, name string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == name
}

// functionBodies returns the body of each method of the app, by name.
func functionBodies(t *testing.T, file string) map[string]*ast.BlockStmt {
	t.Helper()
	source, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), file, source, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	out := map[string]*ast.BlockStmt{}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Body == nil {
			continue
		}
		out[fn.Name.Name] = fn.Body
	}
	return out
}

// contains tells whether a list holds a value.
func contains(list []string, want string) bool {
	return slices.Contains(list, want)
}
func pow(x, y float64) float64 { return math.Pow(x, y) }

// sourceFiles lists the package's own Go files, without the tests.
func sourceFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list the source: %v", err)
	}
	var out []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		out = append(out, file)
	}
	slices.Sort(out)
	return out
}

// walkCalls calls check for every call of a method named name in a file.
func walkCalls(t *testing.T, file, name string, check func(*testing.T, *ast.CallExpr, string)) {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return true
		}
		position := fset.Position(call.Pos())
		check(t, call, filepath.Base(position.Filename)+":"+strconv.Itoa(position.Line))
		return true
	})
}

// isUIPackage tells whether a call is made on the ui package.
func isUIPackage(t *testing.T, call *ast.CallExpr) bool {
	t.Helper()
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "ui"
}

// exprText returns an expression as it was written, for a failure message.
func exprText(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.BasicLit:
		return value.Value
	case *ast.SelectorExpr:
		return exprText(value.X) + "." + value.Sel.Name
	default:
		return "…"
	}
}
