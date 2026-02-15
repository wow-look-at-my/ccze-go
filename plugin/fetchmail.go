package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var fetchmailRe = regexp.MustCompile(`(reading message) ([^@]*@[^:]*):([0-9]*) of ([0-9]*) (.*)`)

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
func (p *FetchmailPlugin) Type() Type          { return TypePartial }
func (p *FetchmailPlugin) Description() string { return "Coloriser for fetchmail(1) sub-logs." }

// Handle attempts to match and colorize a fetchmail log line.
func (p *FetchmailPlugin) Handle(line string) (bool, string) {
	m := fetchmailRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	start := m[1]
	addy := m[2]
	current := m[3]
	full := m[4]
	rest := m[5]

	p.ct.WriteColored(p.w, color.Default, start)
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
