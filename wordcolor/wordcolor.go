// Package wordcolor implements word-level colorization for CCZE log output.
package wordcolor

import (
	"io"
	"strings"

	"ccze-go/color"
)

// Processor holds a color table for performing word-level colorization of
// log messages. Pattern matching is done with generated DFA-based matchers
// from go-regex-compiler for consistent behavior and no runtime regex overhead.
type Processor struct {
	ct *color.Table
}

var wordsBad = []string{
	"warn", "restart", "exit", "stop", "end", "shutting", "down", "close",
	"unreach", "can't", "cannot", "skip", "deny", "disable", "ignored",
	"miss", "oops", "not", "backdoor", "blocking", "ignoring",
	"unable", "readonly", "offline", "terminate", "empty", "virus",
}

var wordsGood = []string{
	"activ", "start", "ready", "online", "load", "ok", "register", "detected",
	"configured", "enable", "listen", "open", "complete", "attempt", "done",
	"check", "listen", "connect", "finish", "clean",
}

var wordsError = []string{
	"error", "crit", "invalid", "fail", "false", "alarm", "fatal",
}

var wordsSystem = []string{
	"ext2-fs", "reiserfs", "vfs", "iso", "isofs", "cslip", "ppp", "bsd",
	"linux", "tcp/ip", "mtrr", "pci", "isa", "scsi", "ide", "atapi",
	"bios", "cpu", "fpu", "discharging", "resume",
}

// New creates a new Processor with the given color table.
func New(ct *color.Table) *Processor {
	return &Processor{ct: ct}
}

// splitPre strips leading punctuation from word using the generated DFA matcher.
func splitPre(word string) (pre, rest string) {
	m := regPreFindSubmatch(word)
	if m == nil {
		return "", word
	}
	return m[1], m[2]
}

// splitPost strips trailing punctuation from word using the generated DFA matcher.
func splitPost(word string) (body, post string) {
	m := regPostFindSubmatch(word)
	if m == nil {
		return word, ""
	}
	return m[1], m[2]
}

// Process colorizes an entire message string, writing the result to w.
// If wcol is false, the message is written with the Default color.
// If slookup is true, service/protocol/user lookups may be attempted.
func (p *Processor) Process(w io.Writer, msg string, wcol bool, slookup bool) {
	if msg == "" {
		return
	}

	if !wcol {
		p.ct.WriteColored(w, color.Default, msg)
		return
	}

	// Check for repeated message or MARK
	if (strings.Contains(msg, "last message repeated") && strings.Contains(msg, "times")) ||
		strings.Contains(msg, "-- MARK --") {
		p.ct.WriteColored(w, color.Repeat, msg)
		return
	}

	words := strings.Split(msg, " ")
	for i, word := range words {
		if word == "" {
			// Preserve spacing: empty words from consecutive spaces
			if i < len(words)-1 {
				p.ct.WriteSpace(w)
			}
			continue
		}
		p.ProcessOne(w, word, slookup)
		if i < len(words)-1 {
			p.ct.WriteSpace(w)
		}
	}
}

// ProcessOne colorizes a single word and writes it to w.
func (p *Processor) ProcessOne(w io.Writer, word string, slookup bool) {
	col := color.Default
	printed := false

	// Extract punctuation prefix
	pre, word := splitPre(word)

	// Extract punctuation suffix
	word, post := splitPost(word)

	lword := strings.ToLower(word)

	// Pattern cascade - order matters, first match wins (except hostip)
	switch {
	case regHostMatch(lword):
		col = color.Host
	case regMACMatch(lword):
		col = color.MAC
	case len(lword) > 0 && lword[0] == '/':
		col = color.Dir
	case regEmailMatch(lword) && regEmail2Match(lword):
		col = color.Email
	case regMsgIDMatch(lword):
		col = color.Email
	case regURIMatch(lword):
		col = color.URI
	case regSizeMatch(lword):
		col = color.Size
	case regVerMatch(lword):
		col = color.Version
	case regTimeMatch(lword):
		col = color.Date
	case regAddrMatch(lword):
		col = color.Address
	case regNumMatch(lword):
		col = color.Numbers
	case regSigMatch(lword):
		col = color.Signal
	case regHostIPMatch(lword):
		// Special handling: split at '[', output host and IP separately.
		idx := strings.Index(word, "[")
		if idx < 0 {
			idx = strings.Index(lword, "[")
		}
		if idx >= 0 {
			host := word[:idx]
			ip := word[idx+1:]
			p.ct.WriteColored(w, color.Default, pre)
			p.ct.WriteColored(w, color.Host, host)
			p.ct.WriteColored(w, color.PIDB, "[")
			p.ct.WriteColored(w, color.Host, ip)
			p.ct.WriteColored(w, color.PIDB, "]")
			p.ct.WriteColored(w, color.Default, post)
			printed = true
		}
	default:
		// Keyword matching: check bad, good, error, system word lists.
		for _, kw := range wordsBad {
			if strings.HasPrefix(lword, kw) {
				col = color.BadWord
			}
		}
		for _, kw := range wordsGood {
			if strings.HasPrefix(lword, kw) {
				col = color.GoodWord
			}
		}
		for _, kw := range wordsError {
			if strings.HasPrefix(lword, kw) {
				col = color.Error
			}
		}
		for _, kw := range wordsSystem {
			if strings.HasPrefix(lword, kw) {
				col = color.SystemWord
			}
		}
	}

	if !printed {
		p.ct.WriteColored(w, color.Default, pre)
		p.ct.WriteColored(w, col, word)
		p.ct.WriteColored(w, color.Default, post)
	}
}
