// Package wordcolor implements word-level colorization for CCZE log output.
package wordcolor

import (
	"io"
	"regexp"
	"strings"

	"ccze-go/color"
	"ccze-go/wordcolor/matchers"
)

// Processor holds precompiled regular expressions and a color table for
// performing word-level colorization of log messages.
//
// Most match-only patterns have been replaced with generated DFA matchers
// from go-regex-compiler. Only regPre and regPost remain as *regexp.Regexp
// because they require capture group extraction (FindStringSubmatch).
type Processor struct {
	ct      *color.Table
	regPre  *regexp.Regexp // punctuation prefix  (needs capture groups)
	regPost *regexp.Regexp // punctuation suffix  (needs capture groups)
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

// New creates a new Processor with all regular expressions compiled and
// the given color table.
func New(ct *color.Table) *Processor {
	return &Processor{
		ct:      ct,
		regPre:  regexp.MustCompile("^([`'\".,!?:;(\\[{<]+)([^`'\".,!?:;(\\[{<]\\S*)$"),
		regPost: regexp.MustCompile("^(\\S*[^`'\".,!?:;)\\]}>])([`'\".,!?:;)\\]}>]+)$"),
	}
}

// matchHost uses the generated DFA matcher for host patterns.
func matchHost(s string) bool { return matchers.MatchHost(s) }

// matchHostIP uses the generated DFA matcher for host[IP] patterns.
func matchHostIP(s string) bool { return matchers.MatchHostIP(s) }

// matchMAC uses the generated DFA matcher for MAC address patterns.
func matchMAC(s string) bool { return matchers.MatchMAC(s) }

// matchEmail uses the generated DFA matcher for email patterns.
func matchEmail(s string) bool { return matchers.MatchEmail(s) }

// matchEmail2 uses the generated DFA matcher for email domain suffix patterns.
func matchEmail2(s string) bool { return matchers.MatchEmail2(s) }

// matchMsgID uses the generated DFA matcher for message ID patterns.
func matchMsgID(s string) bool { return matchers.MatchMsgID(s) }

// matchURI uses the generated DFA matcher for URI patterns.
func matchURI(s string) bool { return matchers.MatchURI(s) }

// matchSize uses the generated DFA matcher for file size patterns.
func matchSize(s string) bool { return matchers.MatchSize(s) }

// matchVer uses the generated DFA matcher for version number patterns.
func matchVer(s string) bool { return matchers.MatchVer(s) }

// matchTime uses the generated DFA matcher for time patterns.
func matchTime(s string) bool { return matchers.MatchTime(s) }

// matchAddr uses the generated DFA matcher for hex address patterns.
func matchAddr(s string) bool { return matchers.MatchAddr(s) }

// matchNum uses the generated DFA matcher for number patterns.
func matchNum(s string) bool { return matchers.MatchNum(s) }

// matchSig uses the generated DFA matcher for signal name patterns.
func matchSig(s string) bool { return matchers.MatchSig(s) }

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
	var pre string
	if m := p.regPre.FindStringSubmatch(word); m != nil {
		pre = m[1]
		word = m[2]
	}

	// Extract punctuation suffix
	var post string
	if m := p.regPost.FindStringSubmatch(word); m != nil {
		post = m[2]
		word = m[1]
	}

	lword := strings.ToLower(word)

	// Pattern cascade - order matters, first match wins (except hostip)
	// All match-only patterns use generated DFA matchers from go-regex-compiler.
	switch {
	case matchHost(lword):
		col = color.Host
	case matchMAC(lword):
		col = color.MAC
	case len(lword) > 0 && lword[0] == '/':
		col = color.Dir
	case matchEmail(lword) && matchEmail2(lword):
		col = color.Email
	case matchMsgID(lword):
		col = color.Email
	case matchURI(lword):
		col = color.URI
	case matchSize(lword):
		col = color.Size
	case matchVer(lword):
		col = color.Version
	case matchTime(lword):
		col = color.Date
	case matchAddr(lword):
		col = color.Address
	case matchNum(lword):
		col = color.Numbers
	case matchSig(lword):
		col = color.Signal
	case matchHostIP(lword):
		// Special handling: split at '[', output host and IP separately.
		// By this point the regPost has already stripped any trailing ']',
		// so the word looks like "hostname[192.168.1.1".
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
		// Service, protocol, and user lookups (slookup).
		// These are skipped for now; can be implemented later.

		// Keyword matching: check bad, good, error, system word lists.
		// The last matching list wins (later lists override earlier ones).
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
