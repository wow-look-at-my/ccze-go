package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var phpRe = regexp.MustCompile(`^(\[\d+-...-\d+ \d+:\d+:\d+\]) PHP (.*)$`)

// PHPPlugin colorizes PHP log lines.
type PHPPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewPHPPlugin creates a new PHPPlugin.
func NewPHPPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *PHPPlugin {
	return &PHPPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *PHPPlugin) Name() string        { return "php" }
func (p *PHPPlugin) Type() Type          { return TypeFull }
func (p *PHPPlugin) Description() string { return "Coloriser for PHP logs." }

func (p *PHPPlugin) Handle(line string) (bool, string) {
	m := phpRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	rest := m[2]

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Keyword, "PHP")
	p.ct.WriteSpace(p.w)

	return true, rest
}
