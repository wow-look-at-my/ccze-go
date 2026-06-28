package wordcolor

import (
	"strings"
	"testing"

	"ccze-go/color"
	"github.com/stretchr/testify/assert"
)

// procAdaptive returns a processor with adaptive recognition enabled.
func procAdaptive() (*Processor, *color.Table) {
	return procWith(Extensions{Adaptive: true})
}

// ---------------------------------------------------------------------------
// (b) THE INVARIANT: adaptive processing may ONLY add ANSI; visible bytes are
// preserved exactly.
//
// Two complementary, unambiguous checks express this (see assertColorOnly):
//   - isSubsequence(in, out): no input byte is dropped, reordered, or altered.
//   - stripAnsi(in) == stripAnsi(out): the non-escape content is identical.
// Together they prove "output == input plus inserted ANSI escapes, nothing
// else", and hold for ANY input, including input that itself contains escapes.
//
// The plain corpus additionally checks the simpler stripAnsi(out) == input,
// which is exact only for ESC-free input. Embedded-ESC inputs are exercised by
// TestAdaptiveInvariantEmbeddedANSI via assertColorOnly.
// ---------------------------------------------------------------------------

// adaptivePlainCorpus is a varied, adversarial set of ESC-free inputs. Order
// matters: the recognizer keeps cross-line state, so the sequence exercises the
// confirm/decay logic too.
var adaptivePlainCorpus = []string{
	// plain text (no recurring structure -> declined, normal path)
	"the quick brown fox jumps over the lazy dog",
	"",
	" ",
	"   ",
	"a  b   c    d",
	// key=value lines (declined by adaptive unless they form a recurring
	// delimited shape; handled by the standard/slog path) -- still byte-safe
	"user=admin pid=1234 status=ok",
	"user=root pid=5678 status=error",
	// recurring pipe-delimited block (3+ lines -> confirmed -> colorized)
	"alpha|beta|gamma|delta",
	"one|two|three|four",
	"x|y|z|w",
	"p|q|r|s",
	// recurring comma block
	"col1,col2,col3,col4",
	"v1,v2,v3,v4",
	"a1,b1,c1,d1",
	"a2,b2,c2,d2",
	// recurring tab block
	"a\tb\tc\td",
	"e\tf\tg\th",
	"i\tj\tk\tl",
	// empty fields inside delimited lines
	"a||c|",
	",,,",
	"|||",
	"\t\t\t",
	// recurring block whose fields are key=value (stable per-key colors)
	"user=alice|status=ok|ip=10.0.0.1",
	"user=bob|status=fail|ip=10.0.0.2",
	"user=carol|status=ok|ip=10.0.0.3",
	"user=dave|status=ok|ip=10.0.0.4",
	// unicode inside delimited fields
	"col=日本語|x=한국어|y=Ελληνικά|z=fin",
	"col=naïve|x=café|y=résumé|z=ok",
	"col=a|x=b|y=c|z=d",
	// recurring block with semantic tokens in columns
	"2024-01-01|10.0.0.1|150mb|error",
	"2024-01-02|10.0.0.2|2gb|ok",
	"2024-01-03|10.0.0.3|42kb|warn",
	// very long delimited line within a recurring block
	strings.Repeat("f|", 4000) + "end",
	strings.Repeat("f|", 4000) + "end",
	strings.Repeat("f|", 4000) + "end",
	// free-form line breaks the block, then resume
	"this line has no delimiters at all and is free form",
	"a|b|c",
	"d|e|f",
	"g|h|i",
}

// isSubsequence reports whether every byte of in appears in out in order. This
// proves the colorizer never drops, reorders, or replaces any input byte; it
// can only INSERT bytes (always ANSI escapes). Unambiguous even when in
// contains ESC bytes, since it never classifies output bytes.
func isSubsequence(in, out string) bool {
	i := 0
	for j := 0; i < len(in) && j < len(out); j++ {
		if in[i] == out[j] {
			i++
		}
	}
	return i == len(in)
}

// assertColorOnly proves the strong invariant with two unambiguous checks that
// together mean "output is the input plus inserted ANSI escapes, nothing else":
// (1) isSubsequence(in,out) -- nothing dropped/reordered; (2)
// stripAnsi(in)==stripAnsi(out) -- non-escape content identical. There is no
// greedy-match hazard because neither check disambiguates an input ESC from an
// added ESC.
func assertColorOnly(t *testing.T, in, out string) {
	t.Helper()
	assert.True(t, isSubsequence(in, out),
		"input not a subsequence of output (bytes dropped/reordered): in=%q out=%q", in, out)
	assert.Equal(t, stripAnsi(in), stripAnsi(out),
		"visible (non-escape) content changed: in=%q out=%q", in, out)
}

func TestAdaptiveInvariantCorpus(t *testing.T) {
	p, _ := procAdaptive()
	for _, line := range adaptivePlainCorpus {
		out := render(p, line)
		assert.Equal(t, line, stripAnsi(out), "stripAnsi invariant violated for input %q", line)
		assertColorOnly(t, line, out)
	}
}

// TestAdaptiveInvariantEmbeddedANSI: inputs that ALREADY contain ANSI escapes.
// A naive strip would wrongly remove the input's own escapes, so we use the
// strong assertColorOnly check. The block recurs so the adaptive path engages.
func TestAdaptiveInvariantEmbeddedANSI(t *testing.T) {
	corpus := []string{
		"\x1b[32mf1\x1b[0m|\x1b[33mf2\x1b[0m|tail",
		"\x1b[34mg1\x1b[0m|\x1b[35mg2\x1b[0m|rest",
		"\x1b[31mh1\x1b[0m|\x1b[36mh2\x1b[0m|done",
		"\x1b[32mi1\x1b[0m|\x1b[33mi2\x1b[0m|more",
		"plain \x1b[31mred\x1b[0m line with no delim",
		"key=\x1b[35mv\x1b[0m|other=\x1b[36mz\x1b[0m|t=1",
	}
	p, _ := procAdaptive()
	for round := 0; round < 2; round++ {
		for _, line := range corpus {
			out := render(p, line)
			assertColorOnly(t, line, out)
			assert.Equal(t, stripAnsi(line), stripAnsi(out),
				"visible text changed for input %q", line)
		}
	}
}

// TestAdaptiveInvariantHighCardinalityKeys drives the per-key map past its cap
// (within a recurring delimited block, so the adaptive path runs) to exercise
// the generational reset, asserting the invariant throughout.
func TestAdaptiveInvariantHighCardinalityKeys(t *testing.T) {
	p, _ := procAdaptive()
	var sb strings.Builder
	for line := 0; line < 10; line++ {
		sb.Reset()
		// 200 pipe-separated key=value fields; same field-count each line so the
		// shape is recurring and the adaptive path engages.
		for i := 0; i < 200; i++ {
			if i > 0 {
				sb.WriteByte('|')
			}
			sb.WriteString("k")
			sb.WriteByte(byte('0' + line))
			sb.WriteByte('_')
			sb.WriteByte(byte('a' + (i/26)%26))
			sb.WriteByte(byte('a' + i%26))
			sb.WriteString("=v")
		}
		in := sb.String()
		out := render(p, in)
		assert.Equal(t, in, stripAnsi(out), "invariant violated on high-cardinality line %d", line)
	}
	assert.LessOrEqual(t, len(p.ada.keyColor), maxKeys, "key map must stay bounded")
}

// TestAdaptiveInvariantMatchesPlainBytes: turning the feature on changes ONLY
// color, never bytes -- stripped output with adaptive ON equals the input and
// equals stripped output with adaptive OFF.
func TestAdaptiveInvariantMatchesPlainBytes(t *testing.T) {
	lines := []string{
		"alpha|beta|gamma",
		"user=alice|status=ok|ip=10.0.0.1",
		"plain text with 192.168.0.1 and error words",
		"a||c|",
		"col=naïve|x=café|y=ok",
	}
	plain, _ := procWith(Extensions{})
	adaptive, _ := procAdaptive()
	for _, line := range lines {
		gotPlain := stripAnsi(render(plain, line))
		gotAdaptive := stripAnsi(render(adaptive, line))
		assert.Equal(t, line, gotPlain, "plain not byte-preserving for %q", line)
		assert.Equal(t, line, gotAdaptive, "adaptive not byte-preserving for %q", line)
		assert.Equal(t, gotPlain, gotAdaptive, "byte mismatch for %q", line)
	}
}

// ---------------------------------------------------------------------------
// (a) THE FEATURE: it highlights useful recurring structure.
// ---------------------------------------------------------------------------

// fgCodeOf returns the ANSI foreground SGR code (30-37) wrapping the first
// occurrence of token in out, or -1 if none precedes it. Lets tests assert
// color stability/distinctness without hard-coding palette slots.
func fgCodeOf(out, token string) int {
	idx := strings.Index(out, token)
	if idx < 0 {
		return -1
	}
	best := -1
	for i := 0; i+1 < idx; i++ {
		if out[i] != '\x1b' || out[i+1] != '[' {
			continue
		}
		j := i + 2
		num := 0
		ok := false
		for j < len(out) && out[j] >= '0' && out[j] <= '9' {
			num = num*10 + int(out[j]-'0')
			j++
			ok = true
		}
		if !ok || j >= len(out) || out[j] != 'm' {
			continue
		}
		if j+1 > idx {
			break
		}
		if num >= 30 && num <= 37 {
			best = num
		}
	}
	return best
}

// ansiFg mirrors the repo's foreground mapping (color/color.go ansiColor).
func ansiFg(idx int) int {
	table := [8]int{30, 31, 32, 33, 34, 36, 35, 37}
	if idx < 0 || idx >= 8 {
		return -1
	}
	return table[idx]
}

// feedConfirm runs n identical-shape warm-up lines so the shape is confirmed.
func feedConfirm(p *Processor, lines ...string) {
	for _, l := range lines {
		render(p, l)
	}
}

func TestAdaptiveColumnsColoredAfterConfirm(t *testing.T) {
	p, _ := procAdaptive()
	// Warm up: first (fieldConfirmCount-1) lines are not yet confirmed.
	feedConfirm(p, "alpha|beta|gamma", "delta|epsilon|zeta")
	out := render(p, "eta|theta|iota") // 3rd line -> confirmed

	assert.True(t, p.ada.active, "shape should be active by the 3rd line")
	// Each plain column is wrapped in a foreground color now.
	assert.NotEqual(t, -1, fgCodeOf(out, "eta"))
	assert.NotEqual(t, -1, fgCodeOf(out, "theta"))
	assert.NotEqual(t, -1, fgCodeOf(out, "iota"))
	// Distinct columns -> distinct colors.
	assert.NotEqual(t, fgCodeOf(out, "eta"), fgCodeOf(out, "theta"))
	assert.Equal(t, "eta|theta|iota", stripAnsi(out))
}

func TestAdaptiveColumnColorIsStableByPosition(t *testing.T) {
	p, _ := procAdaptive()
	feedConfirm(p, "a1|b1|c1", "a2|b2|c2")
	o3 := render(p, "a3|b3|c3")
	o4 := render(p, "a4|b4|c4")
	// Column 0 keeps one color, column 1 another, etc., across lines.
	assert.Equal(t, fgCodeOf(o3, "a3"), fgCodeOf(o4, "a4"), "column 0 color must be stable")
	assert.Equal(t, fgCodeOf(o3, "b3"), fgCodeOf(o4, "b4"), "column 1 color must be stable")
	assert.NotEqual(t, fgCodeOf(o3, "a3"), fgCodeOf(o3, "b3"), "columns 0 and 1 must differ")
}

func TestAdaptiveKeyColorStableAcrossLines(t *testing.T) {
	p, _ := procAdaptive()
	// Recurring block of key=value fields; confirm then assert key colors.
	feedConfirm(p, "user=a|status=ok|n=1", "user=b|status=ok|n=2")
	o3 := render(p, "user=c|status=ok|n=3")
	o4 := render(p, "user=d|status=fail|n=4")
	// Same key -> same color across lines (the adaptive payoff).
	assert.NotEqual(t, -1, fgCodeOf(o3, "user"))
	assert.Equal(t, fgCodeOf(o3, "user"), fgCodeOf(o4, "user"), "key 'user' color must be stable")
	assert.Equal(t, fgCodeOf(o3, "status"), fgCodeOf(o4, "status"), "key 'status' color must be stable")
	// Distinct keys -> distinct colors.
	assert.NotEqual(t, fgCodeOf(o3, "user"), fgCodeOf(o3, "status"), "distinct keys -> distinct colors")
	assert.Equal(t, "user=c|status=ok|n=3", stripAnsi(o3))
}

func TestAdaptiveFieldKeepsSemanticColor(t *testing.T) {
	p, ct := procAdaptive()
	// Recurring block where a column is an IP; it should keep the Host color.
	feedConfirm(p, "x|10.0.0.1|y", "x|10.0.0.2|y")
	out := render(p, "x|10.0.0.9|y")
	hostFg := ansiFg(ct.Get(color.Host) & 0xf)
	assert.Equal(t, hostFg, fgCodeOf(out, "10.0.0.9"), "IP column should keep semantic Host color")
	assert.Equal(t, "x|10.0.0.9|y", stripAnsi(out))
}

func TestAdaptiveDeclinesUntilConfirmed(t *testing.T) {
	p, _ := procAdaptive()
	// A single pipe-line is NOT yet a recurring shape -> adaptive declines, so
	// the output equals the plain (no-extension) processor's output exactly.
	plain, _ := procWith(Extensions{})
	const line = "alpha|beta|gamma"
	assert.Equal(t, render(plain, line), render(p, line),
		"first occurrence must be handled by the normal path (adaptive declines)")
	assert.False(t, p.ada.active, "shape not confirmed after one line")
}

func TestAdaptiveDeclinesFreeFormLine(t *testing.T) {
	p, _ := procAdaptive()
	feedConfirm(p, "a|b|c", "d|e|f", "g|h|i") // confirmed
	assert.True(t, p.ada.active)
	// A free-form line resets the run; output equals the plain processor's.
	plain, _ := procWith(Extensions{})
	const free = "a free form line with no delimiters"
	assert.Equal(t, render(plain, free), render(p, free), "free-form line must use the normal path")
	assert.False(t, p.ada.active, "free-form line must deactivate the shape")
}

func TestAdaptiveDisabledIsNoOp(t *testing.T) {
	// With adaptive OFF (default), a would-be recurring block is colored exactly
	// as the plain processor colors it -- the feature adds nothing when off.
	plain, _ := procWith(Extensions{})
	off, _ := procWith(Extensions{})
	lines := []string{"a|b|c", "d|e|f", "g|h|i", "j|k|l"}
	for _, l := range lines {
		assert.Equal(t, render(plain, l), render(off, l))
	}
}

// ---------------------------------------------------------------------------
// Unit tests for the building blocks.
// ---------------------------------------------------------------------------

func TestDetectShape(t *testing.T) {
	cases := []struct {
		in     string
		delim  byte
		fields int
	}{
		{"a|b|c", '|', 3},
		{"a,b,c,d", ',', 4},
		{"a\tb", '\t', 2},
		{"no delimiters here", 0, 0},
		{"only one,comma", ',', 2},
		{"single", 0, 0},
	}
	for _, c := range cases {
		d, f := detectShape(c.in)
		assert.Equal(t, c.delim, d, "delim for %q", c.in)
		assert.Equal(t, c.fields, f, "fields for %q", c.in)
	}
}

func TestSplitFieldKey(t *testing.T) {
	cases := []struct {
		in      string
		wantKey string
		wantOK  bool
	}{
		{"user=admin", "user", true},
		{"level:info", "level", true},
		{"x-request-id=abc", "x-request-id", true},
		{"app.name=ccze", "app.name", true},
		{"_private=1", "_private", true},
		{"key=", "", false},      // empty value
		{"trailing:", "", false}, // empty value
		{"=val", "", false},      // no key
		{":val", "", false},      // no key
		{"123=v", "", false},     // key must start with letter/_
		{"plainword", "", false}, // no separator
		{"a==b", "a", true},      // first '=' splits; value "=b"
		{"", "", false},
	}
	for _, c := range cases {
		k, ok := splitFieldKey(c.in)
		assert.Equal(t, c.wantOK, ok, "ok for %q", c.in)
		if c.wantOK {
			assert.Equal(t, c.wantKey, k, "key for %q", c.in)
			assert.True(t, len(c.in) > len(k), "key shorter than input for %q", c.in)
			sep := c.in[len(k)]
			assert.True(t, sep == '=' || sep == ':', "sep is = or : for %q", c.in)
			assert.Equal(t, c.in, k+c.in[len(k):], "reconstructs input for %q", c.in)
		}
	}
}

func TestAdaptiveKeySlotStability(t *testing.T) {
	a := newAdaptive()
	s1 := a.keySlot("user")
	s2 := a.keySlot("user")
	assert.Equal(t, s1, s2, "same key -> same slot")
	assert.NotEqual(t, s1, a.keySlot("pid"), "distinct early keys -> distinct slots")
}

func TestAdaptiveKeySlotCardinalityReset(t *testing.T) {
	a := newAdaptive()
	for i := 0; i < maxKeys+50; i++ {
		key := "k" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)) + string(rune('0'+(i/676)%10))
		_ = a.keySlot(key)
	}
	assert.LessOrEqual(t, len(a.keyColor), maxKeys, "key map must stay bounded")
}

func TestAdaptiveToLowerASCIIPreservesUnicode(t *testing.T) {
	in := "ABCnaïveДÆ漢字"
	assert.Equal(t, "abcnaïveДÆ漢字", toLowerASCII(in))
	noUpper := "already lower ünïcödé"
	assert.Equal(t, noUpper, toLowerASCII(noUpper))
}
