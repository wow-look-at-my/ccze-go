package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

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

// parseVsftpd hand-parses a vsftpd log line.
// Format: ^(\S+\s+\S+\s+\d{1,2}\s+\d{1,2}:\d{1,2}:\d{1,2}\s+\d+)(\s+)\[pid (\d+)\]\s+(\[(\S+)\])?\s*(.*)$
func parseVsftpd(line string) (date, sspace, pid, user, other string, ok bool) {
	// Date format: "Mon Oct 15 14:30:22 2023"
	// We need to find: word space(s) word space(s) digits space(s) time space(s) year
	// Then space(s) "[pid " digits "]"

	// Find "[pid " marker
	pidIdx := strings.Index(line, "[pid ")
	if pidIdx < 1 {
		return
	}

	// Everything before "[pid " is date + spacing
	// Trim trailing spaces to separate date from spacing
	dateEnd := pidIdx
	for dateEnd > 0 && line[dateEnd-1] == ' ' {
		dateEnd--
	}
	if dateEnd == 0 {
		return
	}

	// Validate the date portion has a time-like pattern (contains :)
	dateStr := line[:dateEnd]
	if !strings.Contains(dateStr, ":") {
		return
	}

	// Verify the date portion ends with a year (digits)
	lastSpace := strings.LastIndex(dateStr, " ")
	if lastSpace < 0 {
		return
	}
	yearPart := dateStr[lastSpace+1:]
	for i := 0; i < len(yearPart); i++ {
		if yearPart[i] < '0' || yearPart[i] > '9' {
			return
		}
	}
	if len(yearPart) == 0 {
		return
	}

	date = dateStr
	sspace = line[dateEnd:pidIdx]

	// Parse "[pid DIGITS]"
	rest := line[pidIdx+5:] // skip "[pid "
	pidEnd := 0
	for pidEnd < len(rest) && rest[pidEnd] >= '0' && rest[pidEnd] <= '9' {
		pidEnd++
	}
	if pidEnd == 0 || pidEnd >= len(rest) || rest[pidEnd] != ']' {
		return
	}
	pid = rest[:pidEnd]
	rest = rest[pidEnd+1:]

	// Skip whitespace
	for len(rest) > 0 && rest[0] == ' ' {
		rest = rest[1:]
	}

	// Optional [user]
	if len(rest) > 0 && rest[0] == '[' {
		closeIdx := strings.Index(rest, "]")
		if closeIdx > 1 {
			user = rest[1:closeIdx]
			rest = rest[closeIdx+1:]
			// Skip whitespace after [user]
			for len(rest) > 0 && rest[0] == ' ' {
				rest = rest[1:]
			}
		}
	}

	other = rest
	ok = true
	return
}

func (p *VsftpdPlugin) Handle(line string) (bool, string) {
	date, sspace, pid, user, other, ok := parseVsftpd(line)
	if !ok {
		return false, ""
	}

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
