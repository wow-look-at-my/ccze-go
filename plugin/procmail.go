package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// ProcmailPlugin colorizes procmail(1) log lines.
type ProcmailPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewProcmailPlugin creates a new ProcmailPlugin.
func NewProcmailPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *ProcmailPlugin {
	return &ProcmailPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *ProcmailPlugin) Name() string        { return "procmail" }
func (p *ProcmailPlugin) Type() Type           { return TypeFull }
func (p *ProcmailPlugin) Description() string  { return "Coloriser for procmail(1) logs." }

func (p *ProcmailPlugin) Handle(line string) (bool, string) {
	m := procmailFindSubmatch(line)
	if m == nil {
		return false, ""
	}

	space1 := m[1]
	header := m[2]
	value := m[3]
	space2 := m[4]
	extra := m[5]

	headerLower := strings.ToLower(header)

	var col color.Color
	handled := false

	switch headerLower {
	case "from", ">from":
		col = color.Email
		handled = true
	case "subject:":
		col = color.Subject
		handled = true
	case "folder:":
		col = color.Dir
		handled = true
	}

	if !handled {
		// Return the original line as rest
		return true, line
	}

	p.ct.WriteColored(p.w, color.Default, space1)
	p.ct.WriteColored(p.w, color.Default, header)
	p.ct.WriteSpace(p.w)

	p.ct.WriteColored(p.w, col, value)

	// After email value, switch to Default for space2
	extraCol := col
	if col == color.Email {
		extraCol = color.Default
	}
	p.ct.WriteColored(p.w, extraCol, space2)

	// Determine color for extra field
	if headerLower == "folder:" {
		extraCol = color.Size
	} else if headerLower == "from" || headerLower == ">from" {
		extraCol = color.Date
	}

	p.ct.WriteColored(p.w, extraCol, extra)
	p.ct.WriteNewline(p.w)

	return true, ""
}
