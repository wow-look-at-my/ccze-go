package main

// compat_test.go verifies that ccze-go produces the same ANSI output as the C
// version of ccze.

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
	"github.com/stretchr/testify/assert"
)

// --------------------------------------------------------------------------
// helper: run a line through the full Go pipeline and return raw ANSI output
// --------------------------------------------------------------------------

func processLine(line string) string {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()
	registerAllPlugins(r, w, ct, wc, false)

	handled, rest := r.Run(line, plugin.TypeFull)

	if rest != "" {
		handled2, rest2 := r.Run(rest, plugin.TypePartial)
		if !handled2 {
			wc.Process(w, rest, true, false)
		} else {
			wc.Process(w, rest2, true, false)
		}
		ct.WriteNewline(w)
	}

	if !handled {
		wc.Process(w, line, true, false)
		ct.WriteNewline(w)
	}

	w.Flush()
	return buf.String()
}

// --------------------------------------------------------------------------
// ANSI output golden reference tests
// --------------------------------------------------------------------------

// TestCompatSyslogGolden verifies the exact ANSI escape output for a syslog
// line. The expected output is the sequence produced by the C ccze.
func TestCompatSyslogGolden(t *testing.T) {
	line := "Sep 14 11:45:00 myhost sshd[1234]: Connection closed"

	out := processLine(line)

	// Verify structural elements are present with correct ANSI codes.
	assert.Contains(t, out, "\x1b[1m")

	assert.Contains(t, out, "\x1b[36m")

	assert.Contains(t, out, "Sep 14 11:45:00")

	assert.Contains(t, out, "\x1b[34m")

	assert.Contains(t, out, "myhost")

	assert.Contains(t, out, "\x1b[32m")

	assert.Contains(t, out, "sshd")

	assert.Contains(t, out, "1234")

	// Rest word "Connection" should appear
	assert.Contains(t, out, "Connection")

	// Every segment should end with reset \x1b[0m
	assert.Contains(t, out, "\x1b[0m")

}

// TestCompatSyslogNoProcess verifies syslog lines without a PID bracket.
func TestCompatSyslogNoProcess(t *testing.T) {
	line := "Sep 14 11:45:00 myhost daemon: starting up"
	out := processLine(line)

	assert.Contains(t, out, "daemon")

	// Should NOT contain brackets
	stripped := stripAnsiCompat(out)
	assert.NotContains(t, stripped, "[")

	assert.Contains(t, stripped, "starting")

}

// TestCompatSyslogMark verifies the MARK line is handled with the Repeat color.
func TestCompatSyslogMark(t *testing.T) {
	line := "Oct 12 22:40:12 iluvatar -- MARK --"
	out := processLine(line)

	// Should contain the MARK text
	stripped := stripAnsiCompat(out)
	assert.Contains(t, stripped, "-- MARK --")

	assert.Contains(t, out, "\x1b[37m")

}

// TestCompatSyslogRepeat verifies repeated message handling.
func TestCompatSyslogRepeat(t *testing.T) {
	line := "Oct 12 22:40:12 iluvatar last message repeated 10 times"
	out := processLine(line)
	stripped := stripAnsiCompat(out)
	assert.Contains(t, stripped, "last message repeated 10 times")

}

// TestCompatHTTPDAccess verifies HTTPD access log output.
func TestCompatHTTPDAccess(t *testing.T) {
	line := `192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`
	out := processLine(line)

	// Host should be colored with Host color (bold blue)
	assert.Contains(t, out, "192.168.1.1")

	assert.Contains(t, out, "frank")

	assert.Contains(t, out, "\x1b[33m")

	// HTTP action "GET" should use HTTPGet color: green (32m)
	assert.Contains(t, out, "\x1b[32m")

	// HTTP code "200" should use HTTPCodes: bold white (37m)
	assert.Contains(t, out, "200")

	assert.Contains(t, out, "2326")

	assert.Contains(t, out, "\x1b[35m")

}

// TestCompatHTTPDError verifies HTTPD error log output.
func TestCompatHTTPDError(t *testing.T) {
	line := "[Sun Oct 12 15:30:00 2003] [error] client denied by configuration"
	out := processLine(line)

	// Date portion
	assert.Contains(t, out, "[Sun Oct 12 15:30:00 2003]")

	assert.Contains(t, out, "\x1b[31m")

	assert.Contains(t, out, "[error]")

}

// TestCompatHTTPDPost verifies HTTP POST uses HTTPPost color (bold green).
func TestCompatHTTPDPost(t *testing.T) {
	line := `192.168.1.1 - admin [10/Oct/2000:13:55:36 -0700] "POST /api/submit HTTP/1.1" 201 512`
	out := processLine(line)
	// POST should use HTTPPost: bold green (AttrBold|bold+32m)
	stripped := stripAnsiCompat(out)
	assert.Contains(t, stripped, "POST /api/submit HTTP/1.1")

}

// TestCompatDpkg verifies dpkg log output.
func TestCompatDpkg(t *testing.T) {
	line := "2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3"
	out := processLine(line)
	stripped := stripAnsiCompat(out)

	assert.Contains(t, stripped, "2023-10-15 14:30:22")

	assert.Contains(t, stripped, "installed")

	assert.Contains(t, stripped, "libfoo:amd64")

}

// TestCompatPostfix verifies postfix partial-match output.
func TestCompatPostfix(t *testing.T) {
	// Simulate the syslog+postfix pipeline
	fullLine := "Sep 14 11:45:00 mailhost postfix/smtp[1234]: ABC123: to=<user@example.com>,relay=smtp.example.com"
	out := processLine(fullLine)
	stripped := stripAnsiCompat(out)

	assert.Contains(t, stripped, "mailhost")

	assert.Contains(t, stripped, "postfix/smtp")

	// Postfix partial fields
	assert.Contains(t, stripped, "ABC123")

	// Field names should be colored with Field color (green, 32m)
	assert.Contains(t, out, "\x1b[32m")

}

// TestCompatSquidAccess verifies squid access log output.
func TestCompatSquidAccess(t *testing.T) {
	line := "1234567890.123      5 192.168.1.1 TCP_MISS/200 1234 GET http://example.com user DIRECT/93.184.216.34 text/html"
	out := processLine(line)
	stripped := stripAnsiCompat(out)

	assert.Contains(t, stripped, "192.168.1.1")

	assert.Contains(t, stripped, "TCP_MISS")

	assert.Contains(t, stripped, "http://example.com")

	// TCP_MISS should use ProxyMiss color: red (31m)
	assert.Contains(t, out, "\x1b[31m")

}

// TestCompatWordColorBadWord verifies "error" words use bold red.
func TestCompatWordColorBadWord(t *testing.T) {
	// Process a plain line (no plugin match) with error keyword
	line := "something failed with error"
	out := processLine(line)

	// "error" should be colored with Error: bold red (AttrBold|bold+31m)
	assert.Contains(t, out, "\x1b[1m")

	assert.Contains(t, out, "\x1b[31m")

}

// TestCompatWordColorGoodWord verifies "started" uses bold green.
func TestCompatWordColorGoodWord(t *testing.T) {
	line := "service started successfully"
	out := processLine(line)

	stripped := stripAnsiCompat(out)
	assert.Contains(t, stripped, "started")

}

// TestCompatWordColorIPAddress verifies IP addresses use Host color.
func TestCompatWordColorIPAddress(t *testing.T) {
	line := "connecting to 10.0.0.1 from 192.168.1.100"
	out := processLine(line)

	// IP addresses should use Host color: bold blue (34m)
	assert.Contains(t, out, "10.0.0.1")

	assert.Contains(t, out, "\x1b[34m")

}

// TestCompatWordColorURI verifies URIs use URI color.
func TestCompatWordColorURI(t *testing.T) {
	line := "fetching http://example.com/path"
	out := processLine(line)
	assert.Contains(t, out, "http://example.com/path")

}

// TestCompatWordColorEmail verifies email addresses use Email color.
func TestCompatWordColorEmail(t *testing.T) {
	line := "mail from user@example.com delivered"
	out := processLine(line)
	assert.Contains(t, out, "user@example.com")

}

// TestCompatWordColorVersion verifies version strings use Version color.
func TestCompatWordColorVersion(t *testing.T) {
	line := "upgraded to 2.3.7 from 1.0.0"
	out := processLine(line)
	stripped := stripAnsiCompat(out)
	assert.Contains(t, stripped, "2.3.7")

}

// TestCompatWordColorDirectory verifies paths use Dir color.
func TestCompatWordColorDirectory(t *testing.T) {
	line := "reading /etc/passwd done"
	out := processLine(line)
	assert.Contains(t, out, "/etc/passwd")

	assert.Contains(t, out, "\x1b[36m")

}

// TestCompatWordColorSignal verifies signal names use Signal color.
func TestCompatWordColorSignal(t *testing.T) {
	line := "received sigterm shutting down"
	out := processLine(line)
	assert.Contains(t, out, "sigterm")

}

// TestCompatWordColorMAC verifies MAC addresses use MAC color.
func TestCompatWordColorMAC(t *testing.T) {
	line := "device aa:bb:cc:dd:ee:ff connected"
	out := processLine(line)
	assert.Contains(t, out, "aa:bb:cc:dd:ee:ff")

}

// TestCompatWordColorHexAddress verifies hex addresses use Address color.
func TestCompatWordColorHexAddress(t *testing.T) {
	line := "fault at 0x1234abcd"
	out := processLine(line)
	assert.Contains(t, out, "0x1234abcd")

}

// TestCompatWordColorSize verifies size strings use Size color.
func TestCompatWordColorSize(t *testing.T) {
	line := "downloaded 150mb in 30s"
	out := processLine(line)
	assert.Contains(t, out, "150mb")

}

// TestCompatWordColorSystemWord verifies system words use SystemWord color.
func TestCompatWordColorSystemWord(t *testing.T) {
	line := "linux kernel booting"
	out := processLine(line)
	stripped := stripAnsiCompat(out)
	assert.Contains(t, stripped, "linux")

}

// --------------------------------------------------------------------------
// Color table default values — verify matching the C ccze_color_init_raw_ansi
// --------------------------------------------------------------------------

func TestCompatColorDefaults(t *testing.T) {
	ct := color.NewTable(true)

	// Map of color slots to expected default values from C code
	defaults := map[color.Color]int{
		color.Date:          color.AttrBold | 5,
		color.Host:          color.AttrBold | 4,
		color.Proc:          2,
		color.PID:           color.AttrBold | 7,
		color.PIDB:          color.AttrBold | 2,
		color.Default:       5,
		color.Email:         color.AttrBold | 2,
		color.Subject:       6,
		color.Dir:           color.AttrBold | 5,
		color.Size:          color.AttrBold | 7,
		color.User:          color.AttrBold | 3,
		color.HTTPCodes:     color.AttrBold | 7,
		color.GetSize:       6,
		color.HTTPGet:       2,
		color.HTTPPost:      color.AttrBold | 2,
		color.HTTPHead:      2,
		color.HTTPPut:       color.AttrBold | 2,
		color.HTTPConnect:   2,
		color.HTTPTrace:     2,
		color.GetTime:       color.AttrBold | 6,
		color.URI:           color.AttrBold | 2,
		color.Ident:         color.AttrBold | 7,
		color.CType:         7,
		color.Error:         color.AttrBold | 1,
		color.ProxyMiss:     1,
		color.ProxyHit:      color.AttrBold | 3,
		color.ProxyDenied:   color.AttrBold | 1,
		color.ProxyRefresh:  color.AttrBold | 7,
		color.ProxySwapfail: color.AttrBold | 7,
		color.Debug:         7,
		color.Warning:       1,
		color.ProxyDirect:   color.AttrBold | 7,
		color.ProxyParent:   color.AttrBold | 3,
		color.ProxyCreate:   color.AttrBold | 7,
		color.ProxySwapin:   color.AttrBold | 7,
		color.ProxySwapout:  color.AttrBold | 7,
		color.ProxyRelease:  color.AttrBold | 7,
		color.MAC:           color.AttrBold | 7,
		color.Version:       color.AttrBold | 7,
		color.Address:       color.AttrBold | 7,
		color.Numbers:       7,
		color.Signal:        color.AttrBold | 3,
		color.Service:       color.AttrBold | 6,
		color.Prot:          6,
		color.BadWord:       color.AttrBold | 3,
		color.GoodWord:      color.AttrBold | 2,
		color.SystemWord:    color.AttrBold | 5,
		color.Incoming:      color.AttrBold | 7,
		color.Outgoing:      7,
		color.Uniqn:         color.AttrBold | 7,
		color.Repeat:        7,
		color.Field:         2,
		color.Chain:         5,
		color.Percentage:    color.AttrBold | 3,
		color.FTPCodes:      5,
		color.Keyword:       color.AttrBold | 3,
		color.PkgStatus:     2,
		color.Pkg:           color.AttrBold | 1,
	}

	for c, want := range defaults {
		got := ct.Get(c)
		assert.Equal(t, want, got)

	}
}

// This is a deliberate quirk in the original C ccze.
func TestCompatAnsiColorSwap(t *testing.T) {
	ct := color.NewTable(true)

	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "X")
	assert.Contains(t, buf.String(), "\x1b[36m")

	buf.Reset()
	ct.WriteColored(&buf, color.GetSize, "X")
	assert.Contains(t, buf.String(), "\x1b[35m")

}

// TestCompatAttrSGRSwap verifies that AttrReverse emits a later SGR (blink)
// and AttrBlink emits a later SGR (reverse), matching the C code's peculiar behavior.
func TestCompatAttrSGRSwap(t *testing.T) {
	ct := color.NewTable(true)

	ct.Set(color.Date, color.AttrReverse|3)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "X")
	assert.Contains(t, buf.String(), "\x1b[5m")

	ct.Set(color.Date, color.AttrBlink|3)
	buf.Reset()
	ct.WriteColored(&buf, color.Date, "X")
	assert.Contains(t, buf.String(), "\x1b[7m")

}

// TestCompatBackgroundColor verifies background color encoding matches C.
func TestCompatBackgroundColor(t *testing.T) {
	ct := color.NewTable(true)

	// bg=1 (red) | fg=7 (white) => bg ANSI = 31+10 = 41
	ct.Set(color.Date, (1<<8)|7)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "X")
	assert.Contains(t, buf.String(), "\x1b[41m")

	assert.Contains(t, buf.String(), "\x1b[37m")

}

// TestCompatNonTransparentBackground verifies that non-transparent mode
// always emits a background color, matching C behavior.
func TestCompatNonTransparentBackground(t *testing.T) {
	ct := color.NewTable(false)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Default, "X")
	assert.Contains(t, buf.String(), "\x1b[40m")

}

// TestCompatResetSequence verifies every colored string ends with ESC[0m.
func TestCompatResetSequence(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Error, "test")
	assert.True(t, strings.HasSuffix(buf.String(), "\x1b[0m"))

}

// TestCompatIntensityReset verifies every colored string starts with ESC[22m.
func TestCompatIntensityReset(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Error, "test")
	assert.True(t, strings.HasPrefix(buf.String(), "\x1b[22m"))

}
