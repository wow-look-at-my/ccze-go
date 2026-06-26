package plugin

import (
	"io"
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
}

// NewSyslogPlugin creates a new SyslogPlugin.
func NewSyslogPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SyslogPlugin {
	return &SyslogPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *SyslogPlugin) Name() string        { return "syslog" }
func (p *SyslogPlugin) Type() Type          { return TypeFull }
func (p *SyslogPlugin) Description() string { return "Generic syslog(8) log coloriser." }

// parseSyslog hand-parses a syslog line.
// Format: ^(\S*\s{1,2}\d{1,2}\s\d\d:\d\d:\d\d)\s(\S+)\s+((\S+:?)\s(.*))$
func parseSyslog(line string) (date, host, send, process, msg string, ok bool) {
	n := len(line)

	// Month: \S* (non-space characters)
	i := 0
	for i < n && line[i] != ' ' {
		i++
	}
	if i == 0 || i >= n {
		return
	}

	// 1-2 spaces: \s{1,2}
	spStart := i
	for i < n && line[i] == ' ' {
		i++
	}
	if i-spStart < 1 || i-spStart > 2 {
		return
	}

	// Day: \d{1,2}
	dStart := i
	for i < n && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i-dStart < 1 || i-dStart > 2 {
		return
	}

	// Space before time
	if i >= n || line[i] != ' ' {
		return
	}
	i++

	// HH:MM:SS — exactly dd:dd:dd (8 chars)
	if i+8 > n {
		return
	}
	t := line[i : i+8]
	if !(t[0] >= '0' && t[0] <= '9' && t[1] >= '0' && t[1] <= '9' && t[2] == ':' &&
		t[3] >= '0' && t[3] <= '9' && t[4] >= '0' && t[4] <= '9' && t[5] == ':' &&
		t[6] >= '0' && t[6] <= '9' && t[7] >= '0' && t[7] <= '9') {
		return
	}
	i += 8
	date = line[:i]

	// \s (single space)
	if i >= n || line[i] != ' ' {
		return
	}
	i++

	// Host: \S+ (non-space)
	hStart := i
	for i < n && line[i] != ' ' {
		i++
	}
	if i == hStart {
		return
	}
	host = line[hStart:i]

	// \s+ (one or more spaces)
	if i >= n || line[i] != ' ' {
		return
	}
	for i < n && line[i] == ' ' {
		i++
	}

	// Rest is send = process + space + msg
	if i >= n {
		return
	}
	send = line[i:]

	// Split send into process (first non-space token) and msg
	j := 0
	for j < len(send) && send[j] != ' ' {
		j++
	}
	process = send[:j]
	if j < len(send) && send[j] == ' ' {
		msg = send[j+1:]
	}

	ok = true
	return
}

func (p *SyslogPlugin) Handle(line string) (bool, string) {
	date, host, send, process, msg, ok := parseSyslog(line)
	if !ok {
		return false, ""
	}

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
