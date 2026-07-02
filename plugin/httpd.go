package plugin

import (
	"io"
	"regexp"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// HTTPDPlugin colorizes generic HTTPD access and error log lines.
type HTTPDPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
	reAccess *regexp.Regexp // kept: complex multi-group Apache combined log format
}

// NewHTTPDPlugin creates a new HTTPDPlugin.
func NewHTTPDPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *HTTPDPlugin {
	return &HTTPDPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
		reAccess: regexp.MustCompile(`^(\S*)\s(\S*)?\s?-\s(\S+)\s(\[\d{1,2}/\S*/\d{4}:\d{2}:\d{2}:\d{2}.{0,6}[^\]]*\])\s("([^ "]+)\s*[^"]*")\s(\d{3})\s(\d+|-)\s*(.*)$`),
	}
}

func (p *HTTPDPlugin) Name() string { return "httpd" }
func (p *HTTPDPlugin) Type() Type   { return TypeFull }
func (p *HTTPDPlugin) Description() string {
	return "Coloriser for generic HTTPD access and error logs."
}

// httpdErrorColor returns the color for an HTTP error log level.
func httpdErrorColor(level string) color.Color {
	if strings.Contains(level, "debug") || strings.Contains(level, "info") ||
		strings.Contains(level, "notice") {
		return color.Debug
	}
	if strings.Contains(level, "warn") {
		return color.Warning
	}
	if strings.Contains(level, "error") || strings.Contains(level, "crit") ||
		strings.Contains(level, "alert") || strings.Contains(level, "emerg") {
		return color.Error
	}
	return color.Unknown
}

// parseHTTPDError hand-parses an HTTPD error log line.
// Format: ^(\[\w{3}\s\w{3}\s{1,2}\d{1,2}\s\d{2}:\d{2}:\d{2}\s\d{4}\])\s(\[\w*\])\s(.*)$
func parseHTTPDError(line string) (date, level, msg string, ok bool) {
	if len(line) == 0 || line[0] != '[' {
		return
	}

	// Find closing bracket for date: [Day Mon DD HH:MM:SS YYYY]
	closeBracket := strings.Index(line, "] ")
	if closeBracket < 10 {
		return
	}
	dateContent := line[1:closeBracket]

	// Validate date format: \w{3}\s\w{3}\s{1,2}\d{1,2}\s\d{2}:\d{2}:\d{2}\s\d{4}
	// e.g. "Sun Oct 12 15:30:00 2003"
	// Quick validation: must contain a time-like pattern and end with 4 digits (year)
	if len(dateContent) < 20 {
		return
	}
	// Check year at end (4 digits)
	year := dateContent[len(dateContent)-4:]
	for i := 0; i < 4; i++ {
		if year[i] < '0' || year[i] > '9' {
			return
		}
	}
	date = line[:closeBracket+1]
	rest := line[closeBracket+2:]

	// Level: [\w*] — brackets around a word
	if len(rest) == 0 || rest[0] != '[' {
		return
	}
	levelClose := strings.Index(rest, "] ")
	if levelClose < 1 {
		return
	}
	level = rest[:levelClose+1]
	msg = rest[levelClose+2:]

	ok = true
	return
}

// httpdDateBracketHint reports whether line contains "[d/" or "[dd/" (a '['
// followed by 1-2 digits and a '/'), which reAccess's date group
// \[\d{1,2}/ requires. Necessary condition only.
func httpdDateBracketHint(line string) bool {
	for i := 0; ; {
		j := strings.IndexByte(line[i:], '[')
		if j < 0 {
			return false
		}
		i += j + 1
		if digitAt(line, i) {
			if i+1 < len(line) && line[i+1] == '/' {
				return true
			}
			if digitAt(line, i+1) && i+2 < len(line) && line[i+2] == '/' {
				return true
			}
		}
	}
}

func (p *HTTPDPlugin) Handle(line string) (bool, string) {
	// Prefilter: reAccess requires a double-quoted request (literal '"')
	// and a bracketed date starting \[\d{1,2}/. Necessary conditions only;
	// the nested optional groups in reAccess backtrack heavily on
	// non-matching lines, so these cheap scans pay off on every miss.
	var m []string
	if strings.IndexByte(line, '"') >= 0 && httpdDateBracketHint(line) {
		m = p.reAccess.FindStringSubmatch(line)
	}

	// Try access log first (kept as regex — complex multi-group pattern)
	if m != nil {
		vhost := m[1]
		host := m[2]
		user := m[3]
		date := m[4]
		fullAction := m[5]
		method := m[6]
		httpCode := m[7]
		gsize := m[8]
		other := m[9]

		p.ct.WriteColored(p.w, color.Host, vhost)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Host, host)
		if host != "" {
			p.ct.WriteSpace(p.w)
		}
		p.ct.WriteColored(p.w, color.Default, "-")
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.User, user)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.Date, date)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, HTTPAction(method), fullAction)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.HTTPCodes, httpCode)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.GetSize, gsize)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.Default, other)
		p.ct.WriteNewline(p.w)

		return true, ""
	}

	// Try error log (hand-parsed — bracket-delimited segments)
	if date, level, msg, ok := parseHTTPDError(line); ok {
		lcol := httpdErrorColor(level)

		p.ct.WriteColored(p.w, color.Date, date)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, lcol, level)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, lcol, msg)

		p.ct.WriteNewline(p.w)

		return true, ""
	}

	return false, ""
}
