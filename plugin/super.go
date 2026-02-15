package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var superRe = regexp.MustCompile(
	`^(\S+)\s(\w+\s+\w+\s+\d+\s+\d+:\d+:\d+\s+\d+)(\s+)(\S+)\s\(([^\)]+)\)`,
)

// SuperPlugin is a FULL plugin.
// Coloriser for super(1) logs.
type SuperPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewSuperPlugin creates a new SuperPlugin.
func NewSuperPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SuperPlugin {
	return &SuperPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *SuperPlugin) Name() string        { return "super" }
func (p *SuperPlugin) Type() Type          { return TypeFull }
func (p *SuperPlugin) Description() string { return "Coloriser for super(1) logs." }

// Handle attempts to match and colorize a super log line.
func (p *SuperPlugin) Handle(line string) (bool, string) {
	m := superRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	email := m[1]
	date := m[2]
	space := m[3]
	suptag := m[4]
	other := m[5]

	p.ct.WriteColored(p.w, color.Email, email)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteColored(p.w, color.Default, space)
	p.ct.WriteColored(p.w, color.Proc, suptag)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.PIDB, "(")
	p.ct.WriteColored(p.w, color.Default, other)
	p.ct.WriteColored(p.w, color.PIDB, ")")

	p.ct.WriteNewline(p.w)

	return true, ""
}
