package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var (
	eximRe           = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2})\s(.*)$`)
	eximReActionType = regexp.MustCompile(`^(\S{16})\s([<=\*][=>\*])\s(\S+.*)$`)
	eximReUniqn      = regexp.MustCompile(`^(\S{16})\s(.*)$`)
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

func (p *EximPlugin) Handle(line string) (bool, string) {
	m := eximRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	msgfull := m[2]

	var uniqn, action, msg string
	var actionColor color.Color

	// Try action type sub-regex first
	if m2 := eximReActionType.FindStringSubmatch(msgfull); m2 != nil {
		uniqn = m2[1]
		action = m2[2]
		msg = m2[3]

		if action[0] == '<' {
			actionColor = color.Incoming
		} else if len(action) > 1 && action[1] == '>' {
			actionColor = color.Outgoing
		} else if action[0] == '=' || action[0] == '*' {
			actionColor = color.Error
		} else {
			actionColor = color.Unknown
		}
	} else if m2 := eximReUniqn.FindStringSubmatch(msgfull); m2 != nil {
		// Try unique ID sub-regex
		uniqn = m2[1]
		msg = m2[2]
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
