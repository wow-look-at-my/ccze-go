package plugin

import (
	"io"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// EximPlugin colorizes exim log lines.
type EximPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewEximPlugin creates a new EximPlugin.
func NewEximPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *EximPlugin {
	return &EximPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *EximPlugin) Name() string        { return "exim" }
func (p *EximPlugin) Type() Type          { return TypeFull }
func (p *EximPlugin) Description() string { return "Coloriser for exim logs." }

// parseEximDate hand-parses the date prefix of an exim log line.
// Format: ^(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2})\s(.*)$
func parseEximDate(line string) (date, rest string, ok bool) {
	// "YYYY-MM-DD HH:MM:SS " = 20 chars minimum
	if len(line) < 20 {
		return
	}
	// YYYY-MM-DD
	d := line[:10]
	if !(d[0] >= '0' && d[0] <= '9' && d[1] >= '0' && d[1] <= '9' &&
		d[2] >= '0' && d[2] <= '9' && d[3] >= '0' && d[3] <= '9' && d[4] == '-' &&
		d[5] >= '0' && d[5] <= '9' && d[6] >= '0' && d[6] <= '9' && d[7] == '-' &&
		d[8] >= '0' && d[8] <= '9' && d[9] >= '0' && d[9] <= '9') {
		return
	}
	if line[10] != ' ' {
		return
	}
	// HH:MM:SS
	t := line[11:19]
	if !(t[0] >= '0' && t[0] <= '9' && t[1] >= '0' && t[1] <= '9' && t[2] == ':' &&
		t[3] >= '0' && t[3] <= '9' && t[4] >= '0' && t[4] <= '9' && t[5] == ':' &&
		t[6] >= '0' && t[6] <= '9' && t[7] >= '0' && t[7] <= '9') {
		return
	}
	if line[19] != ' ' {
		return
	}
	date = line[:19]
	rest = line[20:]
	ok = true
	return
}

// parseEximActionType tries to parse "UNIQUEID ACTION REST" from msgfull.
// Format: ^(\S{16})\s([<=*][=>*])\s(\S+.*)$
func parseEximActionType(msgfull string) (uniqn, action, rest string, ok bool) {
	if len(msgfull) < 19 { // 16 + space + 2 + space minimum
		return
	}
	// 16-char unique ID: all non-space
	for i := 0; i < 16; i++ {
		if msgfull[i] == ' ' {
			return
		}
	}
	if msgfull[16] != ' ' {
		return
	}
	uniqn = msgfull[:16]

	// Action: two chars from [<=*][=>*]
	a := msgfull[17:19]
	if !((a[0] == '<' || a[0] == '=' || a[0] == '*') && (a[1] == '=' || a[1] == '>' || a[1] == '*')) {
		return
	}
	action = a

	if len(msgfull) < 20 || msgfull[19] != ' ' {
		return
	}
	rest = msgfull[20:]
	if len(rest) == 0 {
		return
	}

	ok = true
	return
}

// parseEximUniqn tries to parse "UNIQUEID REST" from msgfull.
// Format: ^(\S{16})\s(.*)$
func parseEximUniqn(msgfull string) (uniqn, rest string, ok bool) {
	if len(msgfull) < 18 { // 16 + space + at least 1
		return
	}
	for i := 0; i < 16; i++ {
		if msgfull[i] == ' ' {
			return
		}
	}
	if msgfull[16] != ' ' {
		return
	}
	uniqn = msgfull[:16]
	rest = msgfull[17:]
	ok = true
	return
}

func (p *EximPlugin) Handle(line string) (bool, string) {
	date, msgfull, ok := parseEximDate(line)
	if !ok {
		return false, ""
	}

	var uniqn, action, msg string
	var actionColor color.Color

	// Try action type first
	if u, a, r, ok := parseEximActionType(msgfull); ok {
		uniqn = u
		action = a
		msg = r

		if action[0] == '<' {
			actionColor = color.Incoming
		} else if len(action) > 1 && action[1] == '>' {
			actionColor = color.Outgoing
		} else if action[0] == '=' || action[0] == '*' {
			actionColor = color.Error
		} else {
			actionColor = color.Unknown
		}
	} else if u, r, ok := parseEximUniqn(msgfull); ok {
		uniqn = u
		msg = r
	} else {
		msg = msgfull
	}

	PrintDate(p.w, p.ct, date, p.convdate)
	p.ct.WriteSpace(p.w)

	if uniqn != "" {
		p.ct.WriteColored(p.w, color.Uniqn, uniqn)
		p.ct.WriteSpace(p.w)
	}

	if action != "" {
		p.ct.WriteColored(p.w, actionColor, action)
		p.ct.WriteSpace(p.w)
	}

	return true, msg
}
