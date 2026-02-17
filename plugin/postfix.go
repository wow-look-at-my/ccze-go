package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// PostfixPlugin colorizes postfix(1) sub-log lines.
type PostfixPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewPostfixPlugin creates a new PostfixPlugin.
func NewPostfixPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *PostfixPlugin {
	return &PostfixPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *PostfixPlugin) Name() string        { return "postfix" }
func (p *PostfixPlugin) Type() Type           { return TypePartial }
func (p *PostfixPlugin) Description() string  { return "Coloriser for postfix(1) sub-logs." }

// postfixProcessOne processes a single field=value segment. Returns true if
// the segment did not contain '=' (was not a field=value pair).
func (p *PostfixPlugin) postfixProcessOne(s string) bool {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return true
	}

	field := s[:idx]
	value := s[idx+1:]

	p.ct.WriteColored(p.w, color.Field, field)
	p.ct.WriteColored(p.w, color.Default, "=")
	p.wc.ProcessOne(p.w, value, true)

	return false
}

func (p *PostfixPlugin) Handle(line string) (bool, string) {
	m := postfixFindSubmatch(line)
	if m == nil {
		return false, ""
	}

	spoolid := m[1]
	s := m[2]

	p.ct.WriteColored(p.w, color.Uniqn, spoolid)
	p.ct.WriteColored(p.w, color.Default, ": ")

	// Split on commas and process each segment
	parts := strings.Split(s, ",")
	for i, part := range parts {
		notField := p.postfixProcessOne(part)
		if notField {
			p.ct.WriteColored(p.w, color.Default, part)
			// C code breaks out of loop when process_one returns 1
			break
		}
		if i < len(parts)-1 {
			p.ct.WriteColored(p.w, color.Default, ",")
		}
	}

	return true, ""
}
