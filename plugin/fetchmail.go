package plugin

import (
	"io"

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

// Handle attempts to match and colorize a fetchmail log line.
func (p *FetchmailPlugin) Handle(line string) (bool, string) {
	m := fetchmailFindSubmatch(line)
	if m == nil {
		return false, ""
	}

	addy := m[2]
	current := m[3]
	full := m[4]
	rest := m[5]

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
