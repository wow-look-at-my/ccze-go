package plugin

import (
	"io"
	"regexp"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// SquidPlugin colorizes squid access, store and cache log lines.
type SquidPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
	reAccess *regexp.Regexp // kept: 13 capture groups, complex
	reStore  *regexp.Regexp // kept: 18 capture groups, complex
}

// NewSquidPlugin creates a new SquidPlugin.
func NewSquidPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SquidPlugin {
	return &SquidPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
		reAccess: regexp.MustCompile(`^(\d{9,10}\.\d{3})(\s+)(\d+)\s(\S+)\s(\w+)/(\d{3})\s(\d+)\s(\w+)\s(\S+)\s(\S+)\s(\w+)/([\d\.]+|-)\s(.*)`),
		reStore:  regexp.MustCompile(`^([\d\.]+)\s(\w+)\s(-?[\dA-F]+)\s+(\S+)\s([\dA-F]+)(\s+)(\d{3}|\?)(\s+)(-?[\d\?]+)(\s+)(-?[\d\?]+)(\s+)(-?[\d\?]+)\s(\S+)\s(-?[\d|\?]+)/(-?[\d|\?]+)\s(\S+)\s(.*)`),
	}
}

func (p *SquidPlugin) Name() string { return "squid" }
func (p *SquidPlugin) Type() Type   { return TypeFull }
func (p *SquidPlugin) Description() string {
	return "Coloriser for squid access, store and cache logs."
}

// proxyAction returns the color for a squid proxy action string.
func proxyAction(action string) color.Color {
	if strings.HasPrefix(action, "ERR") {
		return color.Error
	}
	if strings.Contains(action, "MISS") {
		return color.ProxyMiss
	}
	if strings.Contains(action, "HIT") {
		return color.ProxyHit
	}
	if strings.Contains(action, "DENIED") {
		return color.ProxyDenied
	}
	if strings.Contains(action, "REFRESH") {
		return color.ProxyRefresh
	}
	if strings.Contains(action, "SWAPFAIL") {
		return color.ProxySwapfail
	}
	if strings.Contains(action, "NONE") {
		return color.Debug
	}
	return color.Unknown
}

// proxyHierarchy returns the color for a squid proxy hierarchy string.
func proxyHierarchy(hierar string) color.Color {
	if strings.HasPrefix(hierar, "NO") {
		return color.Warning
	}
	if strings.Contains(hierar, "DIRECT") {
		return color.ProxyDirect
	}
	if strings.Contains(hierar, "PARENT") {
		return color.ProxyParent
	}
	if strings.Contains(hierar, "MISS") {
		return color.ProxyMiss
	}
	return color.Unknown
}

// proxyTag returns the color for a squid proxy store tag string.
func proxyTag(tag string) color.Color {
	if strings.Contains(tag, "CREATE") {
		return color.ProxyCreate
	}
	if strings.Contains(tag, "SWAPIN") {
		return color.ProxySwapin
	}
	if strings.Contains(tag, "SWAPOUT") {
		return color.ProxySwapout
	}
	if strings.Contains(tag, "RELEASE") {
		return color.ProxyRelease
	}
	return color.Unknown
}

// parseSquidCache hand-parses a squid cache log line.
// Format: ^(\d{4}/\d{2}/\d{2}\s(\d{2}:){2}\d{2}\|)\s(.*)$
// e.g. "2023/10/15 14:30:22| Starting Squid Cache"
func parseSquidCache(line string) (date, other string, ok bool) {
	// Minimum: "YYYY/MM/DD HH:MM:SS| X"  = 21 chars
	if len(line) < 21 {
		return
	}
	// YYYY/MM/DD
	d := line[:10]
	if !(d[0] >= '0' && d[0] <= '9' && d[1] >= '0' && d[1] <= '9' &&
		d[2] >= '0' && d[2] <= '9' && d[3] >= '0' && d[3] <= '9' && d[4] == '/' &&
		d[5] >= '0' && d[5] <= '9' && d[6] >= '0' && d[6] <= '9' && d[7] == '/' &&
		d[8] >= '0' && d[8] <= '9' && d[9] >= '0' && d[9] <= '9') {
		return
	}
	if line[10] != ' ' {
		return
	}
	// HH:MM:SS
	t := line[11:19]
	if !(t[0] >= '0' && t[0] <= '9' && t[1] >= '0' && t[1] <= '9' && t[2] == ':' &&
		t[3] >= '0' && t[3] <= '9' && t[4] >= '0' && t[4] <= '9' && t[5] == ':' &&
		t[6] >= '0' && t[6] <= '9' && t[7] >= '0' && t[7] <= '9') {
		return
	}
	// |
	if line[19] != '|' {
		return
	}
	date = line[:20] // includes the |
	// space
	if len(line) < 22 || line[20] != ' ' {
		return
	}
	other = line[21:]
	ok = true
	return
}

func (p *SquidPlugin) Handle(line string) (bool, string) {
	// Try access log (kept as regex — 13 capture groups)
	if m := p.reAccess.FindStringSubmatch(line); m != nil {
		date := m[1]
		espace := m[2]
		elaps := m[3]
		host := m[4]
		action := m[5]
		httpc := m[6]
		gsize := m[7]
		method := m[8]
		uri := m[9]
		ident := m[10]
		hierar := m[11]
		fhost := m[12]
		ctype := m[13]

		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteColored(p.w, color.Default, espace)
		p.ct.WriteColored(p.w, color.GetTime, elaps)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.Host, host)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, proxyAction(action), action)
		p.ct.WriteColored(p.w, color.Default, "/")
		p.ct.WriteColored(p.w, color.HTTPCodes, httpc)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.GetSize, gsize)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, HTTPAction(method), method)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.URI, uri)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.Ident, ident)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, proxyHierarchy(hierar), hierar)
		p.ct.WriteColored(p.w, color.Default, "/")
		p.ct.WriteColored(p.w, color.Host, fhost)
		p.ct.WriteSpace(p.w)

		p.ct.WriteColored(p.w, color.CType, ctype)

		p.ct.WriteNewline(p.w)

		return true, ""
	}

	// Try store log (kept as regex — 18 capture groups)
	if m := p.reStore.FindStringSubmatch(line); m != nil {
		date := m[1]
		tag := m[2]
		swapnum := m[3]
		swapname := m[4]
		swapsum := m[5]
		space1 := m[6]
		hcode := m[7]
		space2 := m[8]
		hdate := m[9]
		space3 := m[10]
		lmdate := m[11]
		space4 := m[12]
		expire := m[13]
		ctype := m[14]
		size := m[15]
		read := m[16]
		method := m[17]
		uri := m[18]

		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, proxyTag(tag), tag)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.SwapNum, swapnum)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.SwapNum, swapname)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.SwapNum, swapsum)
		p.ct.WriteColored(p.w, color.Default, space1)
		p.ct.WriteColored(p.w, color.HTTPCodes, hcode)
		p.ct.WriteColored(p.w, color.Default, space2)
		PrintDate(p.w, p.ct, hdate, p.convdate)
		p.ct.WriteColored(p.w, color.Default, space3)
		PrintDate(p.w, p.ct, lmdate, p.convdate)
		p.ct.WriteColored(p.w, color.Default, space4)
		PrintDate(p.w, p.ct, expire, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.CType, ctype)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.GetSize, size)
		p.ct.WriteColored(p.w, color.Default, "/")
		p.ct.WriteColored(p.w, color.GetSize, read)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, HTTPAction(method), method)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.URI, uri)

		p.ct.WriteNewline(p.w)

		return true, ""
	}

	// Try cache log (hand-parsed — fixed date format with | delimiter)
	if date, other, ok := parseSquidCache(line); ok {
		p.ct.WriteColored(p.w, color.Date, date)
		p.ct.WriteSpace(p.w)

		return true, other
	}

	return false, ""
}
