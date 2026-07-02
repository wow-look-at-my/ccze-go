package plugin

import (
	"io"
	"regexp"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var (
	proftpdReAccess = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+) (\S+) (\S+) \[(\d{2}/.{3}/\d{4}:\d{2}:\d{2}:\d{2} [-+]\d{4})\] "([A-Z]+) ([^"]+)" (\d{3}) (-|\d+)`)
	proftpdReAuth   = regexp.MustCompile(`^(\S+) ftp server \[(\d+)\] (\d+\.\d+\.\d+\.\d+) \[(\d{2}/.{3}/\d{4}:\d{2}:\d{2}:\d{2} [-+]\d{4})\] "([A-Z]+) ([^"]+)" (\d{3})`)
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
	// Prefilter: reAccess starts ^\d+\. (leading digit) and reAuth requires
	// the literal " ftp server [". Necessary conditions only.
	var m []string
	if digitAt(line, 0) {
		m = proftpdReAccess.FindStringSubmatch(line)
	}

	// Try access log
	if m != nil {
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
	m = nil
	if strings.Contains(line, " ftp server [") {
		m = proftpdReAuth.FindStringSubmatch(line)
	}
	if m != nil {
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
