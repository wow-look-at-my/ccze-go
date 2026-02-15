package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// SulogPlugin is a FULL plugin.
// Coloriser for su(1) logs.
type SulogPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
	re       *regexp.Regexp
}

// NewSulogPlugin creates a new SulogPlugin.
func NewSulogPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SulogPlugin {
	return &SulogPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
		re: regexp.MustCompile(
			`^SU (\d{2}/\d{2} \d{2}:\d{2}) ([+\-]) (\S+) ([^\-]+)-(.*)$`,
		),
	}
}

func (p *SulogPlugin) Name() string        { return "sulog" }
func (p *SulogPlugin) Type() Type           { return TypeFull }
func (p *SulogPlugin) Description() string  { return "Coloriser for su(1) logs." }

// Handle attempts to match and colorize a sulog line.
func (p *SulogPlugin) Handle(line string) (bool, string) {
	m := p.re.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	islogin := m[2]
	tty := m[3]
	fromuser := m[4]
	touser := m[5]

	p.ct.WriteColored(p.w, color.Default, "SU ")
	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Default, islogin)
	p.ct.WriteSpace(p.w)

	if len(tty) > 0 && tty[0] == '?' {
		p.ct.WriteColored(p.w, color.Unknown, tty)
	} else {
		p.ct.WriteColored(p.w, color.Dir, tty)
	}

	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.User, fromuser)
	p.ct.WriteColored(p.w, color.Default, "-")
	p.ct.WriteColored(p.w, color.User, touser)

	p.ct.WriteNewline(p.w)

	return true, ""
}
