package plugin

import (
	"io"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// ProFTPDPlugin colorizes ProFTPD access and auth log lines.
type ProFTPDPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewProFTPDPlugin creates a new ProFTPDPlugin.
func NewProFTPDPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *ProFTPDPlugin {
	return &ProFTPDPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *ProFTPDPlugin) Name() string        { return "proftpd" }
func (p *ProFTPDPlugin) Type() Type          { return TypeFull }
func (p *ProFTPDPlugin) Description() string { return "Coloriser for ProFTPD logs." }

func (p *ProFTPDPlugin) Handle(line string) (bool, string) {
	// Try access log
	if m := proftpdAccessFindSubmatch(line); m != nil {
		host := m[1]
		user := m[2]
		auser := m[3]
		date := m[4]
		command := m[5]
		file := m[6]
		ftpcode := m[7]
		size := m[8]

		p.ct.WriteColored(p.w, color.Host, host)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.User, user)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.User, auser)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.Default, "[")
		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteColored(p.w, color.Default, "]")
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.Default, "\"")
		p.ct.WriteColored(p.w, color.Keyword, command)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.URI, file)
		p.ct.WriteColored(p.w, color.Default, "\"")
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.FTPCodes, ftpcode)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.GetSize, size)

		p.ct.WriteNewline(p.w)

		return true, ""
	}

	// Try auth log
	if m := proftpdAuthFindSubmatch(line); m != nil {
		servhost := m[1]
		pid := m[2]
		remhost := m[3]
		date := m[4]
		cmd := m[5]
		value := m[6]
		ftpcode := m[7]

		p.ct.WriteColored(p.w, color.Host, servhost)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, "ftp server")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.PIDB, "[")
		p.ct.WriteColored(p.w, color.PID, pid)
		p.ct.WriteColored(p.w, color.PIDB, "]")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Host, remhost)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, "[")
		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteColored(p.w, color.Default, "]")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, "\"")
		p.ct.WriteColored(p.w, color.Keyword, cmd)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, value)
		p.ct.WriteColored(p.w, color.Default, "\"")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.FTPCodes, ftpcode)

		p.ct.WriteNewline(p.w)

		return true, ""
	}

	return false, ""
}
