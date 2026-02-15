package plugin

import (
	"io"
	"regexp"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// DpkgPlugin colorizes dpkg log lines.
type DpkgPlugin struct {
	w          io.Writer
	ct         *color.Table
	wc         *wordcolor.Processor
	convdate   bool
	reStatus   *regexp.Regexp
	reAction   *regexp.Regexp
	reConffile *regexp.Regexp
}

// NewDpkgPlugin creates a new DpkgPlugin.
func NewDpkgPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *DpkgPlugin {
	return &DpkgPlugin{
		w:          w,
		ct:         ct,
		wc:         wc,
		convdate:   convdate,
		reStatus:   regexp.MustCompile(`^([-\d]{10}\s[:\d]{8})\sstatus\s(\S+)\s(\S+)\s(\S+)$`),
		reAction:   regexp.MustCompile(`^([-\d]{10}\s[:\d]{8})\s(install|upgrade|remove|purge)\s(\S+)\s(\S+)\s(\S+)$`),
		reConffile: regexp.MustCompile(`^([-\d]{10}\s[:\d]{8})\sconffile\s(\S+)\s(install|keep)$`),
	}
}

func (p *DpkgPlugin) Name() string        { return "dpkg" }
func (p *DpkgPlugin) Type() Type           { return TypeFull }
func (p *DpkgPlugin) Description() string  { return "Coloriser for dpkg logs." }

func (p *DpkgPlugin) Handle(line string) (bool, string) {
	// Try status line
	if m := p.reStatus.FindStringSubmatch(line); m != nil {
		date := m[1]
		state := m[2]
		pkg := m[3]
		installedVersion := m[4]

		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Keyword, "status")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.PkgStatus, state)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Pkg, pkg)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, installedVersion)
		p.ct.WriteNewline(p.w)

		return true, ""
	}

	// Try action line
	if m := p.reAction.FindStringSubmatch(line); m != nil {
		date := m[1]
		action := m[2]
		pkg := m[3]
		installedVersion := m[4]
		availableVersion := m[5]

		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Keyword, action)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Pkg, pkg)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, installedVersion)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, availableVersion)
		p.ct.WriteNewline(p.w)

		return true, ""
	}

	// Try conffile line
	if m := p.reConffile.FindStringSubmatch(line); m != nil {
		date := m[1]
		filename := m[2]
		decision := m[3]

		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Keyword, "conffile")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.File, filename)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Keyword, decision)
		p.ct.WriteNewline(p.w)

		return true, ""
	}

	return false, ""
}
