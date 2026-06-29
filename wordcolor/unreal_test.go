package wordcolor

import (
	"testing"

	"ccze-go/color"
	"github.com/stretchr/testify/assert"
)

func TestExtUnrealFullLine(t *testing.T) {
	p, ct := procWith(Extensions{Unreal: true})
	line := "[2026.06.28-13.42.12:957][  0]LogCoreRedirects: Verbose: hello /Script/X"
	out := render(p, line)

	// Text must be preserved exactly.
	assert.Equal(t, line, stripAnsi(out))

	// Each prefix field gets its color.
	assert.Contains(t, out, fgSeq(ct, color.PIDB), "brackets")
	assert.Contains(t, out, fgSeq(ct, color.Date), "date part of timestamp")
	assert.Contains(t, out, fgSeq(ct, color.GetTime), "time part of timestamp")
	assert.Contains(t, out, fgSeq(ct, color.Keyword), "log category")
	assert.Contains(t, out, fgSeq(ct, color.Debug), "Verbose verbosity")

	// With Unreal off, the prefix is not specially colored.
	p2, _ := procWith(Extensions{})
	assert.NotEqual(t, out, render(p2, line), "unreal flag must change coloring")
}

func TestExtUnrealNoTimestamp(t *testing.T) {
	// LogTimes=None style: category at the very start, no [time]/[frame].
	p, ct := procWith(Extensions{Unreal: true})
	line := "LogTemp: Warning: disk almost full"
	out := render(p, line)
	assert.Equal(t, line, stripAnsi(out))
	assert.Contains(t, out, fgSeq(ct, color.Keyword), "category colored without an anchor (Log* prefix)")
	assert.Contains(t, out, fgSeq(ct, color.Warning), "Warning verbosity colored")
}

func TestExtUnrealVerbosityColors(t *testing.T) {
	cases := map[string]color.Color{
		"Fatal":       color.Error,
		"Error":       color.Error,
		"Warning":     color.Warning,
		"Display":     color.GoodWord,
		"Log":         color.Debug,
		"Verbose":     color.Debug,
		"VeryVerbose": color.Debug,
	}
	for v, want := range cases {
		p, ct := procWith(Extensions{Unreal: true})
		line := "LogTemp: " + v + ": something happened"
		out := render(p, line)
		assert.Equal(t, line, stripAnsi(out), "preserve %q", line)
		assert.Contains(t, out, fgSeq(ct, want), "verbosity %q -> its color", v)
	}
}

func TestExtUnrealNotAVerbosity(t *testing.T) {
	// A "Word:" after the category that is NOT a known verbosity must be left
	// to the message body (not consumed/recolored as a verbosity).
	p, _ := procWith(Extensions{Unreal: true})
	line := "[  3]LogActor: SpawnActor failed: reason"
	out := render(p, line)
	assert.Equal(t, line, stripAnsi(out))
}

func TestExtUnrealGateRejectsNonUnreal(t *testing.T) {
	// Generic "word:" lines with no UE anchor and no Log* category must NOT be
	// claimed by the Unreal pre-pass (output identical to Unreal-off).
	lines := []string{
		"level: info here",
		"note: see above",
		"[2024-01-15 10:30:00] not unreal",
		"[abc] not a frame",
	}
	on, _ := procWith(Extensions{Unreal: true})
	off, _ := procWith(Extensions{})
	for _, l := range lines {
		assert.Equal(t, render(off, l), render(on, l), "non-unreal line must be untouched by -o unreal: %q", l)
	}
}

func TestMatchUnrealTimestamp(t *testing.T) {
	good := []string{
		"[2026.06.28-13.42.12:957]",
		"[2026.06.28-13.42.12:957][  0]rest",
		"[1.2.3-4.5.6:7]x",
	}
	for _, s := range good {
		assert.Positive(t, matchUnrealTimestamp(s), "want timestamp: %q", s)
	}
	bad := []string{
		"", "x", "[", "[]", "[2026.06.28]", "[2024-01-15 10:30:00]",
		"[  0]", "[2026.06.28-13.42.12]", "[2026/06/28-13:42:12:957]",
	}
	for _, s := range bad {
		assert.Zero(t, matchUnrealTimestamp(s), "want NOT timestamp: %q", s)
	}
}

func TestMatchUnrealFrame(t *testing.T) {
	assert.Equal(t, 5, matchUnrealFrame("[  0]rest"))
	assert.Equal(t, 5, matchUnrealFrame("[123]"))
	assert.Equal(t, 3, matchUnrealFrame("[7]"))
	for _, s := range []string{"", "[]", "[ ]", "[abc]", "x[0]", "[12"} {
		assert.Zero(t, matchUnrealFrame(s), "want NOT frame: %q", s)
	}
}
