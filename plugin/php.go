package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

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
func (p *PHPPlugin) Type() Type           { return TypeFull }
func (p *PHPPlugin) Description() string  { return "Coloriser for PHP logs." }

// parsePHP hand-parses a PHP log line.
// Format: ^(\[\d+-...-\d+ \d+:\d+:\d+\]) PHP (.*)$
func parsePHP(line string) (date, rest string, ok bool) {
	if len(line) == 0 || line[0] != '[' {
		return
	}

	// Find closing bracket followed by "] PHP "
	closeIdx := strings.Index(line, "] PHP ")
	if closeIdx < 1 {
		return
	}

	// Validate date portion: starts with digit after [
	if line[1] < '0' || line[1] > '9' {
		return
	}

	date = line[:closeIdx+1] // includes the [ and ]
	rest = line[closeIdx+6:] // skip "] PHP "

	ok = true
	return
}

func (p *PHPPlugin) Handle(line string) (bool, string) {
	date, rest, ok := parsePHP(line)
	if !ok {
		return false, ""
	}

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Keyword, "PHP")
	p.ct.WriteSpace(p.w)

	return true, rest
}
