package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var (
	icecastRe = regexp.MustCompile(
		`^(\[\d+/.../\d+:\d+:\d+:\d+\]) (Admin)? *(\[(\d+)?:?([^\]]*)\]) (.*)$`,
	)
	icecastReUsage = regexp.MustCompile(
		`^(\[\d+/.../\d+:\d+:\d+:\d+\]) (\[(\d+):([^\]]*)\]) (\[\d+/.../\d+:\d+:\d+:\d+\]) Bandwidth:([\d\.]+)([^ ]*) Sources:(\d+) Clients:(\d+) Admins:(\d+)`,
	)
)

// IcecastPlugin is a FULL plugin.
// Coloriser for Icecast(8) logs.
type IcecastPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewIcecastPlugin creates a new IcecastPlugin.
func NewIcecastPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *IcecastPlugin {
	return &IcecastPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *IcecastPlugin) Name() string        { return "icecast" }
func (p *IcecastPlugin) Type() Type          { return TypeFull }
func (p *IcecastPlugin) Description() string { return "Coloriser for Icecast(8) logs." }

// Handle attempts to match and colorize an Icecast log line.
// It tries the usage pattern first, then the general pattern.
func (p *IcecastPlugin) Handle(line string) (bool, string) {
	// Prefilter: both regexps start ^\[\d+/, so a match needs "[" then a
	// digit. Necessary condition only - never rejects a match.
	if len(line) < 2 || line[0] != '[' || line[1] < '0' || line[1] > '9' {
		return false, ""
	}

	// Try usage pattern first
	if m := icecastReUsage.FindStringSubmatch(line); m != nil {
		return p.handleUsage(m), ""
	}

	// Try general pattern
	if m := icecastRe.FindStringSubmatch(line); m != nil {
		return true, p.handleGeneral(m)
	}

	return false, ""
}

func (p *IcecastPlugin) handleUsage(m []string) bool {
	date := m[1]
	threadno := m[3]
	thread := m[4]
	date2 := m[5]
	bw := m[6]
	unit := m[7]
	src := m[8]
	clients := m[9]
	admins := m[10]

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.PIDB, "[")
	p.ct.WriteColored(p.w, color.Numbers, threadno)
	p.ct.WriteColored(p.w, color.Default, ":")
	p.ct.WriteColored(p.w, color.Keyword, thread)
	p.ct.WriteColored(p.w, color.PIDB, "]")
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.Date, date2)
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.Keyword, "Bandwidth:")
	p.ct.WriteColored(p.w, color.Numbers, bw)
	p.ct.WriteColored(p.w, color.Default, unit)
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.Keyword, "Sources:")
	p.ct.WriteColored(p.w, color.Numbers, src)
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.Keyword, "Clients:")
	p.ct.WriteColored(p.w, color.Numbers, clients)
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, color.Keyword, "Admins:")
	p.ct.WriteColored(p.w, color.Numbers, admins)

	p.ct.WriteNewline(p.w)

	return true
}

func (p *IcecastPlugin) handleGeneral(m []string) string {
	date := m[1]
	admin := m[2]
	// m[3] is the full bracket group including brackets
	threadno := m[4]
	thread := m[5]
	rest := m[6]

	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteSpace(p.w)

	if admin != "" {
		p.ct.WriteColored(p.w, color.Keyword, admin)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.PIDB, "[")
		p.ct.WriteColored(p.w, color.Host, thread)
		p.ct.WriteColored(p.w, color.PIDB, "]")
	} else {
		p.ct.WriteColored(p.w, color.PIDB, "[")
		p.ct.WriteColored(p.w, color.Numbers, threadno)
		p.ct.WriteColored(p.w, color.Default, ":")
		p.ct.WriteColored(p.w, color.Keyword, thread)
		p.ct.WriteColored(p.w, color.PIDB, "]")
	}
	p.ct.WriteSpace(p.w)

	return rest
}
