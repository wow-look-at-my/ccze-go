package wordcolor

import (
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
	"github.com/stretchr/testify/assert"
)

// procWith returns a processor with the given extensions enabled.
func procWith(e Extensions) (*Processor, *color.Table) {
	ct := color.NewTable(true)
	p := New(ct)
	p.SetExtensions(e)
	return p, ct
}

// render runs a full message through Process and returns the raw output.
func render(p *Processor, msg string) string {
	var buf bytes.Buffer
	p.Process(&buf, msg, true, false)
	return buf.String()
}

// ---------------------------------------------------------------------------
// matchDuration
// ---------------------------------------------------------------------------

func TestMatchDuration(t *testing.T) {
	good := []string{
		"30.2s", "1m30s", "100ms", "1.5h", "500us", "300µs", "2h45m",
		"1ns", "0s", "-5m", "+1h", "2h45m30s", "1m0.5s", "10m", "3h",
	}
	for _, s := range good {
		assert.True(t, matchDuration(s), "expected duration: %q", s)
	}
	bad := []string{
		"", "5", "12:30", "abc", "5x", "30.2", "s", "m", "ms", "-", "+",
		"5gb", "0x1f", "1.2.3", "5d", "100", "v1.2", "12:30:00",
	}
	for _, s := range bad {
		assert.False(t, matchDuration(s), "expected NOT duration: %q", s)
	}
}

func TestExtDurations(t *testing.T) {
	p, ct := procWith(Extensions{Durations: true})
	out := render(p, "took 30.2s and 100ms")
	// GetTime is bold (1m) + the GetTime color. Just assert the token survived
	// and that a non-default color escape surrounds it.
	assert.Contains(t, stripAnsi(out), "30.2s")
	assert.Contains(t, stripAnsi(out), "100ms")
	// With durations OFF, 30.2s should not be GetTime-colored.
	p2, _ := procWith(Extensions{})
	out2 := render(p2, "took 30.2s and 100ms")
	assert.NotEqual(t, out, out2, "durations flag should change coloring")
	_ = ct
}

// ---------------------------------------------------------------------------
// Tags
// ---------------------------------------------------------------------------

func TestExtTags(t *testing.T) {
	p, _ := procWith(Extensions{Tags: true})

	cases := map[string]color.Color{
		"[ERROR]": color.Error,
		"[WARN]":  color.Warning,
		"[INFO]":  color.GoodWord,
		"[DEBUG]": color.Debug,
		"[auth]":  color.Keyword,
	}
	ct := color.NewTable(true)
	for tag, want := range cases {
		out := render(p, tag)
		// The inner text must be present and the bracket preserved.
		inner := strings.Trim(tag, "[]")
		assert.Equal(t, tag, stripAnsi(out), "tag text must be preserved: %q", tag)
		// The inner color escape (foreground) must appear in the output.
		var want_buf bytes.Buffer
		ct.WriteColored(&want_buf, want, inner)
		// Extract the fg sequence used for `want` and assert it is in out.
		assert.Contains(t, out, fgSeq(ct, want), "tag %q should use its level color", tag)
	}
}

func TestExtTagsWithPunctuation(t *testing.T) {
	p, _ := procWith(Extensions{Tags: true})
	// Surrounding punctuation must be preserved exactly.
	for _, in := range []string{"[INFO]:", "([INFO])", "[INFO],", "prefix[INFO]"} {
		out := render(p, in)
		assert.Equal(t, in, stripAnsi(out), "must preserve %q", in)
	}
}

// ---------------------------------------------------------------------------
// Counters ([N/M])
// ---------------------------------------------------------------------------

func TestExtCounter(t *testing.T) {
	p, ct := procWith(Extensions{Tags: true})

	numFg := fgSeq(ct, color.Numbers)
	brkFg := fgSeq(ct, color.PIDB)

	// [22/43]: digits get the numbers color, the [ / ] glyphs the bracket color.
	out := render(p, "[22/43]")
	assert.Equal(t, "[22/43]", stripAnsi(out), "counter text must be preserved")
	assert.Contains(t, out, numFg, "counter digits should use the numbers color")
	assert.Contains(t, out, brkFg, "counter glyphs should use the bracket color")
	// Both digit runs and all three glyphs are individually colored: the
	// numbers fg appears twice (22 and 43) and the bracket fg three times
	// ([, /, ]).
	assert.Equal(t, 2, strings.Count(out, numFg), "both digit runs should be numbers-colored")
	assert.Equal(t, 3, strings.Count(out, brkFg), "[, / and ] should each be bracket-colored")

	// Byte-for-byte invariant for the canonical case.
	assert.Equal(t, "[22/43]", stripAnsi(render(p, "[22/43]")))

	// Different digit widths.
	out2 := render(p, "[1/10]")
	assert.Equal(t, "[1/10]", stripAnsi(out2))
	assert.Contains(t, out2, numFg)
	assert.Contains(t, out2, brkFg)

	// Trailing punctuation is preserved.
	assert.Equal(t, "[22/43]:", stripAnsi(render(p, "[22/43]:")))

	// Non-counter brackets still render as TAGS, not counters. [INFO] keeps its
	// level color; the multi-slash and empty-half cases fall through to the tag
	// path (exercising the ok==false branches of counterParts).
	infoOut := render(p, "[INFO]")
	assert.Equal(t, "[INFO]", stripAnsi(infoOut))
	assert.Contains(t, infoOut, fgSeq(ct, color.GoodWord), "[INFO] should stay a level tag")
	for _, in := range []string{"[22/43/99]", "[22/]", "[/43]", "[a/b]"} {
		out := render(p, in)
		assert.Equal(t, in, stripAnsi(out), "non-counter bracket must be preserved: %q", in)
		// These are tags, not counters: the inner content (digits/slashes) is one
		// keyword-colored run, so the numbers color must NOT appear.
		assert.NotContains(t, out, numFg, "%q should render as a tag, not a counter", in)
	}
}

func TestCounterParts(t *testing.T) {
	// Positive case.
	a, b, ok := counterParts("22/43")
	assert.True(t, ok)
	assert.Equal(t, "22", a)
	assert.Equal(t, "43", b)

	// Negative cases — exercise every false branch of counterParts.
	for _, in := range []string{"22/", "/43", "a/b", "22/4/3", "2243", "/", ""} {
		_, _, ok := counterParts(in)
		assert.False(t, ok, "counterParts(%q) should be false", in)
	}
}

// fgSeq returns the foreground SGR sequence WriteColored emits for a color,
// so tests can assert a specific color was applied.
func fgSeq(ct *color.Table, c color.Color) string {
	var buf bytes.Buffer
	ct.WriteColored(&buf, c, "\x00")
	s := buf.String()
	idx := strings.IndexByte(s, 0)
	// The escape immediately before the sentinel byte is the fg color set.
	prefix := s[:idx]
	last := strings.LastIndex(prefix, "\x1b[")
	return prefix[last:]
}

// ---------------------------------------------------------------------------
// slog / logfmt
// ---------------------------------------------------------------------------

func TestExtSlog(t *testing.T) {
	p, ct := procWith(Extensions{Slog: true, Durations: true})
	out := render(p, "level=info user=alice latency=30.2s")
	stripped := stripAnsi(out)
	assert.Equal(t, "level=info user=alice latency=30.2s", stripped)
	// Keys should be Field-colored.
	assert.Contains(t, out, fgSeq(ct, color.Field))
	// The duration value should be GetTime-colored (value classification).
	assert.Contains(t, out, fgSeq(ct, color.GetTime))
}

func TestExtSlogQuotedAndGuards(t *testing.T) {
	p, _ := procWith(Extensions{Slog: true})
	cases := []string{
		`msg="hello"`,
		`path='/var/log'`,
		`error=`,
		`a==b`,        // equality, not kv (== guard)
		`dGVzdA==`,    // base64 padding, not kv
		`--flag=x`,    // CLI flag, not slog key
		`123=456`,     // numeric key, not slog
		`key=value=2`, // only first = splits
	}
	for _, in := range cases {
		out := render(p, in)
		assert.Equal(t, in, stripAnsi(out), "slog must preserve %q", in)
	}
}

// TestExtSlogValueTypes checks that slog values are classified by type, and
// that a bare-filename value prefers File over the hostname syntax it also
// matches (only when the Files extension is also enabled).
func TestExtSlogValueTypes(t *testing.T) {
	p, ct := procWith(Extensions{Slog: true, Files: true, Durations: true})
	out := render(p, "file=main.go host=example.com dur=1m30s")
	assert.Equal(t, "file=main.go host=example.com dur=1m30s", stripAnsi(out))
	assert.Contains(t, out, fgSeq(ct, color.File), "file=main.go value should be File")
	assert.Contains(t, out, fgSeq(ct, color.Host), "host=example.com value should be Host")
	assert.Contains(t, out, fgSeq(ct, color.GetTime), "dur=1m30s value should be a duration")
}

// ---------------------------------------------------------------------------
// Files / paths
// ---------------------------------------------------------------------------

func TestExtFiles(t *testing.T) {
	p, ct := procWith(Extensions{Files: true})

	fileTokens := []string{"main.go", "config.yaml", "/var/log/syslog.1", "./app.py", "src/pkg/x.rs"}
	for _, f := range fileTokens {
		out := render(p, f)
		assert.Equal(t, f, stripAnsi(out), "file token preserved: %q", f)
		assert.Contains(t, out, fgSeq(ct, color.File), "%q should be File-colored", f)
	}

	// Extensionless paths are directories.
	for _, d := range []string{"/usr/bin", "/etc/hosts"} {
		out := render(p, d)
		assert.Equal(t, d, stripAnsi(out))
		assert.Contains(t, out, fgSeq(ct, color.Dir), "%q should be Dir-colored", d)
	}

	// Hostnames must NOT be hijacked as files (com is not a file extension).
	out := render(p, "example.com")
	assert.Contains(t, out, fgSeq(ct, color.Host), "example.com stays a host")

	// "and/or" is not a path.
	assert.False(t, looksLikePath("and/or"))
	assert.True(t, looksLikePath("/abs"))
	assert.True(t, looksLikePath("./rel.txt"))
	assert.True(t, looksLikePath("a/b/c.go"))
}

// ---------------------------------------------------------------------------
// The non-negotiable invariant: stripped(output) == input, always.
// ---------------------------------------------------------------------------

// invariantCorpus is a deliberately adversarial set of lines.
var invariantCorpus = []string{
	"",
	" ",
	"   ",
	"a  b   c",
	" leading and trailing ",
	"Sep 14 11:45:00 myhost sshd[1234]: Connection closed",
	"level=info msg=\"started\" user=alice dur=1m30s count=42",
	"[ERROR] [auth] login failed for user=bob from 10.0.0.1",
	"path=/var/log/app.log size=4.5kb took=300µs",
	"GET /index.html?a=b&c=d HTTP/1.1 200 1234",
	"[]  []  [x]  ][  =  ==  ===  a=b=c=d",
	"unicode: café ☃ µs π=3.14 名前=値",
	"weird::ipv6::beef and aa:bb:cc:dd:ee:ff mac",
	"--flag=1 -x=2 ---=3 ====",
	"https://example.com/path?q=1 user@host.com 0xdeadbeef",
	"trailing punctuation!!! (parenthetical) [bracketed], <angled>",
	"key=\"value with no closing and a [ bracket",
	strings.Repeat("k=v ", 200),
	strings.Repeat("verylongtokenwithoutspaces", 500),
	"tabs\tand\tstuff", // tabs are not the split delimiter (space is)
	"[2026.06.28-13.42.12:957][  0]LogCoreRedirects: Verbose: RedirectNameAndValues(/Script/BlueprintGraph.K2Node_CallFunction:bIsPureFunc) replaced by /Script/X",
	"LogTemp: Warning: something happened with code=5",
	"make[1]: *** No rule to make target 'folder/file.cpp', needed by 'folder/file.o'.  Stop.",
	"make[1]: *** Waiting for unfinished jobs....",
	"make: *** [Makefile:10: all] Error 1",
	"make[1]: Entering directory '/tmp/ccze/src'",
}

func TestExtInvariantAllCombos(t *testing.T) {
	// Exhaustively toggle every extension combination (2^6) and assert the
	// visible text is never altered for any corpus line.
	for mask := 0; mask < 128; mask++ {
		e := Extensions{
			Tags:      mask&1 != 0,
			Files:     mask&2 != 0,
			Slog:      mask&4 != 0,
			Durations: mask&8 != 0,
			Unreal:    mask&16 != 0,
			Adaptive:  mask&32 != 0,
			Make:      mask&64 != 0,
		}
		p, _ := procWith(e)
		for _, line := range invariantCorpus {
			out := render(p, line)
			assert.Equal(t, line, stripAnsi(out),
				"INVARIANT VIOLATED (mask=%07b): input=%q", mask, line)
		}
	}
}

// TestExtDisabledIdenticalToBaseline proves that with all extensions off, the
// output is byte-for-byte identical to a processor that never had extensions
// configured — i.e. the C-compatible path is untouched.
func TestExtDisabledIdenticalToBaseline(t *testing.T) {
	baseline, _ := newTestProcessor()
	withExt, _ := procWith(Extensions{}) // all false

	for _, line := range invariantCorpus {
		var a, b bytes.Buffer
		baseline.Process(&a, line, true, false)
		withExt.Process(&b, line, true, false)
		assert.Equal(t, a.String(), b.String(),
			"extensions-off output must match baseline for %q", line)
	}
}
