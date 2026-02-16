package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var ftpstatsRe = regexp.MustCompile(
	`^(\d{9,10})\s([\da-f]+\.[\da-f]+)\s([^\s]+)\s([^\s]+)\s(U|D)\s(\d+)\s(\d+)\s(.*)$`,
)

// FtpstatsPlugin is a FULL plugin.
// Coloriser for ftpstats (pure-ftpd) logs.
type FtpstatsPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewFtpstatsPlugin creates a new FtpstatsPlugin.
func NewFtpstatsPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *FtpstatsPlugin {
	return &FtpstatsPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *FtpstatsPlugin) Name() string        { return "ftpstats" }
func (p *FtpstatsPlugin) Type() Type          { return TypeFull }
func (p *FtpstatsPlugin) Description() string { return "Coloriser for ftpstats (pure-ftpd) logs." }

// Handle attempts to match and colorize an ftpstats log line.
func (p *FtpstatsPlugin) Handle(line string) (bool, string) {
	m := ftpstatsRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	sessionid := m[2]
	user := m[3]
	host := m[4]
	typ := m[5]
	size := m[6]
	duration := m[7]
	file := m[8]

	PrintDate(p.w, p.ct, date, p.convdate)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Uniqn, sessionid)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.User, user)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Host, host)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.FTPCodes, typ)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.GetSize, size)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Date, duration)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Dir, file)
	p.ct.WriteNewline(p.w)

	return true, ""
}
