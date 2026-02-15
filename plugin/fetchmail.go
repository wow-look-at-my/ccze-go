package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// FetchmailPlugin is a PARTIAL plugin.
// Coloriser for fetchmail(1) sub-logs.
type FetchmailPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewFetchmailPlugin creates a new FetchmailPlugin.
func NewFetchmailPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *FetchmailPlugin {
	return &FetchmailPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *FetchmailPlugin) Name() string        { return "fetchmail" }
func (p *FetchmailPlugin) Type() Type           { return TypePartial }
func (p *FetchmailPlugin) Description() string  { return "Coloriser for fetchmail(1) sub-logs." }

// parseFetchmail hand-parses a fetchmail log line.
// Format: (reading message) ([^@]*@[^:]*):([0-9]*) of ([0-9]*) (.*)
func parseFetchmail(line string) (addy, current, full, rest string, ok bool) {
	// Find "reading message " anywhere in the line
	idx := strings.Index(line, "reading message ")
	if idx < 0 {
		return
	}
	s := line[idx+16:] // skip "reading message "

	// addy: [^@]*@[^:]* — everything up to ':'
	colonIdx := strings.Index(s, ":")
	if colonIdx < 0 {
		return
	}
	addy = s[:colonIdx]
	// Validate addy contains @
	if !strings.Contains(addy, "@") {
		return
	}
	s = s[colonIdx+1:]

	// current: [0-9]*
	numEnd := 0
	for numEnd < len(s) && s[numEnd] >= '0' && s[numEnd] <= '9' {
		numEnd++
	}
	current = s[:numEnd]
	s = s[numEnd:]

	// " of "
	if !strings.HasPrefix(s, " of ") {
		return
	}
	s = s[4:]

	// full: [0-9]*
	numEnd = 0
	for numEnd < len(s) && s[numEnd] >= '0' && s[numEnd] <= '9' {
		numEnd++
	}
	full = s[:numEnd]
	s = s[numEnd:]

	// " " + rest
	if len(s) < 1 || s[0] != ' ' {
		return
	}
	rest = s[1:]

	ok = true
	return
}

// Handle attempts to match and colorize a fetchmail log line.
func (p *FetchmailPlugin) Handle(line string) (bool, string) {
	addy, current, full, rest, ok := parseFetchmail(line)
	if !ok {
		return false, ""
	}

	p.ct.WriteColored(p.w, color.Default, "reading message")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Email, addy)
	p.ct.WriteColored(p.w, color.Default, ":")
	p.ct.WriteColored(p.w, color.Numbers, current)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Default, "of")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Numbers, full)
	p.ct.WriteSpace(p.w)

	return true, rest
}
