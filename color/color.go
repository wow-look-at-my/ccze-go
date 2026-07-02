// Package color implements CCZE color handling: color enum, ANSI color table,
// config file parsing, and ANSI escape sequence output.
package color

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

// Color identifies a logical color slot used throughout CCZE.
type Color int

// Color constants matching the C enum order exactly.
const (
	Date Color = iota
	Host
	Proc
	PID
	PIDB
	Default
	Email
	Subject
	Dir
	File
	Size
	User
	HTTPCodes
	GetSize
	HTTPGet
	HTTPPost
	HTTPHead
	HTTPPut
	HTTPConnect
	HTTPTrace
	Unknown
	GetTime
	URI
	Ident
	CType
	Error
	ProxyMiss
	ProxyHit
	ProxyDenied
	ProxyRefresh
	ProxySwapfail
	Debug
	Warning
	ProxyDirect
	ProxyParent
	SwapNum
	ProxyCreate
	ProxySwapin
	ProxySwapout
	ProxyRelease
	MAC
	Version
	Address
	Numbers
	Signal
	Service
	Prot
	BadWord
	GoodWord
	SystemWord
	Incoming
	Outgoing
	Uniqn
	Repeat
	Field
	Chain
	Percentage
	FTPCodes
	Keyword
	PkgStatus
	Pkg
	StaticBlack
	StaticRed
	StaticGreen
	StaticYellow
	StaticBlue
	StaticCyan
	StaticMagenta
	StaticWhite
	StaticBoldBlack
	StaticBoldRed
	StaticBoldGreen
	StaticBoldYellow
	StaticBoldBlue
	StaticBoldCyan
	StaticBoldMagenta
	StaticBoldWhite
	Last
)

// Attribute flags stored in the upper bits of color values.
const (
	AttrBold      = 0x1000
	AttrUnderline = 0x2000
	AttrReverse   = 0x4000 // C code emits SGR 5 (blink) for this - preserved
	AttrBlink     = 0x8000 // C code emits SGR 7 (reverse) for this - preserved
)

// keywordMap maps Color values to their config file keyword strings.
// Only settable (non-static) entries are included.
var keywordMap = map[Color]string{
	Date:          "date",
	Host:          "host",
	Proc:          "process",
	PID:           "pid",
	PIDB:          "pid-sqbr",
	Default:       "default",
	Email:         "email",
	Subject:       "subject",
	Dir:           "dir",
	File:          "file",
	Size:          "size",
	User:          "user",
	HTTPCodes:     "httpcodes",
	GetSize:       "getsize",
	HTTPGet:       "get",
	HTTPPost:      "post",
	HTTPHead:      "head",
	HTTPPut:       "put",
	HTTPConnect:   "connect",
	HTTPTrace:     "trace",
	Unknown:       "unknown",
	GetTime:       "gettime",
	URI:           "uri",
	Ident:         "ident",
	CType:         "ctype",
	Error:         "error",
	ProxyMiss:     "miss",
	ProxyHit:      "hit",
	ProxyDenied:   "deny",
	ProxyRefresh:  "refresh",
	ProxySwapfail: "swapfail",
	Debug:         "debug",
	Warning:       "warning",
	ProxyDirect:   "direct",
	ProxyParent:   "parent",
	SwapNum:       "swapnum",
	ProxyCreate:   "create",
	ProxySwapin:   "swapin",
	ProxySwapout:  "swapout",
	ProxyRelease:  "release",
	MAC:           "mac",
	Version:       "version",
	Address:       "address",
	Numbers:       "numbers",
	Signal:        "signal",
	Service:       "service",
	Prot:          "prot",
	BadWord:       "bad",
	GoodWord:      "good",
	SystemWord:    "system",
	Incoming:      "incoming",
	Outgoing:      "outgoing",
	Uniqn:         "uniqn",
	Repeat:        "repeat",
	Field:         "field",
	Chain:         "chain",
	Percentage:    "percentage",
	FTPCodes:      "ftpcodes",
	Keyword:       "keyword",
	PkgStatus:     "pkgstatus",
	Pkg:           "pkg",
}

// reverseKeywordMap is the keyword string -> Color lookup, built at init.
var reverseKeywordMap map[string]Color

// hiddenKeywordMap maps the static color keyword names. These are not
// settable via the normal keyword lookup (settable=true) in the C code,
// but are usable via the full keyword lookup (hiddentoo=true).
var hiddenKeywordMap = map[string]Color{
	"black":        StaticBlack,
	"red":          StaticRed,
	"green":        StaticGreen,
	"yellow":       StaticYellow,
	"blue":         StaticBlue,
	"cyan":         StaticCyan,
	"magenta":      StaticMagenta,
	"white":        StaticWhite,
	"bold_black":   StaticBoldBlack,
	"bold_red":     StaticBoldRed,
	"bold_green":   StaticBoldGreen,
	"bold_yellow":  StaticBoldYellow,
	"bold_blue":    StaticBoldBlue,
	"bold_cyan":    StaticBoldCyan,
	"bold_magenta": StaticBoldMagenta,
	"bold_white":   StaticBoldWhite,
}

func init() {
	reverseKeywordMap = make(map[string]Color, len(keywordMap))
	for c, name := range keywordMap {
		reverseKeywordMap[name] = c
	}
}

// ColorName returns the config-file keyword for a Color, or "" if none.
func ColorName(c Color) string {
	if s, ok := keywordMap[c]; ok {
		return s
	}
	return ""
}

// KeywordLookup returns the Color for a keyword string.
// It searches the settable keywords first, then the hidden (static) keywords.
func KeywordLookup(name string) (Color, bool) {
	if c, ok := reverseKeywordMap[name]; ok {
		return c, true
	}
	if c, ok := hiddenKeywordMap[name]; ok {
		return c, true
	}
	return 0, false
}

// settableKeywordLookup returns the Color for a settable-only keyword.
func settableKeywordLookup(name string) (Color, bool) {
	c, ok := reverseKeywordMap[name]
	return c, ok
}

// ansiColor maps internal color index (0-7) to ANSI SGR color codes.
// Note: cyan (index 5) and magenta (index 6) are swapped, matching the C code.
var ansiColor = [8]int{30, 31, 32, 33, 34, 36, 35, 37}

// colorNameValues maps color name strings to their numeric values (0-7).
var colorNameValues = map[string]int{
	"black":   0,
	"red":     1,
	"green":   2,
	"yellow":  3,
	"blue":    4,
	"cyan":    5,
	"magenta": 6,
	"white":   7,
}

// bgColorNameValues maps background color name strings to numeric values (0-7).
var bgColorNameValues = map[string]int{
	"on_black":   0,
	"on_red":     1,
	"on_green":   2,
	"on_yellow":  3,
	"on_blue":    4,
	"on_cyan":    5,
	"on_magenta": 6,
	"on_white":   7,
}

// Table holds the color values for all color slots.
type Table struct {
	colors      [Last + 1]int
	transparent bool

	// prefixes caches the ANSI escape prefix emitted before text for each
	// color slot ("" = not built yet). A slot's prefix depends only on its
	// color value and the transparent flag, so it is built lazily on first
	// use and invalidated whenever the slot's color changes.
	prefixes [Last + 1]string
	// spaceSeq caches the full escape sequence for a Default-colored space.
	spaceSeq string
}

// NewTable creates a Table with all default colors initialized, matching
// ccze_color_init_raw_ansi() in the C code.
func NewTable(transparent bool) *Table {
	t := &Table{transparent: transparent}

	t.colors[Date] = AttrBold | 5
	t.colors[Host] = AttrBold | 4
	t.colors[Proc] = 2
	t.colors[PID] = AttrBold | 7
	t.colors[PIDB] = AttrBold | 2
	t.colors[Default] = 5
	t.colors[Email] = AttrBold | 2
	t.colors[Subject] = 6
	t.colors[Dir] = AttrBold | 5
	t.colors[File] = t.colors[Dir]
	t.colors[Size] = AttrBold | 7
	t.colors[User] = AttrBold | 3
	t.colors[HTTPCodes] = AttrBold | 7
	t.colors[GetSize] = 6
	t.colors[HTTPGet] = 2
	t.colors[HTTPPost] = AttrBold | 2
	t.colors[HTTPHead] = 2
	t.colors[HTTPPut] = AttrBold | 2
	t.colors[HTTPConnect] = 2
	t.colors[HTTPTrace] = 2
	t.colors[Unknown] = t.colors[Default]
	t.colors[GetTime] = AttrBold | 6
	t.colors[URI] = AttrBold | 2
	t.colors[Ident] = AttrBold | 7
	t.colors[CType] = 7
	t.colors[Error] = AttrBold | 1
	t.colors[ProxyMiss] = 1
	t.colors[ProxyHit] = AttrBold | 3
	t.colors[ProxyDenied] = AttrBold | 1
	t.colors[ProxyRefresh] = AttrBold | 7
	t.colors[ProxySwapfail] = AttrBold | 7
	t.colors[Debug] = 7
	t.colors[Warning] = 1
	t.colors[ProxyDirect] = AttrBold | 7
	t.colors[ProxyParent] = AttrBold | 3
	t.colors[SwapNum] = t.colors[Default]
	t.colors[ProxyCreate] = AttrBold | 7
	t.colors[ProxySwapin] = AttrBold | 7
	t.colors[ProxySwapout] = AttrBold | 7
	t.colors[ProxyRelease] = AttrBold | 7
	t.colors[MAC] = AttrBold | 7
	t.colors[Version] = AttrBold | 7
	t.colors[Address] = AttrBold | 7
	t.colors[Numbers] = 7
	t.colors[Signal] = AttrBold | 3
	t.colors[Service] = AttrBold | 6
	t.colors[Prot] = 6
	t.colors[BadWord] = AttrBold | 3
	t.colors[GoodWord] = AttrBold | 2
	t.colors[SystemWord] = AttrBold | 5
	t.colors[Incoming] = AttrBold | 7
	t.colors[Outgoing] = 7
	t.colors[Uniqn] = AttrBold | 7
	t.colors[Repeat] = 7
	t.colors[Field] = 2
	t.colors[Chain] = 5
	t.colors[Percentage] = AttrBold | 3
	t.colors[FTPCodes] = 5
	t.colors[Keyword] = AttrBold | 3
	t.colors[PkgStatus] = 2
	t.colors[Pkg] = AttrBold | 1

	t.colors[StaticBlack] = 0
	t.colors[StaticRed] = 1
	t.colors[StaticGreen] = 2
	t.colors[StaticYellow] = 3
	t.colors[StaticBlue] = 4
	t.colors[StaticCyan] = 5
	t.colors[StaticMagenta] = 6
	t.colors[StaticWhite] = 7
	t.colors[StaticBoldBlack] = AttrBold | 0
	t.colors[StaticBoldRed] = AttrBold | 1
	t.colors[StaticBoldGreen] = AttrBold | 2
	t.colors[StaticBoldYellow] = AttrBold | 3
	t.colors[StaticBoldBlue] = AttrBold | 4
	t.colors[StaticBoldCyan] = AttrBold | 5
	t.colors[StaticBoldMagenta] = AttrBold | 6
	t.colors[StaticBoldWhite] = AttrBold | 7

	t.colors[Last] = 5

	return t
}

// Get returns the raw color value for a given Color slot.
func (t *Table) Get(c Color) int {
	return t.colors[c]
}

// Set sets the raw color value for a given Color slot.
func (t *Table) Set(c Color, v int) {
	t.colors[c] = v
	t.prefixes[c] = ""
	t.spaceSeq = ""
}

// Transparent returns whether the table is in transparent mode.
func (t *Table) Transparent() bool {
	return t.transparent
}

// buildPrefix constructs the ANSI escape prefix for a color slot, matching
// the C ccze_addstr_internal for RAW_ANSI mode exactly. The escape sequence
// generation preserves the C code's swap where AttrReverse emits SGR 5
// (blink) and AttrBlink emits SGR 7 (reverse).
func (t *Table) buildPrefix(col Color) string {
	c := t.colors[col]

	buf := make([]byte, 0, 24)

	// ESC[22m - reset to normal intensity
	buf = append(buf, "\x1b[22m"...)

	// Bold
	if c&AttrBold != 0 {
		buf = append(buf, "\x1b[1m"...)
	}

	// Underline
	if c&AttrUnderline != 0 {
		buf = append(buf, "\x1b[4m"...)
	}

	// Reverse flag -> SGR 5 (blink) - preserving the C behavior
	if c&AttrReverse != 0 {
		buf = append(buf, "\x1b[5m"...)
	}

	// Blink flag -> SGR 7 (reverse) - preserving the C behavior
	if c&AttrBlink != 0 {
		buf = append(buf, "\x1b[7m"...)
	}

	// Strip attribute bits to get color portion
	cv := c & 0x0FFF

	// Background color: emit if bg > 0 or not transparent
	bg := cv >> 8
	if bg > 0 || !t.transparent {
		buf = append(buf, "\x1b["...)
		buf = strconv.AppendInt(buf, int64(ansiColor[bg]+10), 10)
		buf = append(buf, 'm')
	}

	// Foreground color
	fg := cv & 0xf
	buf = append(buf, "\x1b["...)
	buf = strconv.AppendInt(buf, int64(ansiColor[fg]), 10)
	buf = append(buf, 'm')

	return string(buf)
}

// prefix returns the (lazily built) ANSI escape prefix for a color slot.
func (t *Table) prefix(col Color) string {
	p := t.prefixes[col]
	if p == "" {
		p = t.buildPrefix(col)
		t.prefixes[col] = p
	}
	return p
}

// WriteColored writes text to w wrapped in ANSI escape sequences. The bytes
// written are identical to the previous fmt-based implementation (and to C
// ccze's RAW_ANSI output); the escape prefix is simply cached per color slot
// instead of being re-formatted for every word.
func (t *Table) WriteColored(w io.Writer, col Color, text string) {
	if text == "" {
		return
	}
	io.WriteString(w, t.prefix(col))
	io.WriteString(w, text)
	io.WriteString(w, "\x1b[0m")
}

// WriteSpace writes a space character colored with the Default color.
func (t *Table) WriteSpace(w io.Writer) {
	if t.spaceSeq == "" {
		t.spaceSeq = t.prefix(Default) + " \x1b[0m"
	}
	io.WriteString(w, t.spaceSeq)
}

// WriteNewline writes a newline character to the writer.
func (t *Table) WriteNewline(w io.Writer) {
	io.WriteString(w, "\n")
}

// LoadFile reads a config file and parses each line for color settings.
// If the file does not exist or cannot be read, the error is silently ignored
// (matching C behavior which just returns on fopen failure).
func (t *Table) LoadFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t.ParseLine(scanner.Text())
	}
	return scanner.Err()
}

// ParseLine parses a single color configuration line of the form:
//
//	keyword [bold|underline|reverse|blink] color [on_bgcolor]
//
// where color is one of: black, red, green, yellow, blue, cyan, magenta, white
// or a quoted keyword like 'date' (meaning "use the same color as that keyword").
// Background is specified with on_black, on_red, etc.
// Lines with css-prefixed keywords are skipped (not relevant for ANSI mode).
func (t *Table) ParseLine(line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}

	keyword := fields[0]

	// Skip css keywords entirely
	if strings.HasPrefix(keyword, "css") {
		return
	}

	// Strip inline comments
	for i, f := range fields {
		if strings.HasPrefix(f, "#") {
			fields = fields[:i]
			break
		}
	}

	// The keyword may use = as separator: "keyword=color" or "keyword color"
	if idx := strings.Index(keyword, "="); idx >= 0 {
		parts := []string{keyword[:idx]}
		if idx+1 < len(keyword) {
			parts = append(parts, keyword[idx+1:])
		}
		fields = append(parts, fields[1:]...)
		keyword = fields[0]
	}

	nkeyword, ok := settableKeywordLookup(keyword)
	if !ok {
		return
	}

	if len(fields) < 2 {
		return
	}

	pos := 1
	var pre string

	// Check for attribute prefix
	if fields[pos] == "bold" || fields[pos] == "underline" ||
		fields[pos] == "reverse" || fields[pos] == "blink" {
		pre = fields[pos]
		pos++
	}

	if pos >= len(fields) {
		return
	}

	colorStr := fields[pos]
	pos++

	ncolor := t.lookupColorName(colorStr)
	if ncolor == -1 {
		return
	}

	// Check for background color
	if pos < len(fields) {
		bgStr := fields[pos]
		if nbg, bgOK := bgColorNameValues[bgStr]; bgOK {
			ncolor += nbg << 8
		}
	}

	// The resolved color value
	rcolor := ncolor

	// Apply attribute
	if pre != "" {
		switch pre {
		case "bold":
			rcolor |= AttrBold
		case "underline":
			rcolor |= AttrUnderline
		case "reverse":
			rcolor |= AttrReverse
		case "blink":
			rcolor |= AttrBlink
		}
	}

	t.Set(nkeyword, rcolor)
}

// lookupColorName resolves a color name string to its numeric value.
// If the name is a single-quoted keyword like 'date', it resolves to the
// current color value of that keyword in the table. Otherwise it looks up
// the name in the standard color name map. Returns -1 on failure.
func (t *Table) lookupColorName(name string) int {
	if len(name) > 2 && name[0] == '\'' && name[len(name)-1] == '\'' {
		inner := name[1 : len(name)-1]
		c, ok := settableKeywordLookup(inner)
		if !ok {
			return -1
		}
		return t.colors[c]
	}

	if v, ok := colorNameValues[name]; ok {
		return v
	}
	return -1
}
