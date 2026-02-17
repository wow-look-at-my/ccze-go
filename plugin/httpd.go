package plugin

import (
	"io"
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
}

// NewHTTPDPlugin creates a new HTTPDPlugin.
func NewHTTPDPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *HTTPDPlugin {
	return &HTTPDPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *HTTPDPlugin) Name() string        { return "httpd" }
func (p *HTTPDPlugin) Type() Type           { return TypeFull }
func (p *HTTPDPlugin) Description() string  { return "Coloriser for generic HTTPD access and error logs." }

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

func (p *HTTPDPlugin) Handle(line string) (bool, string) {
	// Try access log first
	if m := httpdAccessFindSubmatch(line); m != nil {
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

	// Try error log
	if m := httpdErrorFindSubmatch(line); m != nil {
		date := m[1]
		level := m[2]
		msg := m[3]
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
