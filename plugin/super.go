package plugin

import (
	"io"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// SuperPlugin is a FULL plugin.
// Coloriser for super(1) logs.
type SuperPlugin struct {
	w        io.Writer
	ct       *color.Table
	wc       *wordcolor.Processor
	convdate bool
}

// NewSuperPlugin creates a new SuperPlugin.
func NewSuperPlugin(w io.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) *SuperPlugin {
	return &SuperPlugin{
		w:        w,
		ct:       ct,
		wc:       wc,
		convdate: convdate,
	}
}

func (p *SuperPlugin) Name() string        { return "super" }
func (p *SuperPlugin) Type() Type           { return TypeFull }
func (p *SuperPlugin) Description() string  { return "Coloriser for super(1) logs." }

// parseSuper hand-parses a super log line.
// Format: ^(\S+)\s(\w+\s+\w+\s+\d+\s+\d+:\d+:\d+\s+\d+)(\s+)(\S+)\s\(([^\)]+)\)
func parseSuper(line string) (email, date, space, suptag, other string, ok bool) {
	// Email: first non-space token
	spIdx := strings.Index(line, " ")
	if spIdx < 1 {
		return
	}
	email = line[:spIdx]
	rest := line[spIdx+1:]

	// Date: "Mon Oct 15 14:30:22 2023" — word spaces word spaces digits spaces time spaces year
	// The date has the format: \w+\s+\w+\s+\d+\s+\d+:\d+:\d+\s+\d+
	// We need to consume 5 space-delimited tokens (day-of-week, month, day, time, year)
	dateStart := 0
	s := rest
	for tok := 0; tok < 5; tok++ {
		// Skip to next space
		idx := strings.Index(s, " ")
		if idx < 0 {
			return
		}
		s = s[idx+1:]
		// Skip extra spaces
		for len(s) > 0 && s[0] == ' ' {
			s = s[1:]
		}
	}
	// s now points past the date; calculate date end position
	dateLen := len(rest) - len(s)
	dateStr := rest[:dateLen]
	// Trim trailing space from date
	dateStr = strings.TrimRight(dateStr, " ")
	if !strings.Contains(dateStr, ":") {
		return
	}
	date = dateStr
	_ = dateStart

	// Space between date and suptag
	spaceStart := dateLen
	for spaceStart < len(rest) && rest[spaceStart] == ' ' {
		spaceStart++
	}
	// Actually the space is from after dateStr to where s starts, minus 1 we already consumed
	space = rest[len(dateStr):spaceStart]
	rest = rest[spaceStart:]

	// Suptag: \S+ (non-space)
	idx := strings.Index(rest, " ")
	if idx < 1 {
		return
	}
	suptag = rest[:idx]
	rest = rest[idx+1:]

	// (other): \(([^\)]+)\)
	if len(rest) == 0 || rest[0] != '(' {
		return
	}
	closeIdx := strings.Index(rest, ")")
	if closeIdx < 2 {
		return
	}
	other = rest[1:closeIdx]

	ok = true
	return
}

// Handle attempts to match and colorize a super log line.
func (p *SuperPlugin) Handle(line string) (bool, string) {
	email, date, space, suptag, other, ok := parseSuper(line)
	if !ok {
		return false, ""
	}

	p.ct.WriteColored(p.w, color.Email, email)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.Date, date)
	p.ct.WriteColored(p.w, color.Default, space)
	p.ct.WriteColored(p.w, color.Proc, suptag)
	p.ct.WriteSpace(p.w)
	p.ct.WriteColored(p.w, color.PIDB, "(")
	p.ct.WriteColored(p.w, color.Default, other)
	p.ct.WriteColored(p.w, color.PIDB, ")")

	p.ct.WriteNewline(p.w)

	return true, ""
}
