package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var oopsRe = regexp.MustCompile(
	`^((Mon|Tue|Wed|Thu|Fri|Sat|Sun) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d+ \d+:\d+:\d+ \d+)(\s+)\[([\dxa-fA-F]+)\]statistics\(\): ([\S]+)(\s*): (\d+)(.*)`,
)

// OopsPlugin is a FULL plugin.
// Coloriser for oops/proxy logs.
type OopsPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewOopsPlugin creates a new OopsPlugin.
func NewOopsPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *OopsPlugin {
	return &OopsPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *OopsPlugin) Name() string        { return "oops" }
func (p *OopsPlugin) Type() Type          { return TypeFull }
func (p *OopsPlugin) Description() string { return "Coloriser for oops proxy logs." }

// Handle attempts to match and colorize an oops log line.
func (p *OopsPlugin) Handle(line string) (bool, string) {
	// Prefilter: the regexp starts ^(Mon|Tue|Wed|Thu|Fri|Sat|Sun) followed
	// by a space. Necessary condition only - never rejects a match.
	if len(line) < 4 || line[3] != ' ' {
		return false, ""
	}
	switch line[:3] {
	case "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun":
	default:
		return false, ""
	}
	m := oopsRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	// m[2] = day of week (part of date group)
	// m[3] = month (part of date group)
	sp1 := m[4]
	id := m[5]
	field := m[6]
	sp2 := m[7]
	value := m[8]
	etc := m[9]

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteColored(p.w, color.Default, sp1)

	p.ct.WriteColored(p.w, color.PIDB, "[")
	p.ct.WriteColored(p.w, color.Proc, id)
	p.ct.WriteColored(p.w, color.PIDB, "]")

	p.ct.WriteColored(p.w, color.Keyword, "statistics()")
	p.ct.WriteColored(p.w, color.Default, ":")
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.Field, field)
	p.ct.WriteColored(p.w, color.Default, sp2)
	p.ct.WriteColored(p.w, color.Default, ":")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Numbers, value)
	p.ct.WriteColored(p.w, color.Default, etc)

	p.ct.WriteNewline(p.w)

	return true, ""
}
