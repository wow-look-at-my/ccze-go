// Package wordcolor implements word-level colorization for CCZE log output.
package wordcolor

import (
	"io"
	"strings"

	"ccze-go/color"
	"github.com/wow-look-at-my/go-containers/set"
)

// Processor holds a color table for performing word-level colorization of log
// messages.
type Processor struct {
	ct  *color.Table
	ext Extensions
	ada *adaptive // non-nil only when ext.Adaptive is set
}

// Extensions, the opt-in highlighters, and their helpers live in extensions.go.

var wordsBad = []string{
	"warn", "restart", "exit", "stop", "end", "shutting", "down", "close",
	"unreach", "can't", "cannot", "skip", "deny", "disable", "ignored",
	"miss", "oops", "not", "backdoor", "blocking", "ignoring",
	"unable", "readonly", "offline", "terminate", "empty", "virus",
}

var wordsGood = []string{
	"activ", "start", "ready", "online", "load", "ok", "register", "detected",
	"configured", "enable", "listen", "open", "complete", "attempt", "done",
	"check", "listen", "connect", "finish", "clean",
}

var wordsError = []string{
	"error", "crit", "invalid", "fail", "false", "alarm", "fatal",
}

var wordsSystem = []string{
	"ext2-fs", "reiserfs", "vfs", "iso", "isofs", "cslip", "ppp", "bsd",
	"linux", "tcp/ip", "mtrr", "pci", "isa", "scsi", "ide", "atapi",
	"bios", "cpu", "fpu", "discharging", "resume",
}

// kwEntry is one keyword-prefix rule: a word starting with kw gets col.
type kwEntry struct {
	kw  string
	col color.Color
}

// kwByFirst indexes the bad/good/error/system keyword lists by first byte.
var kwByFirst [256][]kwEntry

func init() {
	add := func(words []string, col color.Color) {
		for _, w := range words {
			kwByFirst[w[0]] = append(kwByFirst[w[0]], kwEntry{w, col})
		}
	}
	// Highest priority first within each bucket.
	add(wordsSystem, color.SystemWord)
	add(wordsError, color.Error)
	add(wordsGood, color.GoodWord)
	add(wordsBad, color.BadWord)
}

// Signal name lookup table. The regex was ^sig(name...) which is effectively
// a hash-set lookup after stripping the "sig" prefix.
var sigNames = set.Of[string]("hup", "int", "quit", "ill", "abrt",
	"fpe", "kill", "segv", "pipe", "alrm",
	"term", "usr1", "usr2", "chld", "cont",
	"stop", "tstp", "tin", "tout", "bus",
	"poll", "prof", "sys", "trap", "urg",
	"vtalrm", "xcpu", "xfsz", "iot", "emt",
	"stkflt", "io", "cld", "pwr", "info",
	"lost", "winch", "unused")

// New creates a new Processor with the given color table.
func New(ct *color.Table) *Processor {
	return &Processor{ct: ct}
}

// ---------------------------------------------------------------------------
// Character classification helpers
// ---------------------------------------------------------------------------

func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isHexLower(c byte) bool   { return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') }
func isLowerAlpha(c byte) bool { return c >= 'a' && c <= 'z' }
func isLowerAlnum(c byte) bool { return isLowerAlpha(c) || isDigit(c) }
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || isDigit(c) || c == '_'
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return len(s) > 0
}

// Punctuation stripping (replaces regPre / regPost)

func isPunctPre(c byte) bool {
	switch c {
	case '`', '\'', '"', '.', ',', '!', '?', ':', ';', '(', '[', '{', '<':
		return true
	}
	return false
}

func isPunctPost(c byte) bool {
	switch c {
	case '`', '\'', '"', '.', ',', '!', '?', ':', ';', ')', ']', '}', '>':
		return true
	}
	return false
}

// splitPre strips leading punctuation from word.
// Equivalent to: ^([`'".,!?:;(\[{<]+)([^`'".,!?:;(\[{<]\S*)$
func splitPre(word string) (pre, rest string) {
	i := 0
	for i < len(word) && isPunctPre(word[i]) {
		i++
	}
	if i == 0 || i >= len(word) {
		return "", word
	}
	return word[:i], word[i:]
}

// splitPost strips trailing punctuation from word.
// Equivalent to: ^(\S*[^`'".,!?:;)\]}>])([`'".,!?:;)\]}>]+)$
func splitPost(word string) (body, post string) {
	i := len(word)
	for i > 0 && isPunctPost(word[i-1]) {
		i--
	}
	if i == len(word) || i == 0 {
		return word, ""
	}
	return word[:i], word[i:]
}

// matchNum: ^[+-]?\d+$
func matchNum(s string) bool {
	if len(s) == 0 {
		return false
	}
	i := 0
	if s[0] == '+' || s[0] == '-' {
		i++
	}
	if i >= len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

// matchAddr: ^0x(\d|[a-f])+$
func matchAddr(s string) bool {
	if len(s) < 3 || s[0] != '0' || s[1] != 'x' {
		return false
	}
	for i := 2; i < len(s); i++ {
		if !isHexLower(s[i]) {
			return false
		}
	}
	return true
}

func matchMAC(s string) bool {
	if len(s) != 17 {
		return false
	}
	for i := 0; i < 17; i++ {
		if i%3 == 2 {
			if s[i] != ':' {
				return false
			}
		} else {
			if !isHexLower(s[i]) {
				return false
			}
		}
	}
	return true
}

// matchSig: ^sig(hup|int|quit|...) — hash-set lookup after "sig" prefix.
// The original regex had no $ anchor, so "sigtermed" would match via "sigterm".
func matchSig(s string) bool {
	if len(s) < 5 || s[0] != 's' || s[1] != 'i' || s[2] != 'g' {
		return false
	}
	rest := s[3:]
	// Try each prefix length.
	for end := 2; end <= 6 && end <= len(rest); end++ {
		if sigNames.Contains(rest[:end]) {
			return true
		}
	}
	return false
}

// (unanchored — matches anywhere)
func matchTime(s string) bool {
	n := len(s)
	for i := 0; i < n; i++ {
		if !isDigit(s[i]) {
			continue
		}
		// Start of potential digit group
		j := i + 1
		if j < n && isDigit(s[j]) {
			j++
		}
		if j >= n || s[j] != ':' {
			continue
		}
		j++ // skip first colon
		if j >= n || !isDigit(s[j]) {
			continue
		}
		j++
		if j < n && isDigit(s[j]) {
			j++
		}
		return true
	}
	return false
}

func matchURI(s string) bool {
	idx := strings.Index(s, "://")
	if idx < 2 {
		return false
	}
	for i := 0; i < idx; i++ {
		if !isWordChar(s[i]) {
			return false
		}
	}
	rest := s[idx+3:]
	if len(rest) == 0 {
		return false
	}
	for i := 0; i < len(rest); i++ {
		if rest[i] <= ' ' {
			return false
		}
	}
	return true
}

// (prefix match, no $ anchor)
func matchSize(s string) bool {
	i := 0
	n := len(s)
	// \d+
	if i >= n || !isDigit(s[i]) {
		return false
	}
	for i < n && isDigit(s[i]) {
		i++
	}
	// (\.\d+)?
	if i < n && s[i] == '.' {
		i++
		if i >= n || !isDigit(s[i]) {
			return false
		}
		for i < n && isDigit(s[i]) {
			i++
		}
	}
	// [kmgt]
	if i >= n {
		return false
	}
	c := s[i]
	if c != 'k' && c != 'm' && c != 'g' && c != 't' {
		return false
	}
	i++
	// i?
	if i < n && s[i] == 'i' {
		i++
	}
	// b?
	if i < n && s[i] == 'b' {
		i++
	}
	// (ytes?)?
	if i+2 < n && s[i] == 'y' && s[i+1] == 't' && s[i+2] == 'e' {
		i += 3
		if i < n && s[i] == 's' {
			i++
		}
	}
	return true
}

func matchVer(s string) bool {
	if len(s) == 0 {
		return false
	}
	i := 0
	if s[i] == 'v' {
		i++
	}
	// First group: one or more digits followed by a dot
	start := i
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i == start || i >= len(s) || s[i] != '.' {
		return false
	}
	i++ // skip dot

	// The last group must NOT end with a dot.
	for {
		start = i
		for i < len(s) && isLowerAlnum(s[i]) {
			i++
		}
		if i == start {
			return false // empty group
		}
		if i == len(s) {
			return true // consumed entire string
		}
		if s[i] != '.' {
			return false // unexpected character
		}
		i++ // skip dot, continue to next group
	}
}

// ---------------------------------------------------------------------------
// Process / ProcessOne
// ---------------------------------------------------------------------------

// Process colorizes an entire message string, writing the result to w.
// If wcol is false, the message is written with the Default color.
// If slookup is true, service/protocol/user lookups may be attempted.
func (p *Processor) Process(w io.Writer, msg string, wcol bool, slookup bool) {
	if msg == "" {
		return
	}

	if !wcol {
		p.ct.WriteColored(w, color.Default, msg)
		return
	}

	// Check for repeated message or MARK
	if (strings.Contains(msg, "last message repeated") && strings.Contains(msg, "times")) ||
		strings.Contains(msg, "-- MARK --") {
		p.ct.WriteColored(w, color.Repeat, msg)
		return
	}

	// GNU make error/fatal lines (opt-in). Color the whole line as an error —
	// the most severe level — while emitting the bytes verbatim.
	if p.ext.Make && isMakeError(msg) {
		p.ct.WriteColored(w, color.Error, msg)
		return
	}

	// Unreal Engine log prefix (opt-in).
	handledUnreal := false
	if p.ext.Unreal {
		if n := p.renderUnrealPrefix(w, msg); n > 0 {
			if n >= len(msg) {
				return
			}
			msg = msg[n:]
			handledUnreal = true
		}
	}

	// Adaptive recurring-structure recognition (opt-in).
	if !handledUnreal && p.ada != nil && p.ada.process(p, w, msg) {
		return
	}

	// Iterate space-separated words in place (no strings.Split allocation).
	start := 0
	for {
		rel := strings.IndexByte(msg[start:], ' ')
		last := rel < 0
		var word string
		if last {
			word = msg[start:]
		} else {
			word = msg[start : start+rel]
		}
		if word != "" {
			p.ProcessOne(w, word, slookup)
		}
		if last {
			return
		}
		p.ct.WriteSpace(w)
		start += rel + 1
	}
}

// ProcessOne colorizes a single word and writes it to w.
func (p *Processor) ProcessOne(w io.Writer, word string, slookup bool) {
	col := color.Default
	printed := false

	// Extract punctuation prefix
	pre, word := splitPre(word)

	// Extract punctuation suffix
	word, post := splitPost(word)

	lword := strings.ToLower(word)

	// One character-class scan gates the matchers below: each matcher can only succeed if certain bytes are present (a '.' or ':' for hosts, '@' for emails, a digit for numbers, ...).
	var hasDigit, hasDot, hasColon, hasAt, hasBracket bool
	for i := 0; i < len(lword); i++ {
		switch c := lword[i]; {
		case c >= '0' && c <= '9':
			hasDigit = true
		case c == '.':
			hasDot = true
		case c == ':':
			hasColon = true
		case c == '@':
			hasAt = true
		case c == '[':
			hasBracket = true
		}
	}

	// Opt-in extensions that need multi-color rendering and short-circuit the
	// cascade. Gated so the default path is byte-for-byte unchanged.
	if p.ext.any() {
		// [INFO] / [component] bracket tags: splitPre/splitPost have moved the
		// brackets into pre/post. A tag looks like pre ending in '[' and post
		// starting with ']' around a non-empty inner word.
		if p.ext.Tags && word != "" &&
			strings.HasSuffix(pre, "[") && strings.HasPrefix(post, "]") {
			if a, b, ok := counterParts(word); ok {
				p.renderCounter(w, pre, a, b, post)
				return
			}
			p.renderTag(w, pre, word, post)
			return
		}
		// slog / logfmt key=value pairs.
		if p.ext.Slog && p.renderKeyValue(w, pre, word, post) {
			return
		}
	}

	// Pattern cascade - order matters, first match wins (except hostip).
	// The p.ext.* cases are inert (never evaluated) when their flag is off, so
	// the default cascade is identical to the C-compatible original.
	switch {
	case p.ext.Durations && matchDuration(lword):
		col = color.GetTime
	case p.ext.Files && isBareFile(lword):
		col = color.File
	case (hasDot || hasColon || lword == "localhost") && matchHost(lword):
		col = color.Host
	case hasColon && matchMAC(lword):
		col = color.MAC
	case p.ext.Files && looksLikePath(lword):
		// Path (/abs, ./rel, ../rel, ~/, or rel/with/ext): a basename with an
		// extension is a File, otherwise a Dir.
		if fileExt(baseName(lword)) != "" {
			col = color.File
		} else {
			col = color.Dir
		}
	case len(lword) > 0 && lword[0] == '/':
		col = color.Dir
	case hasAt && matchEmail(lword) && matchEmail2(lword):
		col = color.Email
	case hasAt && matchMsgID(lword):
		col = color.Email
	case hasColon && matchURI(lword):
		col = color.URI
	case hasDigit && matchSize(lword):
		col = color.Size
	case hasDigit && hasDot && matchVer(lword):
		col = color.Version
	case hasDigit && hasColon && matchTime(lword):
		col = color.Date
	case hasDigit && matchAddr(lword):
		col = color.Address
	case hasDigit && matchNum(lword):
		col = color.Numbers
	case matchSig(lword):
		col = color.Signal
	case hasBracket && matchHostIP(lword):
		// Special handling: split at '[', output host and IP separately.
		idx := strings.Index(word, "[")
		if idx >= 0 {
			host := word[:idx]
			ip := word[idx+1:]
			p.ct.WriteColored(w, color.Default, pre)
			p.ct.WriteColored(w, color.Host, host)
			p.ct.WriteColored(w, color.PIDB, "[")
			p.ct.WriteColored(w, color.Host, ip)
			// The closing ']' lives in post (splitPost moved it there); color it as a
			// bracket and emit the remainder.
			if strings.HasPrefix(post, "]") {
				p.ct.WriteColored(w, color.PIDB, "]")
				p.ct.WriteColored(w, color.Default, post[1:])
			} else {
				p.ct.WriteColored(w, color.Default, post)
			}
			printed = true
		}
	default:
		// Service, protocol, and user lookups (slookup). These are skipped; can be
		// implemented later.

		// Keyword matching against the bad/good/error/system word lists,
		// via the first-byte index (see kwByFirst for the priority rules).
		if lword != "" {
			for _, e := range kwByFirst[lword[0]] {
				if strings.HasPrefix(lword, e.kw) {
					col = e.col
					break
				}
			}
		}
	}

	if !printed {
		p.ct.WriteColored(w, color.Default, pre)
		p.ct.WriteColored(w, col, word)
		p.ct.WriteColored(w, color.Default, post)
	}
}
