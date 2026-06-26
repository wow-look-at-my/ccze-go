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
func (p *ProcmailPlugin) Type() Type          { return TypeFull }
func (p *ProcmailPlugin) Description() string { return "Coloriser for procmail(1) logs." }

// parseProcmail hand-parses a procmail log line.
// Format: ^(\s*)(>?From|Subject:|Folder:)?\s(\S+)(\s+)?(.*)
func parseProcmail(line string) (space1, header, value, space2, extra string, ok bool) {
	i := 0
	n := len(line)

	// Leading whitespace: \s*
	for i < n && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	space1 = line[:i]

	// Optional header keyword: (>?From|Subject:|Folder:)?
	rest := line[i:]
	header = ""
	for _, kw := range []string{">From", "From", "Subject:", "Folder:"} {
		if strings.HasPrefix(rest, kw) {
			header = kw
			rest = rest[len(kw):]
			break
		}
	}

	// \s — at least one whitespace char
	if len(rest) == 0 || (rest[0] != ' ' && rest[0] != '\t') {
		// If no header matched, backtrack one space from space1 (regex backtracking)
		if header == "" && i > 0 {
			i--
			space1 = line[:i]
			rest = line[i:]
			// Now consume the mandatory \s
			rest = rest[1:]
		} else {
			return
		}
	} else {
		// Skip exactly one space (the regex consumes one \s)
		rest = rest[1:]
	}

	// Value: \S+ (non-whitespace)
	j := 0
	for j < len(rest) && rest[j] != ' ' && rest[j] != '\t' {
		j++
	}
	if j == 0 {
		return
	}
	value = rest[:j]
	rest = rest[j:]

	// Optional space: (\s+)?
	k := 0
	for k < len(rest) && (rest[k] == ' ' || rest[k] == '\t') {
		k++
	}
	space2 = rest[:k]
	extra = rest[k:]

	ok = true
	return
}

func (p *ProcmailPlugin) Handle(line string) (bool, string) {
	space1, header, value, space2, extra, ok := parseProcmail(line)
	if !ok {
		return false, ""
	}

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
