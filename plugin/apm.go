package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

var apmRe = regexp.MustCompile(`Battery: (-?\d*)%, ((.*)charging) \((-?\d*)% ([^ ]*) (\d*:\d*:\d*)\), (\d*:\d*:\d*) (.*)`)

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
func (p *ApmPlugin) Type() Type          { return TypePartial }
func (p *ApmPlugin) Description() string { return "Coloriser for APM sub-logs." }

// Handle attempts to match and colorize an APM log line.
func (p *ApmPlugin) Handle(line string) (bool, string) {
	m := apmRe.FindStringSubmatch(line)
	if m == nil {
		return false, ""
	}

	battery := m[1]
	charge := m[2]
	// m[3] is the inner group (e.g. "" or "not "), not used separately
	rate := m[4]
	stuff1 := m[5]
	elapsed := m[6]
	remain := m[7]
	stuff2 := m[8]

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
