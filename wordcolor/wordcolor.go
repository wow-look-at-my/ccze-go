// Package wordcolor implements word-level colorization for CCZE log output.
package wordcolor

import (
	"io"
	"regexp"
	"strings"

	"ccze-go/color"
)

// Processor holds precompiled regular expressions and a color table for
// performing word-level colorization of log messages.
type Processor struct {
	ct        *color.Table
	regPre    *regexp.Regexp // punctuation prefix
	regPost   *regexp.Regexp // punctuation suffix
	regHost   *regexp.Regexp
	regHostIP *regexp.Regexp
	regMAC    *regexp.Regexp
	regEmail  *regexp.Regexp
	regEmail2 *regexp.Regexp
	regMsgID  *regexp.Regexp
	regURI    *regexp.Regexp
	regSize   *regexp.Regexp
	regVer    *regexp.Regexp
	regTime   *regexp.Regexp
	regAddr   *regexp.Regexp
	regNum    *regexp.Regexp
	regSig    *regexp.Regexp
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
		ct:        ct,
		regPre:    regexp.MustCompile("^([`'\".,!?:;(\\[{<]+)([^`'\".,!?:;(\\[{<]\\S*)$"),
		regPost:   regexp.MustCompile("^(\\S*[^`'\".,!?:;)\\]}>])([`'\".,!?:;)\\]}>]+)$"),
		regHost:   regexp.MustCompile("^(((\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})|(([a-z0-9-_]+\\.)+[a-z]{2,3})|(localhost)|(\\w*::\\w+)+)(:\\d{1,5})?)$"),
		regHostIP: regexp.MustCompile("^(((\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})|(([a-z0-9-_\\.]+)+)|(localhost)|(\\w*::\\w+)+)(:\\d{1,5})?)\\["),
		regMAC:    regexp.MustCompile("^([0-9a-f]{2}:){5}[0-9a-f]{2}$"),
		regEmail:  regexp.MustCompile("^[a-z0-9-_=\\+]+@([a-z0-9-_\\.]+)+(\\.[a-z]{2,4})+"),
		regEmail2: regexp.MustCompile("(\\.[a-z]{2,4})+$"),
		regMsgID:  regexp.MustCompile("^[a-z0-9-_\\.\\$=\\+]+@([a-z0-9-_\\.]+)+(\\.[a-z]+)+"),
		regURI:    regexp.MustCompile("^\\w{2,}:\\/\\/(\\S+\\/?)+$"),
		regSize:   regexp.MustCompile("^\\d+(\\.\\d+)?[kmgt]i?b?(ytes?)?"),
		regVer:    regexp.MustCompile("^v?(\\d+\\.){1}((\\d|[a-z])+\\.)*(\\d|[a-z])+$"),
		regTime:   regexp.MustCompile("\\d{1,2}:\\d{1,2}(:\\d{1,2})?"),
		regAddr:   regexp.MustCompile("^0x(\\d|[a-f])+$"),
		regNum:    regexp.MustCompile("^[+-]?\\d+$"),
		regSig:    regexp.MustCompile("^sig(hup|int|quit|ill|abrt|fpe|kill|segv|pipe|alrm|term|usr1|usr2|chld|cont|stop|tstp|tin|tout|bus|poll|prof|sys|trap|urg|vtalrm|xcpu|xfsz|iot|emt|stkflt|io|cld|pwr|info|lost|winch|unused)"),
	}
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
	switch {
	case p.regHost.MatchString(lword):
		col = color.Host
	case p.regMAC.MatchString(lword):
		col = color.MAC
	case len(lword) > 0 && lword[0] == '/':
		col = color.Dir
	case p.regEmail.MatchString(lword) && p.regEmail2.MatchString(lword):
		col = color.Email
	case p.regMsgID.MatchString(lword):
		col = color.Email
	case p.regURI.MatchString(lword):
		col = color.URI
	case p.regSize.MatchString(lword):
		col = color.Size
	case p.regVer.MatchString(lword):
		col = color.Version
	case p.regTime.MatchString(lword):
		col = color.Date
	case p.regAddr.MatchString(lword):
		col = color.Address
	case p.regNum.MatchString(lword):
		col = color.Numbers
	case p.regSig.MatchString(lword):
		col = color.Signal
	case p.regHostIP.MatchString(lword):
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
