package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// UlogdPlugin is a PARTIAL plugin.
// Coloriser for ulogd sub-logs.
type UlogdPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewUlogdPlugin creates a new UlogdPlugin.
func NewUlogdPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *UlogdPlugin {
	return &UlogdPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *UlogdPlugin) Name() string        { return "ulogd" }
func (p *UlogdPlugin) Type() Type          { return TypePartial }
func (p *UlogdPlugin) Description() string { return "Coloriser for ulogd sub-logs." }

// ulogd netfilter keywords that trigger a match.
var ulogdKeywords = []string{"IN=", "OUT=", "MAC=", "TTL=", "SRC=", "TOS=", "PREC=", "SPT="}

// Handle attempts to match and colorize a ulogd log line.
// If the line contains netfilter keywords, it splits on spaces and
// colorizes field=value pairs individually.
func (p *UlogdPlugin) Handle(line string) (bool, string) {
	// Check for any netfilter keyword (replaces regex match)
	found := false
	for _, kw := range ulogdKeywords {
		if strings.Contains(line, kw) {
			found = true
			break
		}
	}
	if !found {
		return false, ""
	}

	words := strings.Split(line, " ")
	for i, word := range words {
		if word == "" {
			continue
		}
		if idx := strings.Index(word, "="); idx >= 0 {
			field := word[:idx]
			value := word[idx+1:]
			p.ct.WriteColored(p.w, color.Field, field)
			p.ct.WriteColored(p.w, color.Default, "=")
			p.wc.ProcessOne(p.w, value, true)
		} else {
			p.ct.WriteColored(p.w, color.Field, word)
		}
		if i < len(words)-1 {
			p.ct.WriteSpace(p.w)
		}
	}

	return true, ""
}
