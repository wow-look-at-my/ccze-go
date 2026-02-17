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
func (p *EximPlugin) Type() Type           { return TypeFull }
func (p *EximPlugin) Description() string  { return "Coloriser for exim logs." }

func (p *EximPlugin) Handle(line string) (bool, string) {
	em := eximFindSubmatch(line)
	if em == nil {
		return false, ""
	}
	date := em[1]
	msgfull := em[2]

	var uniqn, action, msg string
	var actionColor color.Color

	if am := eximActionTypeFindSubmatch(msgfull); am != nil {
		uniqn = am[1]
		action = am[2]
		msg = am[3]
		if action[0] == '<' {
			actionColor = color.Incoming
		} else if len(action) > 1 && action[1] == '>' {
			actionColor = color.Outgoing
		} else if action[0] == '=' || action[0] == '*' {
			actionColor = color.Error
		} else {
			actionColor = color.Unknown
		}
	} else if um := eximUniqnFindSubmatch(msgfull); um != nil {
		uniqn = um[1]
		msg = um[2]
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
