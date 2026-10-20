package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runStream pipes input through processStream with the full default pipeline
// and returns the raw ANSI output.
func runStream(t *testing.T, input string, remfac bool) string {
	t.Helper()
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()
	registerAllPlugins(r, w, ct, wc, false)
	processStream(strings.NewReader(input), w, r, ct, wc, remfac, true, false)
	return buf.String()
}

func TestProcessStreamLongLine(t *testing.T) {
	// Regression: the bufio.Scanner-based loop had a 1MB cap on line length.
	long := strings.Repeat("x", 2*1024*1024)
	input := "before the long line\n" + long + "\nafter the long line\n"

	out := stripAnsiCompat(runStream(t, input, false))

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "before the long line", lines[0])
	assert.Equal(t, long, lines[1])
	assert.Equal(t, "after the long line", lines[2])
}

func TestProcessStreamCRLFAndPartialLastLine(t *testing.T) {
	// CRLF terminators are stripped like bufio.ScanLines (at most one \r before the \n).
	out := stripAnsiCompat(runStream(t, "alpha beta\r\nno trailing newline", false))
	assert.Equal(t, "alpha beta\nno trailing newline\n", out)
}

func TestProcessStreamEmptyLinesPreserved(t *testing.T) {
	out := stripAnsiCompat(runStream(t, "one\n\n\ntwo\n", false))
	assert.Equal(t, "one\n\n\ntwo\n", out)
}

func TestProcessStreamRemfac(t *testing.T) {
	out := stripAnsiCompat(runStream(t, "<13>plain message here\n", true))
	assert.Equal(t, "plain message here\n", out)
}

func TestConvertColorOverride(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"date=boldcyan", "date bold cyan"},
		{"error=red", "error red"},
		{"host=underlinegreen", "host underline green"},
		{"warning=yellow", "warning yellow"},
		{"noequals", "noequals"},
		{"date=boldredon_blue", "date bold red on_blue"},
		{"date=reversewhite", "date reverse white"},
		{"date=blinkmagenta", "date blink magenta"},
	}
	for _, tt := range tests {
		got := convertColorOverride(tt.input)
		assert.Equal(t, tt.want, got)

	}
}

func TestApplyOptions(t *testing.T) {
	// "modern" umbrella turns on the stable highlighters, not adaptive.
	var ext wordcolor.Extensions
	tr, wc, sl := true, true, true
	applyOptions("modern", &tr, &wc, &sl, &ext)
	assert.Equal(t, wordcolor.Extensions{Tags: true, Files: true, Slog: true, Durations: true, Unreal: true, Make: true}, ext)

	// Layering.
	ext = wordcolor.Extensions{}
	applyOptions("modern", &tr, &wc, &sl, &ext)          // baseline (env)
	applyOptions("noslog,adaptive", &tr, &wc, &sl, &ext) // override (-o)
	assert.True(t, ext.Tags)
	assert.True(t, ext.Files)
	assert.False(t, ext.Slog, "noslog must override modern")
	assert.True(t, ext.Durations)
	assert.True(t, ext.Adaptive)

	// Base flags and their no- variants, plus whitespace tolerance.
	ext = wordcolor.Extensions{}
	tr, wc, sl = true, true, true
	applyOptions("notransparent, nowordcolor , nolookups", &tr, &wc, &sl, &ext)
	assert.False(t, tr)
	assert.False(t, wc)
	assert.False(t, sl)

	// Empty string and unknown tokens are no-ops.
	ext = wordcolor.Extensions{Tags: true}
	applyOptions("", &tr, &wc, &sl, &ext)
	applyOptions("bogus,unknown", &tr, &wc, &sl, &ext)
	assert.Equal(t, wordcolor.Extensions{Tags: true}, ext)

	// nomodern clears the stable flags but leaves adaptive untouched.
	ext = wordcolor.Extensions{Tags: true, Files: true, Slog: true, Durations: true, Unreal: true, Make: true, Adaptive: true}
	applyOptions("nomodern", &tr, &wc, &sl, &ext)
	assert.Equal(t, wordcolor.Extensions{Adaptive: true}, ext)
}

func TestRegisterAllPlugins(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	registerAllPlugins(r, w, ct, wc, false)

	plugins := r.Plugins()
	assert.Equal(t, 20, len(plugins))

}

func TestFilterPlugins(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	registerAllPlugins(r, w, ct, wc, false)
	filterPlugins(r, []string{"syslog", "httpd"})

	plugins := r.Plugins()
	assert.Equal(t, 2, len(plugins))

}

func TestListAllPlugins(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	registerAllPlugins(r, w, ct, wc, false)

	// listAllPlugins writes to stdout; verify it doesn't panic We can't easily capture stdout in a test.
	listAllPlugins(r)
}
