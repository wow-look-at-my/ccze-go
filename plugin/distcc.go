package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

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
func (p *DistccPlugin) Type() Type           { return TypeFull }
func (p *DistccPlugin) Description() string  { return "Coloriser for distcc(1) logs." }

// parseDistcc hand-parses a distcc log line.
// Format: ^distccd\[(\d+)\] (\([^\)]+\))? ?(.*)
func parseDistcc(line string) (pid, funcName, rest string, ok bool) {
	if !strings.HasPrefix(line, "distccd[") {
		return
	}
	s := line[8:] // skip "distccd["

	// PID: \d+
	pidEnd := 0
	for pidEnd < len(s) && s[pidEnd] >= '0' && s[pidEnd] <= '9' {
		pidEnd++
	}
	if pidEnd == 0 {
		return
	}
	pid = s[:pidEnd]
	s = s[pidEnd:]

	// ] followed by space
	if len(s) < 2 || s[0] != ']' || s[1] != ' ' {
		return
	}
	s = s[2:]

	// Optional function name: (\([^\)]+\))?
	if len(s) > 0 && s[0] == '(' {
		closeIdx := strings.Index(s, ")")
		if closeIdx > 1 {
			funcName = s[:closeIdx+1]
			s = s[closeIdx+1:]
			// Optional space after function name
			if len(s) > 0 && s[0] == ' ' {
				s = s[1:]
			}
		}
	}

	rest = s
	ok = true
	return
}

// Handle attempts to match and colorize a distcc log line.
func (p *DistccPlugin) Handle(line string) (bool, string) {
	pid, funcName, rest, ok := parseDistcc(line)
	if !ok {
		return false, ""
	}

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
