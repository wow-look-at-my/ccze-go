package plugin

import (
	"io"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// DpkgPlugin colorizes dpkg log lines.
type DpkgPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewDpkgPlugin creates a new DpkgPlugin.
func NewDpkgPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *DpkgPlugin {
	return &DpkgPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *DpkgPlugin) Name() string        { return "dpkg" }
func (p *DpkgPlugin) Type() Type           { return TypeFull }
func (p *DpkgPlugin) Description() string  { return "Coloriser for dpkg logs." }

func (p *DpkgPlugin) Handle(line string) (bool, string) {
	// Try status line
	if m := dpkgStatusFindSubmatch(line); m != nil {
		date, state, pkg, ver := m[1], m[2], m[3], m[4]
		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Keyword, "status")
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.PkgStatus, state)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Pkg, pkg)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, ver)
		p.ct.WriteNewline(p.w)
		return true, ""
	}

	// Try conffile line
	if m := dpkgConffileFindSubmatch(line); m != nil {
		date, filename, decision := m[1], m[2], m[3]
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

	// Try action line
	if m := dpkgActionFindSubmatch(line); m != nil {
		date, action, pkg, installedVer, availableVer := m[1], m[2], m[3], m[4], m[5]
		PrintDate(p.w, p.ct, date, p.convdate)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Keyword, action)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Pkg, pkg)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, installedVer)
		p.ct.WriteSpace(p.w)
		p.ct.WriteColored(p.w, color.Default, availableVer)
		p.ct.WriteNewline(p.w)
		return true, ""
	}

	return false, ""
}
