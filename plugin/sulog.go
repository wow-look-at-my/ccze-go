package plugin

import (
	"io"
	"strings"

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
}

// NewSulogPlugin creates a new SulogPlugin.
func NewSulogPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SulogPlugin {
	return &SulogPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *SulogPlugin) Name() string        { return "sulog" }
func (p *SulogPlugin) Type() Type          { return TypeFull }
func (p *SulogPlugin) Description() string { return "Coloriser for su(1) logs." }

// parseSulog hand-parses an su log line.
// Format: ^SU (\d{2}/\d{2} \d{2}:\d{2}) ([+-]) (\S+) ([^-]+)-(.*)$
func parseSulog(line string) (date, islogin, tty, fromuser, touser string, ok bool) {
	if !strings.HasPrefix(line, "SU ") {
		return
	}
	s := line[3:]

	// Date: dd/dd dd:dd (11 chars)
	if len(s) < 12 {
		return
	}
	d := s[:11]
	if !(d[0] >= '0' && d[0] <= '9' && d[1] >= '0' && d[1] <= '9' && d[2] == '/' &&
		d[3] >= '0' && d[3] <= '9' && d[4] >= '0' && d[4] <= '9' && d[5] == ' ' &&
		d[6] >= '0' && d[6] <= '9' && d[7] >= '0' && d[7] <= '9' && d[8] == ':' &&
		d[9] >= '0' && d[9] <= '9' && d[10] >= '0' && d[10] <= '9') {
		return
	}
	date = d
	s = s[11:]

	// Space
	if len(s) < 1 || s[0] != ' ' {
		return
	}
	s = s[1:]

	// +/- indicator
	if len(s) < 1 || (s[0] != '+' && s[0] != '-') {
		return
	}
	islogin = s[:1]
	s = s[1:]

	// Space
	if len(s) < 1 || s[0] != ' ' {
		return
	}
	s = s[1:]

	// TTY: \S+ (non-space)
	idx := strings.Index(s, " ")
	if idx < 1 {
		return
	}
	tty = s[:idx]
	s = s[idx+1:]

	// fromuser-touser: ([^-]+)-(.*)$
	dashIdx := strings.Index(s, "-")
	if dashIdx < 1 {
		return
	}
	fromuser = s[:dashIdx]
	touser = s[dashIdx+1:]

	ok = true
	return
}

// Handle attempts to match and colorize a sulog line.
func (p *SulogPlugin) Handle(line string) (bool, string) {
	date, islogin, tty, fromuser, touser, ok := parseSulog(line)
	if !ok {
		return false, ""
	}

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
