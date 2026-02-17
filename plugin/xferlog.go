package plugin

import (
	"io"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// XferlogPlugin is a FULL plugin.
// Coloriser for xferlog(5) logs.
type XferlogPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewXferlogPlugin creates a new XferlogPlugin.
func NewXferlogPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *XferlogPlugin {
	return &XferlogPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *XferlogPlugin) Name() string        { return "xferlog" }
func (p *XferlogPlugin) Type() Type          { return TypeFull }
func (p *XferlogPlugin) Description() string { return "Generic xferlog coloriser." }

// Handle attempts to match and colorize an xferlog line.
func (p *XferlogPlugin) Handle(line string) (bool, string) {
	m := xferlogFindSubmatch(line)
	if m == nil {
		return false, ""
	}

	curtime := m[1]
	transtime := m[2]
	host := m[3]
	fsize := m[4]
	fname := m[5]
	transtype := m[6]
	actionflag := m[7]
	direction := m[8]
	amode := m[9]
	user := m[10]
	service := m[11]
	amethod := m[12]
	auid := m[13]
	status := m[14]

	p.ct.WriteColored(p.w, color.Date, curtime)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.GetTime, transtime)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Host, host)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.GetSize, fsize)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Dir, fname)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.PIDB, transtype)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.FTPCodes, actionflag)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.FTPCodes, direction)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.FTPCodes, amode)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.User, user)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Service, service)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.FTPCodes, amethod)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.User, auid)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.FTPCodes, status)

	p.ct.WriteNewline(p.w)

	return true, ""
}
