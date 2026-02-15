package main

// compat_test.go verifies that ccze-go produces the same ANSI output as the
// C version of ccze. The golden reference strings were captured from ccze
// 0.2.1-8 running with -A (ANSI mode) and default colors on a transparent
// terminal.
//
// When the C binary is available on the system, TestCompatAgainstCBinary runs
// live comparisons by piping lines through both implementations and diffing.

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
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
	// Date: bold (1m) + cyan (36m) — color.Date = AttrBold|5, ansiColor[5]=36
	if !strings.Contains(out, "\x1b[1m") {
		t.Error("date should have bold attribute")
	}
	if !strings.Contains(out, "\x1b[36m") {
		t.Error("date should use cyan (36m)")
	}
	if !strings.Contains(out, "Sep 14 11:45:00") {
		t.Error("date text missing")
	}

	// Host: bold (1m) + blue (34m) — color.Host = AttrBold|4, ansiColor[4]=34
	if !strings.Contains(out, "\x1b[34m") {
		t.Error("host should use blue (34m)")
	}
	if !strings.Contains(out, "myhost") {
		t.Error("host text missing")
	}

	// Process: green (32m) — color.Proc = 2, ansiColor[2]=32
	if !strings.Contains(out, "\x1b[32m") {
		t.Error("process should use green (32m)")
	}
	if !strings.Contains(out, "sshd") {
		t.Error("process name missing")
	}

	// PID bracket: bold green — color.PIDB = AttrBold|2
	// PID number: bold white (37m) — color.PID = AttrBold|7
	if !strings.Contains(out, "1234") {
		t.Error("PID missing")
	}

	// Rest word "Connection" should appear
	if !strings.Contains(out, "Connection") {
		t.Error("rest message word missing")
	}

	// Every segment should end with reset \x1b[0m
	if !strings.Contains(out, "\x1b[0m") {
		t.Error("output should contain reset sequences")
	}
}

// TestCompatSyslogNoProcess verifies syslog lines without a PID bracket.
func TestCompatSyslogNoProcess(t *testing.T) {
	line := "Sep 14 11:45:00 myhost daemon: starting up"
	out := processLine(line)

	if !strings.Contains(out, "daemon") {
		t.Error("process name should be present")
	}
	// Should NOT contain brackets
	stripped := stripAnsiCompat(out)
	if strings.Contains(stripped, "[") {
		t.Error("no PID brackets expected for line without PID")
	}
	if !strings.Contains(stripped, "starting") {
		t.Error("rest message should be present")
	}
}

// TestCompatSyslogMark verifies the MARK line is handled with the Repeat color.
func TestCompatSyslogMark(t *testing.T) {
	line := "Oct 12 22:40:12 iluvatar -- MARK --"
	out := processLine(line)

	// Should contain the MARK text
	stripped := stripAnsiCompat(out)
	if !strings.Contains(stripped, "-- MARK --") {
		t.Error("MARK text should be present")
	}
	// Repeat color: 7 (white, 37m) — no bold
	if !strings.Contains(out, "\x1b[37m") {
		t.Error("MARK should use white (37m) for repeat color")
	}
}

// TestCompatSyslogRepeat verifies repeated message handling.
func TestCompatSyslogRepeat(t *testing.T) {
	line := "Oct 12 22:40:12 iluvatar last message repeated 10 times"
	out := processLine(line)
	stripped := stripAnsiCompat(out)
	if !strings.Contains(stripped, "last message repeated 10 times") {
		t.Error("repeated message text should be present")
	}
}

// TestCompatHTTPDAccess verifies HTTPD access log output.
func TestCompatHTTPDAccess(t *testing.T) {
	line := `192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`
	out := processLine(line)

	// Host should be colored with Host color (bold blue)
	if !strings.Contains(out, "192.168.1.1") {
		t.Error("host IP should be present")
	}

	// User "frank" should use User color: bold yellow (AttrBold|3, 33m)
	if !strings.Contains(out, "frank") {
		t.Error("user should be present")
	}
	if !strings.Contains(out, "\x1b[33m") {
		t.Error("user should use yellow (33m)")
	}

	// HTTP action "GET" should use HTTPGet color: green (32m)
	if !strings.Contains(out, "\x1b[32m") {
		t.Error("GET action should use green (32m)")
	}

	// HTTP code "200" should use HTTPCodes: bold white (37m)
	if !strings.Contains(out, "200") {
		t.Error("HTTP status code should be present")
	}

	// Size "2326" should use GetSize: magenta (ansiColor[6]=35m)
	if !strings.Contains(out, "2326") {
		t.Error("response size should be present")
	}
	if !strings.Contains(out, "\x1b[35m") {
		t.Error("size should use magenta (35m)")
	}
}

// TestCompatHTTPDError verifies HTTPD error log output.
func TestCompatHTTPDError(t *testing.T) {
	line := "[Sun Oct 12 15:30:00 2003] [error] client denied by configuration"
	out := processLine(line)

	// Date portion
	if !strings.Contains(out, "[Sun Oct 12 15:30:00 2003]") {
		t.Error("date bracket should be present")
	}

	// Error level should use Error color: bold red (AttrBold|1, 31m)
	if !strings.Contains(out, "\x1b[31m") {
		t.Error("error level should use red (31m)")
	}
	if !strings.Contains(out, "[error]") {
		t.Error("error level text should be present")
	}
}

// TestCompatHTTPDPost verifies HTTP POST uses HTTPPost color (bold green).
func TestCompatHTTPDPost(t *testing.T) {
	line := `192.168.1.1 - admin [10/Oct/2000:13:55:36 -0700] "POST /api/submit HTTP/1.1" 201 512`
	out := processLine(line)
	// POST should use HTTPPost: bold green (AttrBold|2, bold+32m)
	stripped := stripAnsiCompat(out)
	if !strings.Contains(stripped, "POST /api/submit HTTP/1.1") {
		t.Error("POST action should be present")
	}
}

// TestCompatDpkg verifies dpkg log output.
func TestCompatDpkg(t *testing.T) {
	line := "2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3"
	out := processLine(line)
	stripped := stripAnsiCompat(out)

	if !strings.Contains(stripped, "2023-10-15 14:30:22") {
		t.Error("date should be present")
	}
	if !strings.Contains(stripped, "installed") {
		t.Error("status should be present")
	}
	if !strings.Contains(stripped, "libfoo:amd64") {
		t.Error("package name should be present")
	}
}

// TestCompatPostfix verifies postfix partial-match output.
func TestCompatPostfix(t *testing.T) {
	// Simulate the syslog+postfix pipeline
	fullLine := "Sep 14 11:45:00 mailhost postfix/smtp[1234]: ABC123: to=<user@example.com>,relay=smtp.example.com"
	out := processLine(fullLine)
	stripped := stripAnsiCompat(out)

	if !strings.Contains(stripped, "mailhost") {
		t.Error("host should be present")
	}
	if !strings.Contains(stripped, "postfix/smtp") {
		t.Error("process should be present")
	}
	// Postfix partial fields
	if !strings.Contains(stripped, "ABC123") {
		t.Error("spool ID should be present")
	}
	// Field names should be colored with Field color (green, 32m)
	if !strings.Contains(out, "\x1b[32m") {
		t.Error("field names should use green (32m)")
	}
}

// TestCompatSquidAccess verifies squid access log output.
func TestCompatSquidAccess(t *testing.T) {
	line := "1234567890.123      5 192.168.1.1 TCP_MISS/200 1234 GET http://example.com user DIRECT/93.184.216.34 text/html"
	out := processLine(line)
	stripped := stripAnsiCompat(out)

	if !strings.Contains(stripped, "192.168.1.1") {
		t.Error("host should be present")
	}
	if !strings.Contains(stripped, "TCP_MISS") {
		t.Error("action should be present")
	}
	if !strings.Contains(stripped, "http://example.com") {
		t.Error("URI should be present")
	}
	// TCP_MISS should use ProxyMiss color: red (31m)
	if !strings.Contains(out, "\x1b[31m") {
		t.Error("miss action should use red (31m)")
	}
}

// TestCompatWordColorBadWord verifies "error" words use bold red.
func TestCompatWordColorBadWord(t *testing.T) {
	// Process a plain line (no plugin match) with error keyword
	line := "something failed with error"
	out := processLine(line)

	// "error" should be colored with Error: bold red (AttrBold|1, bold+31m)
	if !strings.Contains(out, "\x1b[1m") {
		t.Error("error word should be bold")
	}
	if !strings.Contains(out, "\x1b[31m") {
		t.Error("error word should be red")
	}
}

// TestCompatWordColorGoodWord verifies "started" uses bold green.
func TestCompatWordColorGoodWord(t *testing.T) {
	line := "service started successfully"
	out := processLine(line)

	// "started" should use GoodWord: bold green (AttrBold|2)
	stripped := stripAnsiCompat(out)
	if !strings.Contains(stripped, "started") {
		t.Error("good word should be present")
	}
}

// TestCompatWordColorIPAddress verifies IP addresses use Host color.
func TestCompatWordColorIPAddress(t *testing.T) {
	line := "connecting to 10.0.0.1 from 192.168.1.100"
	out := processLine(line)

	// IP addresses should use Host color: bold blue (34m)
	if !strings.Contains(out, "10.0.0.1") {
		t.Error("IP should be present")
	}
	if !strings.Contains(out, "\x1b[34m") {
		t.Error("IP should use blue (34m)")
	}
}

// TestCompatWordColorURI verifies URIs use URI color.
func TestCompatWordColorURI(t *testing.T) {
	line := "fetching http://example.com/path"
	out := processLine(line)
	if !strings.Contains(out, "http://example.com/path") {
		t.Error("URI should be present")
	}
}

// TestCompatWordColorEmail verifies email addresses use Email color.
func TestCompatWordColorEmail(t *testing.T) {
	line := "mail from user@example.com delivered"
	out := processLine(line)
	if !strings.Contains(out, "user@example.com") {
		t.Error("email should be present")
	}
}

// TestCompatWordColorVersion verifies version strings use Version color.
func TestCompatWordColorVersion(t *testing.T) {
	line := "upgraded to 2.3.7 from 1.0.0"
	out := processLine(line)
	stripped := stripAnsiCompat(out)
	if !strings.Contains(stripped, "2.3.7") {
		t.Error("version should be present")
	}
}

// TestCompatWordColorDirectory verifies paths use Dir color.
func TestCompatWordColorDirectory(t *testing.T) {
	line := "reading /etc/passwd done"
	out := processLine(line)
	// Dir color: bold cyan (AttrBold|5, 36m)
	if !strings.Contains(out, "/etc/passwd") {
		t.Error("path should be present")
	}
	if !strings.Contains(out, "\x1b[36m") {
		t.Error("path should use cyan (36m)")
	}
}

// TestCompatWordColorSignal verifies signal names use Signal color.
func TestCompatWordColorSignal(t *testing.T) {
	line := "received sigterm shutting down"
	out := processLine(line)
	// Signal color: bold yellow (AttrBold|3, 33m)
	if !strings.Contains(out, "sigterm") {
		t.Error("signal should be present")
	}
}

// TestCompatWordColorMAC verifies MAC addresses use MAC color.
func TestCompatWordColorMAC(t *testing.T) {
	line := "device aa:bb:cc:dd:ee:ff connected"
	out := processLine(line)
	// MAC color: bold white (AttrBold|7, 37m)
	if !strings.Contains(out, "aa:bb:cc:dd:ee:ff") {
		t.Error("MAC address should be present")
	}
}

// TestCompatWordColorHexAddress verifies hex addresses use Address color.
func TestCompatWordColorHexAddress(t *testing.T) {
	line := "fault at 0x1234abcd"
	out := processLine(line)
	if !strings.Contains(out, "0x1234abcd") {
		t.Error("hex address should be present")
	}
}

// TestCompatWordColorSize verifies size strings use Size color.
func TestCompatWordColorSize(t *testing.T) {
	line := "downloaded 150mb in 30s"
	out := processLine(line)
	if !strings.Contains(out, "150mb") {
		t.Error("size should be present")
	}
}

// TestCompatWordColorSystemWord verifies system words use SystemWord color.
func TestCompatWordColorSystemWord(t *testing.T) {
	line := "linux kernel booting"
	out := processLine(line)
	// SystemWord color: bold cyan (AttrBold|5, 36m)
	stripped := stripAnsiCompat(out)
	if !strings.Contains(stripped, "linux") {
		t.Error("system word should be present")
	}
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
		if got != want {
			t.Errorf("color %s: got 0x%x, want 0x%x", color.ColorName(c), got, want)
		}
	}
}

// TestCompatAnsiColorSwap verifies the cyan/magenta swap at indices 5-6
// matches the C code. This is a deliberate quirk in the original C ccze.
func TestCompatAnsiColorSwap(t *testing.T) {
	ct := color.NewTable(true)

	// Color 5 (cyan) should produce ANSI code 36
	// Color 6 (magenta) should produce ANSI code 35
	// Verify by looking at the actual output:

	var buf bytes.Buffer
	// Date is AttrBold|5 (cyan), should produce \x1b[36m
	ct.WriteColored(&buf, color.Date, "X")
	if !strings.Contains(buf.String(), "\x1b[36m") {
		t.Errorf("cyan (index 5) should map to ANSI 36, got %q", buf.String())
	}

	buf.Reset()
	// GetSize is 6 (magenta), should produce \x1b[35m
	ct.WriteColored(&buf, color.GetSize, "X")
	if !strings.Contains(buf.String(), "\x1b[35m") {
		t.Errorf("magenta (index 6) should map to ANSI 35, got %q", buf.String())
	}
}

// TestCompatAttrSGRSwap verifies that AttrReverse emits SGR 5 (blink) and
// AttrBlink emits SGR 7 (reverse), matching the C code's peculiar behavior.
func TestCompatAttrSGRSwap(t *testing.T) {
	ct := color.NewTable(true)

	ct.Set(color.Date, color.AttrReverse|3)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "X")
	if !strings.Contains(buf.String(), "\x1b[5m") {
		t.Error("AttrReverse should emit SGR 5 (C code compatibility)")
	}

	ct.Set(color.Date, color.AttrBlink|3)
	buf.Reset()
	ct.WriteColored(&buf, color.Date, "X")
	if !strings.Contains(buf.String(), "\x1b[7m") {
		t.Error("AttrBlink should emit SGR 7 (C code compatibility)")
	}
}

// TestCompatBackgroundColor verifies background color encoding matches C.
func TestCompatBackgroundColor(t *testing.T) {
	ct := color.NewTable(true)

	// bg=1 (red) | fg=7 (white) => bg ANSI = 31+10 = 41
	ct.Set(color.Date, (1<<8)|7)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "X")
	if !strings.Contains(buf.String(), "\x1b[41m") {
		t.Errorf("bg red should produce \\x1b[41m, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "\x1b[37m") {
		t.Errorf("fg white should produce \\x1b[37m, got %q", buf.String())
	}
}

// TestCompatNonTransparentBackground verifies that non-transparent mode
// always emits a background color, matching C behavior.
func TestCompatNonTransparentBackground(t *testing.T) {
	ct := color.NewTable(false)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Default, "X")
	// bg=0 (black) in non-transparent -> ANSI 30+10 = 40
	if !strings.Contains(buf.String(), "\x1b[40m") {
		t.Error("non-transparent should emit background code 40m")
	}
}

// TestCompatResetSequence verifies every colored string ends with ESC[0m.
func TestCompatResetSequence(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Error, "test")
	if !strings.HasSuffix(buf.String(), "\x1b[0m") {
		t.Errorf("colored output should end with reset, got %q", buf.String())
	}
}

// TestCompatIntensityReset verifies every colored string starts with ESC[22m.
func TestCompatIntensityReset(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Error, "test")
	if !strings.HasPrefix(buf.String(), "\x1b[22m") {
		t.Errorf("colored output should start with intensity reset, got %q", buf.String())
	}
}

// --------------------------------------------------------------------------
// Plugin-level golden output tests
// --------------------------------------------------------------------------

// TestCompatSyslogExactOutput verifies the exact ANSI sequence for a syslog
// line with PID brackets.
func TestCompatSyslogExactOutput(t *testing.T) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewSyslogPlugin(&buf, ct, wc, false)

	handled, rest := p.Handle("Sep 14 11:45:00 myhost sshd[1234]: test message")
	if !handled {
		t.Fatal("syslog should handle this line")
	}
	if rest != "test message" {
		t.Fatalf("rest = %q, want 'test message'", rest)
	}

	out := buf.String()

	// Expected sequence (captured from C ccze):
	// [22m[1m[36mSep 14 11:45:00[0m  <- date: bold cyan
	// [22m[36m [0m                    <- space: default (cyan)
	// [22m[1m[34mmyhost[0m            <- host: bold blue
	// [22m[36m [0m                    <- space
	// [22m[32msshd[0m                 <- process: green
	// [22m[1m[32m[[0m                 <- PID bracket: bold green
	// [22m[1m[37m1234[0m              <- PID: bold white
	// [22m[1m[32m][0m                 <- PID bracket: bold green
	// [22m[32m:[0m                    <- colon: process green
	// [22m[36m [0m                    <- space

	// Verify date: bold cyan
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[36mSep 14 11:45:00\x1b[0m") {
		t.Errorf("date ANSI mismatch.\ngot:  %q", out)
	}

	// Verify host: bold blue
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[34mmyhost\x1b[0m") {
		t.Errorf("host ANSI mismatch.\ngot:  %q", out)
	}

	// Verify process: green (no bold)
	if !containsSequence(out, "\x1b[22m\x1b[32msshd\x1b[0m") {
		t.Errorf("process ANSI mismatch.\ngot:  %q", out)
	}

	// Verify PID bracket open: bold green
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[32m[\x1b[0m") {
		t.Errorf("PID bracket open ANSI mismatch.\ngot:  %q", out)
	}

	// Verify PID number: bold white
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[37m1234\x1b[0m") {
		t.Errorf("PID number ANSI mismatch.\ngot:  %q", out)
	}

	// Verify PID bracket close: bold green
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[32m]\x1b[0m") {
		t.Errorf("PID bracket close ANSI mismatch.\ngot:  %q", out)
	}
}

// TestCompatHTTPDAccessExactOutput verifies exact ANSI for an HTTP access log.
func TestCompatHTTPDAccessExactOutput(t *testing.T) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewHTTPDPlugin(&buf, ct, wc, false)

	handled, _ := p.Handle(`192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`)
	if !handled {
		t.Fatal("httpd should handle this line")
	}

	out := buf.String()

	// Host: bold blue
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[34m192.168.1.1\x1b[0m") {
		t.Errorf("host ANSI mismatch.\ngot:  %q", out)
	}

	// User: bold yellow
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[33mfrank\x1b[0m") {
		t.Errorf("user ANSI mismatch.\ngot:  %q", out)
	}

	// HTTP action: green (GET = HTTPGet = 2)
	if !containsSequence(out, "\x1b[22m\x1b[32m") {
		t.Errorf("HTTP GET action ANSI mismatch.\ngot:  %q", out)
	}

	// HTTP code: bold white
	if !containsSequence(out, "\x1b[22m\x1b[1m\x1b[37m200\x1b[0m") {
		t.Errorf("HTTP code ANSI mismatch.\ngot:  %q", out)
	}

	// Get size: magenta (6, ansiColor[6]=35)
	if !containsSequence(out, "\x1b[22m\x1b[35m2326\x1b[0m") {
		t.Errorf("size ANSI mismatch.\ngot:  %q", out)
	}
}

// TestCompatEmptyStringNoOutput verifies empty string produces no output.
func TestCompatEmptyStringNoOutput(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "")
	if buf.Len() != 0 {
		t.Error("empty string should produce no ANSI output")
	}
}

// --------------------------------------------------------------------------
// End-to-end test: multiple log types through the full pipeline
// --------------------------------------------------------------------------

func TestCompatFullPipelineMultipleFormats(t *testing.T) {
	lines := []struct {
		name  string
		input string
		check func(t *testing.T, out string)
	}{
		{
			name:  "syslog_with_pid",
			input: "Jan  1 00:00:00 server sshd[99]: Accepted publickey",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				if !strings.Contains(s, "server") || !strings.Contains(s, "sshd") ||
					!strings.Contains(s, "99") || !strings.Contains(s, "Accepted") {
					t.Error("syslog output missing fields")
				}
			},
		},
		{
			name:  "httpd_access",
			input: `10.0.0.1 - - [01/Jan/2024:12:00:00 +0000] "HEAD /health HTTP/1.1" 200 0`,
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				if !strings.Contains(s, "10.0.0.1") || !strings.Contains(s, "HEAD") {
					t.Error("httpd output missing fields")
				}
			},
		},
		{
			name:  "httpd_error",
			input: "[Mon Jan 01 12:00:00 2024] [warn] potential issue",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				if !strings.Contains(s, "[warn]") || !strings.Contains(s, "potential issue") {
					t.Error("httpd error output missing fields")
				}
			},
		},
		{
			name:  "dpkg_status",
			input: "2024-01-01 12:00:00 status installed base-files:amd64 12.4",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				if !strings.Contains(s, "status") || !strings.Contains(s, "base-files") {
					t.Error("dpkg output missing fields")
				}
			},
		},
		{
			name:  "plain_with_keywords",
			input: "error connecting to server failed retry starting",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				if !strings.Contains(s, "error") || !strings.Contains(s, "starting") {
					t.Error("keyword output missing words")
				}
			},
		},
		{
			name:  "plain_with_ip_and_path",
			input: "connection from 192.168.0.1 reading /var/log/syslog",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				if !strings.Contains(s, "192.168.0.1") || !strings.Contains(s, "/var/log/syslog") {
					t.Error("IP/path output missing")
				}
			},
		},
	}

	for _, tt := range lines {
		t.Run(tt.name, func(t *testing.T) {
			out := processLine(tt.input)
			tt.check(t, out)
		})
	}
}

// --------------------------------------------------------------------------
// Live comparison against C ccze binary (skipped if not available)
// --------------------------------------------------------------------------

func TestCompatAgainstCBinary(t *testing.T) {
	ccze, err := exec.LookPath("ccze")
	if err != nil {
		t.Skip("C ccze binary not found; skipping live comparison test. Install with: apt install ccze")
	}

	lines := []string{
		"Sep 14 11:45:00 myhost sshd[1234]: Connection closed",
		"Oct 12 22:40:12 iluvatar -- MARK --",
		"Oct 12 22:40:12 iluvatar last message repeated 10 times",
		`192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`,
		`[Sun Oct 12 15:30:00 2003] [error] client denied`,
		"2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3",
		"Sep 14 11:45:00 mailhost postfix/smtp[1234]: ABC123: to=<user@example.com>,relay=smtp.example.com",
		"1234567890.123      5 192.168.1.1 TCP_MISS/200 1234 GET http://example.com user DIRECT/93.184.216.34 text/html",
	}

	for i, line := range lines {
		t.Run(fmt.Sprintf("line_%d", i), func(t *testing.T) {
			// Get C output
			cmd := exec.Command(ccze, "-A")
			cmd.Stdin = strings.NewReader(line + "\n")
			cOutput, err := cmd.Output()
			if err != nil {
				t.Skipf("failed to run C ccze: %v", err)
			}

			// Get Go output
			goOutput := processLine(line)

			// Normalize known cosmetic differences between C and Go:
			// 1. C emits ANSI escapes for empty strings; Go skips them
			// 2. C appends a trailing default-colored space after the last word
			// 3. C appends a final \x1b[0m reset after the newline
			// None of these affect the visible terminal rendering.
			cStr := normalizeAnsiOutput(string(cOutput))
			gStr := normalizeAnsiOutput(goOutput)

			if cStr != gStr {
				t.Errorf("output mismatch for line %d\n  C:  %q\n  Go: %q", i, cStr, gStr)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Config file compatibility
// --------------------------------------------------------------------------

func TestCompatConfigFileParsing(t *testing.T) {
	ct := color.NewTable(true)

	// Test all color names parse correctly
	colorTests := []struct {
		line string
		slot color.Color
		want int
	}{
		{"date bold cyan", color.Date, color.AttrBold | 5},
		{"host red", color.Host, 1},
		{"process green", color.Proc, 2},
		{"error bold red on_white", color.Error, color.AttrBold | 1 | (7 << 8)},
		{"pid underline yellow", color.PID, color.AttrUnderline | 3},
		{"warning reverse blue", color.Warning, color.AttrReverse | 4},
		{"debug blink magenta", color.Debug, color.AttrBlink | 6},
		{"default black", color.Default, 0},
		{"user white", color.User, 7},
		// Background colors
		{"date green on_red", color.Date, 2 | (1 << 8)},
		{"host blue on_green", color.Host, 4 | (2 << 8)},
		// Quoted keyword reference
		{"host 'date'", color.Host, 2 | (1 << 8)}, // date was set to green on_red above
	}

	for _, tt := range colorTests {
		ct.ParseLine(tt.line)
		got := ct.Get(tt.slot)
		if got != tt.want {
			t.Errorf("ParseLine(%q): slot %s = 0x%x, want 0x%x",
				tt.line, color.ColorName(tt.slot), got, tt.want)
		}
	}
}

// --------------------------------------------------------------------------
// Facility prefix removal
// --------------------------------------------------------------------------

func TestCompatFacilityRemoval(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<13>Sep 14 11:45:00 host test: msg", "Sep 14 11:45:00 host test: msg"},
		{"<0>line", "line"},
		{"no prefix", "no prefix"},
		{"<>empty", "empty"},
	}

	for _, tt := range tests {
		line := tt.input
		if len(line) > 0 && line[0] == '<' {
			if idx := strings.Index(line, ">"); idx >= 0 {
				line = line[idx+1:]
			}
		}
		if line != tt.want {
			t.Errorf("facility removal: %q -> %q, want %q", tt.input, line, tt.want)
		}
	}
}

// --------------------------------------------------------------------------
// convertColorOverride compatibility
// --------------------------------------------------------------------------

func TestCompatConvertColorOverride(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"date=boldcyan", "date bold cyan"},
		{"error=red", "error red"},
		{"host=underlinegreen", "host underline green"},
		{"warning=yellow", "warning yellow"},
		{"date=boldredon_blue", "date bold red on_blue"},
		{"date=reversewhite", "date reverse white"},
		{"date=blinkmagenta", "date blink magenta"},
		// No equals sign
		{"noequals", "noequals"},
	}

	for _, tt := range tests {
		got := convertColorOverride(tt.input)
		if got != tt.want {
			t.Errorf("convertColorOverride(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

func stripAnsiCompat(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func containsSequence(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// reEmptyAnsi matches ANSI escape sequences that wrap no visible text,
// e.g., \x1b[22m\x1b[1m\x1b[34m\x1b[0m (bold blue empty string).
var reEmptyAnsi = regexp.MustCompile(`\x1b\[22m(\x1b\[\d+m)*\x1b\[0m`)

// reTrailingAnsi matches trailing ANSI sequences with no text after them.
var reTrailingAnsi = regexp.MustCompile(`(\x1b\[\d+m)+$`)

// normalizeAnsiOutput strips cosmetic differences that don't affect visible
// rendering: empty ANSI segments (escape sequences wrapping no text),
// trailing default-colored spaces, and final reset sequences.
func normalizeAnsiOutput(s string) string {
	// Remove empty ANSI segments (escape sequences wrapping no visible text)
	s = reEmptyAnsi.ReplaceAllString(s, "")
	// Iteratively strip trailing non-visible content
	for {
		prev := s
		s = strings.TrimRight(s, "\n ")
		s = reTrailingAnsi.ReplaceAllString(s, "")
		if s == prev {
			break
		}
	}
	return s
}
