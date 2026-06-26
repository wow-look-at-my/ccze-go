package plugin

import (
	"io"
	"strings"

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
func (p *DpkgPlugin) Type() Type          { return TypeFull }
func (p *DpkgPlugin) Description() string { return "Coloriser for dpkg logs." }

// parseDpkgDate extracts the date prefix from a dpkg log line.
// Format: [-\d]{10}\s[:\d]{8}  (e.g. "2023-10-15 14:30:22")
// Returns the date, the remainder after the space, and whether parsing succeeded.
func parseDpkgDate(line string) (date, rest string, ok bool) {
	// Need at least 20 chars: 10 (date) + 1 (space) + 8 (time) + 1 (space)
	if len(line) < 20 {
		return
	}
	for i := 0; i < 10; i++ {
		c := line[i]
		if c != '-' && (c < '0' || c > '9') {
			return
		}
	}
	if line[10] != ' ' {
		return
	}
	for i := 11; i < 19; i++ {
		c := line[i]
		if c != ':' && (c < '0' || c > '9') {
			return
		}
	}
	if line[19] != ' ' {
		return
	}
	date = line[:19]
	rest = line[20:]
	ok = true
	return
}

// splitSpaceFields splits s into up to n space-delimited fields.
// Returns the fields and whether exactly n were found.
func splitSpaceFields(s string, n int) ([]string, bool) {
	fields := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := strings.Index(s, " ")
		if idx < 0 {
			return nil, false
		}
		fields = append(fields, s[:idx])
		s = s[idx+1:]
	}
	fields = append(fields, s)
	return fields, true
}

func (p *DpkgPlugin) Handle(line string) (bool, string) {
	date, rest, ok := parseDpkgDate(line)
	if !ok {
		return false, ""
	}

	// Branch on keyword after date
	if strings.HasPrefix(rest, "status ") {
		// Status line: status <state> <pkg> <version>
		fields, ok := splitSpaceFields(rest[7:], 3)
		if !ok {
			return false, ""
		}
		state, pkg, ver := fields[0], fields[1], fields[2]
		// Reject if any field contains spaces (i.e. final field shouldn't have extra)
		if strings.Contains(ver, " ") {
			return false, ""
		}

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

	if strings.HasPrefix(rest, "conffile ") {
		// Conffile line: conffile <filename> <install|keep>
		fields, ok := splitSpaceFields(rest[9:], 2)
		if !ok {
			return false, ""
		}
		filename, decision := fields[0], fields[1]
		if decision != "install" && decision != "keep" {
			return false, ""
		}

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

	// Action line: <action> <pkg> <installed_ver> <available_ver>
	// Actions: install, upgrade, remove, purge
	for _, action := range []string{"install", "upgrade", "remove", "purge"} {
		prefix := action + " "
		if strings.HasPrefix(rest, prefix) {
			fields, ok := splitSpaceFields(rest[len(prefix):], 3)
			if !ok {
				continue
			}
			pkg, installedVer, availableVer := fields[0], fields[1], fields[2]
			if strings.Contains(availableVer, " ") {
				continue
			}

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
	}

	return false, ""
}
