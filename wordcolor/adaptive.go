// Package wordcolor — adaptive.go.
//
// Adaptive recurring-structure recognition (opt-in, enabled by -o adaptive).
//
// This recognizer learns the recurring shape of the log stream and colorizes a
// line ONLY when that line participates in a confirmed recurring delimited
// structure (a stable number of fields separated by a single-byte delimiter
// such as '|', ',' or TAB — CSV-ish / piped / fixed-field logs). For such a
// line it paints each positional column a stable color so columns line up
// visually by color across the whole stream, gives recurring key names
// (key=value / key:value fields) a stable per-key color so the same key is
// always the same color, and keeps recognizable tokens (IPs, numbers, error
// words, ...) in their usual semantic color. For every other line it declines
// (process returns false) and the caller falls back to the standard per-word
// pipeline.
//
// Why "only confirmed recurring shapes": a single one-off line carries no
// recurring structure to learn, and the standard cascade (plus the other -o
// highlighters, e.g. slog key=value) already color such lines well. Taking over
// a line only once its shape has recurred is what makes this "adaptive" and
// keeps it from fighting the other highlighters.
//
// INVARIANT (hard requirement): adaptive processing may ONLY add ANSI color.
// For any input line, the output with all escape sequences stripped equals the
// input byte-for-byte. Every emission below routes the ORIGINAL substrings
// through color.Table.WriteColored, which copies its text argument through
// unchanged; the concatenation of the substrings emitted for a line is exactly
// the line. See adaptive_test.go for the invariant proofs (subsequence + non-
// escape-equality) across an adversarial corpus, including inputs that already
// contain ANSI escapes.
//
// State is strictly bounded, so it is safe on an unbounded stream: a per-key
// color map capped at maxKeys (with a generational reset on overflow) plus a
// handful of integer counters for shape confirmation. Nothing buffers the
// stream.
package wordcolor

import (
	"io"

	"ccze-go/color"
)

// adaptivePalette is the ordered set of color slots the adaptive layer paints
// with. They are the high, visually distinct static slots from the color enum,
// which the normal cascade never uses, so adaptive coloring is unambiguous to
// the eye and cannot be confused with a semantic color. Order is chosen so
// adjacent columns / hash buckets are easy to tell apart.
var adaptivePalette = [...]color.Color{
	color.StaticBoldCyan,
	color.StaticBoldYellow,
	color.StaticBoldGreen,
	color.StaticBoldMagenta,
	color.StaticBoldBlue,
	color.StaticBoldRed,
	color.StaticCyan,
	color.StaticYellow,
	color.StaticGreen,
	color.StaticMagenta,
	color.StaticBlue,
	color.StaticWhite,
}

// Tuning constants. All cross-line state is bounded by these.
const (
	// maxKeys caps the number of distinct key->color bindings retained. On
	// overflow the map is cleared (a cheap generational reset) rather than
	// growing without bound: memory is O(maxKeys) regardless of stream length.
	maxKeys = 256

	// minKeyLen / maxKeyLen bound what counts as a "key" name.
	minKeyLen = 1
	maxKeyLen = 40

	// fieldConfirmCount is how many consecutive lines must share the same
	// (delimiter, field-count) shape before the shape is considered recurring
	// and its lines are colorized. This prevents a single coincidental line
	// (one stray line containing a few pipes) from triggering column coloring.
	fieldConfirmCount = 3

	// minFields / maxFields bound a delimited shape worth coloring. Two fields
	// (one delimiter) is the minimum interesting structure; the upper bound
	// keeps the per-line walk cheap and bounded.
	minFields = 2
	maxFields = 64
)

// fieldDelims are the single-byte delimiters considered for positional-field
// structure, in priority order. TAB and '|' are strong structure signals; ','
// is common in CSV-ish logs.
var fieldDelims = [...]byte{'\t', '|', ','}

// adaptive holds bounded cross-line state for recurring-structure recognition.
// It is NOT safe for concurrent use; the colorizer pipeline is single-threaded
// per stream, matching how Processor is used elsewhere.
type adaptive struct {
	// keyColor binds a key name to a palette index. Capped at maxKeys entries.
	keyColor map[string]int
	// nextSlot is the round-robin cursor for assigning a NEW key its slot, so
	// distinct keys seen early get distinct colors instead of hash collisions.
	nextSlot int

	// recurring-shape detection (bounded integer state):
	curDelim  byte // delimiter of the shape currently being confirmed/active
	curFields int  // field count of that shape
	runLen    int  // consecutive lines matching (curDelim, curFields)
	active    bool // true once runLen has reached fieldConfirmCount
}

// newAdaptive constructs the recognizer.
func newAdaptive() *adaptive {
	return &adaptive{keyColor: make(map[string]int, 64)}
}

// process attempts to colorize an entire message using learned structure. It
// returns true if it fully handled the line (wrote every byte of msg to w,
// colorized). A false return means "not handled": the caller falls back to the
// normal per-word path and msg has NOT been written.
//
// It handles a line iff that line participates in a CONFIRMED recurring
// delimited shape (see the package doc).
func (a *adaptive) process(p *Processor, w io.Writer, msg string) bool {
	if msg == "" {
		return false
	}

	delim, fields := detectShape(msg)
	if !a.observe(delim, fields) {
		// Either no delimited shape, or the shape has not recurred enough yet.
		// Decline so the standard pipeline (and other -o highlighters) runs.
		return false
	}

	// This line is part of a confirmed recurring shape: colorize its columns.
	a.emitColumns(p, w, msg, delim)
	return true
}

// observe advances the recurring-shape state machine with the shape detected on
// the current line and reports whether the current line should be treated as a
// member of a confirmed recurring shape.
func (a *adaptive) observe(delim byte, fields int) bool {
	switch {
	case delim != 0 && delim == a.curDelim && fields == a.curFields:
		// Same shape continues.
		if a.runLen < fieldConfirmCount {
			a.runLen++
		}
		if a.runLen >= fieldConfirmCount {
			a.active = true
		}
	case delim != 0:
		// New candidate shape; (re)start confirmation.
		a.curDelim = delim
		a.curFields = fields
		a.runLen = 1
		a.active = fieldConfirmCount <= 1
	default:
		// No delimited shape: a blank/free-form line breaks any block.
		a.curDelim = 0
		a.curFields = 0
		a.runLen = 0
		a.active = false
	}
	return a.active && delim != 0 && delim == a.curDelim && fields == a.curFields
}

// ---------------------------------------------------------------------------
// Shape detection
// ---------------------------------------------------------------------------

// countDelims counts occurrences of d in msg, capped so a pathological line
// cannot blow the bound. Field count is delimiters+1.
func countDelims(msg string, d byte) int {
	count := 0
	for i := 0; i < len(msg); i++ {
		if msg[i] == d {
			count++
			if count > maxFields {
				return count
			}
		}
	}
	return count
}

// detectShape picks the best delimited shape for msg: the first delimiter (in
// fieldDelims priority order) whose field count lands within [minFields,
// maxFields]. Returns delim=0 if none qualifies.
func detectShape(msg string) (delim byte, fields int) {
	for _, d := range fieldDelims {
		f := countDelims(msg, d) + 1
		if f >= minFields && f <= maxFields {
			return d, f
		}
	}
	return 0, 0
}

// ---------------------------------------------------------------------------
// Key color assignment (stable across the whole stream)
// ---------------------------------------------------------------------------

// isAdaptiveKeyByte reports whether c is allowed inside a key name. Keys are
// identifier-like: letters, digits, '_', '-', '.'.
func isAdaptiveKeyByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.'
}

func isAdaptiveKeyStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

// splitFieldKey reports whether field has the shape key<sep>value (sep ∈ {=,:})
// with an identifier-like key of bounded length and a non-empty value, and if
// so returns the key. The separator and value are recovered by the caller as
// field[len(key)] and field[len(key)+1:], so no bytes are lost.
func splitFieldKey(field string) (key string, ok bool) {
	n := len(field)
	if n < minKeyLen+2 {
		return "", false
	}
	if !isAdaptiveKeyStart(field[0]) {
		return "", false
	}
	i := 0
	for i < n && isAdaptiveKeyByte(field[i]) {
		i++
	}
	if i == 0 || i >= n {
		return "", false
	}
	if c := field[i]; c != '=' && c != ':' {
		return "", false
	}
	if i < minKeyLen || i > maxKeyLen {
		return "", false
	}
	if i+1 >= n { // need a non-empty value
		return "", false
	}
	return field[:i], true
}

// fnv1a is a tiny, allocation-free FNV-1a hash over a string.
func fnv1a(s string) uint32 {
	const (
		offset = 2166136261
		prime  = 16777619
	)
	h := uint32(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	return h
}

// keySlot returns the stable palette index bound to key, assigning one on first
// sight (round-robin while slots remain, hashed thereafter) and bounding the map
// at maxKeys with a generational reset.
func (a *adaptive) keySlot(key string) int {
	if slot, ok := a.keyColor[key]; ok {
		return slot
	}
	if len(a.keyColor) >= maxKeys {
		for k := range a.keyColor {
			delete(a.keyColor, k)
		}
		a.nextSlot = 0
	}
	var slot int
	if a.nextSlot < len(adaptivePalette) {
		slot = a.nextSlot
		a.nextSlot++
	} else {
		slot = int(fnv1a(key) % uint32(len(adaptivePalette)))
	}
	a.keyColor[key] = slot
	return slot
}

// fieldSlot maps a 0-based column index to a palette index (columns cycle).
func fieldSlot(col int) int {
	if col < 0 {
		col = 0
	}
	return col % len(adaptivePalette)
}

// ---------------------------------------------------------------------------
// Emission
// ---------------------------------------------------------------------------

// emitColumns writes msg split on delimiter d, coloring each field by column
// (or by a stable per-key color when the field is key=value, or by its semantic
// color when the field is a recognizable token), and the delimiters in Default.
//
// Byte preservation: the loop walks msg once, emitting, for each column, the
// field substring followed by the single delimiter byte; the final field has no
// trailing delimiter. The concatenation of every substring written is exactly
// msg. Every write goes through ct.WriteColored, which never alters text.
func (a *adaptive) emitColumns(p *Processor, w io.Writer, msg string, d byte) {
	ct := p.ct
	col := 0
	start := 0
	for i := 0; i < len(msg); i++ {
		if msg[i] == d {
			a.emitField(p, w, msg[start:i], col)
			ct.WriteColored(w, color.Default, msg[i:i+1]) // delimiter byte, verbatim
			col++
			start = i + 1
		}
	}
	a.emitField(p, w, msg[start:], col) // trailing field (may be empty -> no-op)
}

// emitField colors one field. Empty fields write nothing (WriteColored
// short-circuits on ""). Priority: a key=value field gets its key painted a
// stable per-key color and its value sub-colored; else a field that classifies
// as a semantic token keeps that color; else the stable per-column color. None
// of these change the bytes.
func (a *adaptive) emitField(p *Processor, w io.Writer, field string, col int) {
	if field == "" {
		return
	}
	ct := p.ct

	// key=value / key:value -> stable per-key color + sub-colored value.
	if key, ok := splitFieldKey(field); ok {
		slot := a.keySlot(key)
		rest := field[len(key):] // sep byte + value
		ct.WriteColored(w, adaptivePalette[slot], key)
		ct.WriteColored(w, color.Default, rest[:1]) // separator byte, verbatim
		value := rest[1:]
		ct.WriteColored(w, classifyWord(value), value)
		return
	}

	// Recognizable standalone token -> keep its semantic color.
	if sc := classifyWord(field); sc != color.Default {
		ct.WriteColored(w, sc, field)
		return
	}

	// Otherwise: stable per-column color.
	ct.WriteColored(w, adaptivePalette[fieldSlot(col)], field)
}

// classifyWord returns the semantic color a bare token (no surrounding
// punctuation) would receive from the standard cascade, WITHOUT writing
// anything. Mirrors the order in ProcessOne's switch so adaptive field colors
// stay consistent with the rest of the pipeline. Returns color.Default when
// nothing matches. The host[ip] split case is collapsed to a single Host color
// because a field is colored as one unit; bytes are preserved regardless.
func classifyWord(word string) color.Color {
	if word == "" {
		return color.Default
	}
	lword := toLowerASCII(word)

	switch {
	case matchHost(lword):
		return color.Host
	case matchMAC(lword):
		return color.MAC
	case lword[0] == '/':
		return color.Dir
	case matchEmail(lword) && matchEmail2(lword):
		return color.Email
	case matchMsgID(lword):
		return color.Email
	case matchURI(lword):
		return color.URI
	case matchSize(lword):
		return color.Size
	case matchVer(lword):
		return color.Version
	case matchTime(lword):
		return color.Date
	case matchAddr(lword):
		return color.Address
	case matchNum(lword):
		return color.Numbers
	case matchSig(lword):
		return color.Signal
	case matchHostIP(lword):
		return color.Host
	}

	col := color.Default
	for _, kw := range wordsBad {
		if hasPrefixASCII(lword, kw) {
			col = color.BadWord
		}
	}
	for _, kw := range wordsGood {
		if hasPrefixASCII(lword, kw) {
			col = color.GoodWord
		}
	}
	for _, kw := range wordsError {
		if hasPrefixASCII(lword, kw) {
			col = color.Error
		}
	}
	for _, kw := range wordsSystem {
		if hasPrefixASCII(lword, kw) {
			col = color.SystemWord
		}
	}
	return col
}

// toLowerASCII lowercases ASCII letters only, leaving every other byte
// (including multi-byte UTF-8) untouched. Returns the input unchanged when it
// has no uppercase ASCII (no allocation in that common case).
func toLowerASCII(s string) string {
	hasUpper := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return s
	}
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// hasPrefixASCII is strings.HasPrefix, inlined to keep imports minimal and the
// hot path allocation-free.
func hasPrefixASCII(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
