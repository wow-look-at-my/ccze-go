package plugin

import (
	"io"
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
}

// NewSquidPlugin creates a new SquidPlugin.
func NewSquidPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SquidPlugin {
	return &SquidPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *SquidPlugin) Name() string        { return "squid" }
func (p *SquidPlugin) Type() Type           { return TypeFull }
func (p *SquidPlugin) Description() string  { return "Coloriser for squid access, store and cache logs." }

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

func (p *SquidPlugin) Handle(line string) (bool, string) {
	// Try access log
	if m := squidAccessFindSubmatch(line); m != nil {
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

	// Try store log
	if m := squidStoreFindSubmatch(line); m != nil {
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

	// Try cache log
	if m := squidCacheFindSubmatch(line); m != nil {
		date := m[1]
		other := m[3]
		p.ct.WriteColored(p.w, color.Date, date)
		p.ct.WriteSpace(p.w)
		return true, other
	}

	return false, ""
}
