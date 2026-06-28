// Package wordcolor — unreal.go.
//
// This file holds the opt-in Unreal Engine log highlighter. Unreal's default
// log line looks like:
//
//	[2026.06.28-13.42.12:957][  0]LogCoreRedirects: Verbose: <message>
//
// i.e. a [YYYY.MM.DD-HH.MM.SS:mmm] timestamp, an optional [frame] counter, a
// LogCategory, and an optional Verbosity. None of these are space-separated
// from each other (the brackets abut), so the per-word cascade in wordcolor.go
// cannot tokenize them — they need a whole-line pre-pass. renderUnrealPrefix
// colors the prefix and returns the byte offset of the message body, which the
// caller (Process) then colors with the normal per-word path.
//
// Like every extension here it only ADDS color: the prefix bytes are emitted
// verbatim, just wrapped in ANSI. It is gated behind Extensions.Unreal and is
// never reached on the default C-compatible path.
package wordcolor

import (
	"io"
	"strings"

	"ccze-go/color"
)

// unrealVerbosities is the set of Unreal Engine verbosity levels (lowercased).
// A "Word:" token immediately after the category is only treated as a verbosity
// when it is one of these; otherwise it is left to the message body. This keeps
// e.g. "LogTemp: SomeFunction: ..." from mis-coloring SomeFunction.
var unrealVerbosities = map[string]bool{
	"fatal":       true,
	"error":       true,
	"warning":     true,
	"display":     true,
	"log":         true,
	"verbose":     true,
	"veryverbose": true,
}

// unrealVerbosityColor maps an Unreal verbosity word (any case) to a color,
// brightest for the most severe levels and dim for the debug levels.
func unrealVerbosityColor(v string) color.Color {
	switch strings.ToLower(v) {
	case "fatal", "error":
		return color.Error
	case "warning":
		return color.Warning
	case "display":
		return color.GoodWord
	default: // log, verbose, veryverbose
		return color.Debug
	}
}

// matchUnrealTimestamp matches an Unreal "[YYYY.MM.DD-HH.MM.SS:mmm]" timestamp
// at the start of s, returning its length in bytes (brackets included) or 0.
// The structural signature "[d.d.d-d.d.d:d]" (each group 1-4 digits) is
// specific enough that it does not collide with other bracketed prefixes.
func matchUnrealTimestamp(s string) int {
	if len(s) == 0 || s[0] != '[' {
		return 0
	}
	i := 1
	digits := func() bool {
		start := i
		for i < len(s) && i-start < 4 && isDigit(s[i]) {
			i++
		}
		return i > start
	}
	sep := func(c byte) bool {
		if i < len(s) && s[i] == c {
			i++
			return true
		}
		return false
	}
	if !digits() || !sep('.') || !digits() || !sep('.') || !digits() ||
		!sep('-') || !digits() || !sep('.') || !digits() || !sep('.') ||
		!digits() || !sep(':') || !digits() {
		return 0
	}
	if i >= len(s) || s[i] != ']' {
		return 0
	}
	return i + 1
}

// matchUnrealFrame matches an Unreal "[  N]" frame counter (optional leading
// spaces then one or more digits) at the start of s, returning its length in
// bytes (brackets included) or 0.
func matchUnrealFrame(s string) int {
	if len(s) < 3 || s[0] != '[' {
		return 0
	}
	i := 1
	for i < len(s) && s[i] == ' ' {
		i++
	}
	d := i
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i == d || i >= len(s) || s[i] != ']' {
		return 0
	}
	return i + 1
}

// matchUnrealCategory scans a leading identifier ([A-Za-z_][A-Za-z0-9_]*) that
// is immediately followed by ':'. It returns the identifier length (excluding
// the colon) and whether it looks like a real Unreal category (PascalCase
// starting with "Log", e.g. LogTemp, LogCoreRedirects). Returns 0 when there is
// no identifier directly followed by a colon.
func matchUnrealCategory(s string) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}
	c := s[0]
	if !(isLowerAlpha(c) || (c >= 'A' && c <= 'Z') || c == '_') {
		return 0, false
	}
	i := 1
	for i < len(s) && isWordChar(s[i]) {
		i++
	}
	if i >= len(s) || s[i] != ':' {
		return 0, false
	}
	isLog := i >= 4 && s[0] == 'L' && s[1] == 'o' && s[2] == 'g' &&
		s[3] >= 'A' && s[3] <= 'Z'
	return i, isLog
}

// matchUnrealVerbosity checks for a " Verbosity:" token at the start of s: a
// single leading space, a known Unreal verbosity word, then a colon. It returns
// the total length consumed (space + word + colon) or 0.
func matchUnrealVerbosity(s string) int {
	if len(s) < 2 || s[0] != ' ' {
		return 0
	}
	start := 1
	i := start
	for i < len(s) && isWordChar(s[i]) {
		i++
	}
	if i == start || i >= len(s) || s[i] != ':' {
		return 0
	}
	if !unrealVerbosities[strings.ToLower(s[start:i])] {
		return 0
	}
	return i + 1
}

// writeUnrealTimestamp colors "[YYYY.MM.DD-HH.MM.SS:mmm]": brackets as PIDB,
// the date as Date, the '-' separator as Default, and the time as GetTime.
func (p *Processor) writeUnrealTimestamp(w io.Writer, tok string) {
	p.ct.WriteColored(w, color.PIDB, "[")
	inner := tok[1 : len(tok)-1]
	if dash := strings.IndexByte(inner, '-'); dash >= 0 {
		p.ct.WriteColored(w, color.Date, inner[:dash])
		p.ct.WriteColored(w, color.Default, "-")
		p.ct.WriteColored(w, color.GetTime, inner[dash+1:])
	} else {
		p.ct.WriteColored(w, color.Date, inner)
	}
	p.ct.WriteColored(w, color.PIDB, "]")
}

// writeUnrealFrame colors "[  N]": brackets as PIDB, the inner (spaces and
// digits) as Numbers.
func (p *Processor) writeUnrealFrame(w io.Writer, tok string) {
	p.ct.WriteColored(w, color.PIDB, "[")
	p.ct.WriteColored(w, color.Numbers, tok[1:len(tok)-1])
	p.ct.WriteColored(w, color.PIDB, "]")
}

// renderUnrealPrefix colors a leading Unreal Engine log prefix
// ("[timestamp][frame]Category: Verbosity:") and returns the number of input
// bytes it consumed, or 0 if the line does not look like an Unreal log line (in
// which case it writes nothing). Each part is optional, but at least one anchor
// — a timestamp, a frame counter, or a Log-prefixed category — must be present.
// The caller colors the remaining message body with the normal per-word path.
// Stripped of ANSI, the written bytes equal msg[:returned].
func (p *Processor) renderUnrealPrefix(w io.Writer, msg string) int {
	pos := 0
	anchor := false

	if n := matchUnrealTimestamp(msg[pos:]); n > 0 {
		p.writeUnrealTimestamp(w, msg[pos:pos+n])
		pos += n
		anchor = true
	}
	if n := matchUnrealFrame(msg[pos:]); n > 0 {
		p.writeUnrealFrame(w, msg[pos:pos+n])
		pos += n
		anchor = true
	}
	if n, isLog := matchUnrealCategory(msg[pos:]); n > 0 && (anchor || isLog) {
		p.ct.WriteColored(w, color.Keyword, msg[pos:pos+n])
		p.ct.WriteColored(w, color.Default, ":")
		pos += n + 1
		anchor = true

		if v := matchUnrealVerbosity(msg[pos:]); v > 0 {
			word := msg[pos+1 : pos+v-1]
			p.ct.WriteColored(w, color.Default, " ")
			p.ct.WriteColored(w, unrealVerbosityColor(word), word)
			p.ct.WriteColored(w, color.Default, ":")
			pos += v
		}
	}

	if !anchor {
		return 0
	}
	return pos
}
