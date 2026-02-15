package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var distccRe = regexp.MustCompile(`^distccd\[(\d+)\] (\([^\)]+\))? ?(.*)`)

// DistccPlugin is a FULL plugin.
// Coloriser for distcc(1) logs.
type DistccPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewDistccPlugin creates a new DistccPlugin.
func NewDistccPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *DistccPlugin {
	return &DistccPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *DistccPlugin) Name() string        { return "distcc" }
func (p *DistccPlugin) Type() Type          { return TypeFull }
func (p *DistccPlugin) Description() string { return "Coloriser for distcc(1) logs." }

// Handle attempts to match and colorize a distcc log line.
func (p *DistccPlugin) Handle(line string) (bool, string) {
	m := distccRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	pid := m[1]
	funcName := m[2]
	rest := m[3]

	p.ct.WriteColored(p.w, color.Proc, "distccd")
	p.ct.WriteColored(p.w, color.PIDB, "[")
	p.ct.WriteColored(p.w, color.PID, pid)
	p.ct.WriteColored(p.w, color.PIDB, "]")
	p.ct.WriteSpace(p.w)

	if funcName != "" {
		p.ct.WriteColored(p.w, color.Keyword, funcName)
		p.ct.WriteSpace(p.w)
	}

	return true, rest
}
