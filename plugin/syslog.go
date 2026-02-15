package plugin

import (
	"io"
	"regexp"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// SyslogPlugin colorizes generic syslog(8) log lines.
type SyslogPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
	re       *regexp.Regexp
}

// NewSyslogPlugin creates a new SyslogPlugin.
func NewSyslogPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SyslogPlugin {
	return &SyslogPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
		re:       regexp.MustCompile(`^(\S*\s{1,2}\d{1,2}\s\d\d:\d\d:\d\d)\s(\S+)\s+((\S+:?)\s(.*))$`),
	}
}

func (p *SyslogPlugin) Name() string        { return "syslog" }
func (p *SyslogPlugin) Type() Type           { return TypeFull }
func (p *SyslogPlugin) Description() string  { return "Generic syslog(8) log coloriser." }

func (p *SyslogPlugin) Handle(line string) (bool, string) {
	m := p.re.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	date := m[1]
	host := m[2]
	send := m[3]
	process := m[4]
	msg := m[5]

	// Check for "last message repeated ... times" or "-- MARK --"
	if (strings.Contains(send, "last message repeated") && strings.Contains(send, "times")) ||
		strings.Contains(send, "-- MARK --") {
		// Special handling: output date, host, entire send as Repeat color
		p.ct.WriteColored(p.w, color.Date, date)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Host, host)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Repeat, send)
		p.ct.WriteNewline(p.w)
		return true, ""
	}

	// Extract PID from process field if present, e.g. "sshd[1234]"
	var procName, pid string
	if idx := strings.Index(process, "["); idx >= 0 {
		procName = process[:idx]
		end := strings.Index(process[idx:], "]")
		if end >= 0 {
			pid = process[idx+1 : idx+end]
		}
	} else {
		procName = process
	}

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Host, host)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Proc, procName)
	if pid != "" {
		p.ct.WriteColored(p.w, color.PIDB, "[")
		p.ct.WriteColored(p.w, color.PID, pid)
		p.ct.WriteColored(p.w, color.PIDB, "]")
		p.ct.WriteColored(p.w, color.Proc, ":")
	}
	p.ct.WriteSpace(p.w)

	return true, msg
}
