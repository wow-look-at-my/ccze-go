package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var vsftpdRe = regexp.MustCompile(`^(\S+\s+\S+\s+\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}\s+\d+)(\s+)\[pid (\d+)\]\s+(\[(\S+)\])?\s*(.*)$`)

// VsftpdPlugin colorizes vsftpd log lines.
type VsftpdPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewVsftpdPlugin creates a new VsftpdPlugin.
func NewVsftpdPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *VsftpdPlugin {
	return &VsftpdPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *VsftpdPlugin) Name() string        { return "vsftpd" }
func (p *VsftpdPlugin) Type() Type          { return TypeFull }
func (p *VsftpdPlugin) Description() string { return "Coloriser for vsftpd logs." }

func (p *VsftpdPlugin) Handle(line string) (bool, string) {
	m := vsftpdRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	sspace := m[2]
	pid := m[3]
	// m[4] is the full "[user]" group (possibly empty)
	user := m[5]
	other := m[6]

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteColored(p.w, color.Default, sspace)

	p.ct.WriteColored(p.w, color.PIDB, "[")
	p.ct.WriteColored(p.w, color.Default, "pid ")
	p.ct.WriteColored(p.w, color.PID, pid)
	p.ct.WriteColored(p.w, color.PIDB, "]")
	p.ct.WriteSpace(p.w)

	if user != "" {
		p.ct.WriteColored(p.w, color.PIDB, "[")
		p.ct.WriteColored(p.w, color.User, user)
		p.ct.WriteColored(p.w, color.PIDB, "]")
		p.ct.WriteSpace(p.w)
	}

	return true, other
}
