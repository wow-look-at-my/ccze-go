package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// ApmPlugin is a PARTIAL plugin.
// Coloriser for APM sub-logs.
type ApmPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewApmPlugin creates a new ApmPlugin.
func NewApmPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *ApmPlugin {
	return &ApmPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *ApmPlugin) Name() string        { return "apm" }
func (p *ApmPlugin) Type() Type           { return TypePartial }
func (p *ApmPlugin) Description() string  { return "Coloriser for APM sub-logs." }

// parseApm hand-parses an APM log line.
// Format: Battery: (-?\d*)%, ((.*)charging) \((-?\d*)% ([^ ]*) (\d*:\d*:\d*)\), (\d*:\d*:\d*) (.*)
func parseApm(line string) (battery, charge, rate, stuff1, elapsed, remain, stuff2 string, ok bool) {
	// Find "Battery: "
	idx := strings.Index(line, "Battery: ")
	if idx < 0 {
		return
	}
	s := line[idx+9:] // skip "Battery: "

	// battery: -?\d*
	i := 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	battery = s[:i]
	s = s[i:]

	// "%, "
	if !strings.HasPrefix(s, "%, ") {
		return
	}
	s = s[3:]

	// charge: (.*)charging — everything up to and including "charging"
	chgIdx := strings.Index(s, "charging")
	if chgIdx < 0 {
		return
	}
	charge = s[:chgIdx+8] // includes "charging"
	s = s[chgIdx+8:]

	// " ("
	if !strings.HasPrefix(s, " (") {
		return
	}
	s = s[2:]

	// rate: -?\d*
	i = 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	rate = s[:i]
	s = s[i:]

	// "% "
	if !strings.HasPrefix(s, "% ") {
		return
	}
	s = s[2:]

	// stuff1: [^ ]* (non-space)
	spIdx := strings.Index(s, " ")
	if spIdx < 0 {
		return
	}
	stuff1 = s[:spIdx]
	s = s[spIdx+1:]

	// elapsed: \d*:\d*:\d*
	spIdx = strings.Index(s, ")")
	if spIdx < 0 {
		return
	}
	elapsed = s[:spIdx]
	s = s[spIdx:]

	// "), "
	if !strings.HasPrefix(s, "), ") {
		return
	}
	s = s[3:]

	// remain: \d*:\d*:\d*
	spIdx = strings.Index(s, " ")
	if spIdx < 0 {
		return
	}
	remain = s[:spIdx]
	stuff2 = s[spIdx+1:]

	ok = true
	return
}

// Handle attempts to match and colorize an APM log line.
func (p *ApmPlugin) Handle(line string) (bool, string) {
	battery, charge, rate, stuff1, elapsed, remain, stuff2, ok := parseApm(line)
	if !ok {
		return false, ""
	}

	p.ct.WriteColored(p.w, color.Default, "Battery:")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Percentage, battery)
	p.ct.WriteColored(p.w, color.Default, "%,")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.SystemWord, charge)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Default, "(")
	p.ct.WriteColored(p.w, color.Percentage, rate)
	p.ct.WriteColored(p.w, color.Default, "%")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Default, stuff1)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Date, elapsed)
	p.ct.WriteColored(p.w, color.Default, "),")
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Date, remain)
	p.ct.WriteSpace(p.w)

	return true, stuff2
}
