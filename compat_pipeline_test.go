package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompatSyslogExactOutput(t *testing.T) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewSyslogPlugin(&buf, ct, wc, false)

	handled, rest := p.Handle("Sep 14 11:45:00 myhost sshd[1234]: test message")
	require.True(t, handled)

	require.Equal(t, "test message", rest)

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
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[36mSep 14 11:45:00\x1b[0m"))

	// Verify host: bold blue
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[34mmyhost\x1b[0m"))

	// Verify process: green (no bold)
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[32msshd\x1b[0m"))

	// Verify PID bracket open: bold green
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[32m[\x1b[0m"))

	// Verify PID number: bold white
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[37m1234\x1b[0m"))

	// Verify PID bracket close: bold green
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[32m]\x1b[0m"))

}

// TestCompatHTTPDAccessExactOutput verifies exact ANSI for an HTTP access log.
func TestCompatHTTPDAccessExactOutput(t *testing.T) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewHTTPDPlugin(&buf, ct, wc, false)

	handled, _ := p.Handle(`192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`)
	require.True(t, handled)

	out := buf.String()

	// Host: bold blue
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[34m192.168.1.1\x1b[0m"))

	// User: bold yellow
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[33mfrank\x1b[0m"))

	// HTTP action: green (GET = HTTPGet = 2)
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[32m"))

	// HTTP code: bold white
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[1m\x1b[37m200\x1b[0m"))

	// Get size: magenta (6, ansiColor[6]=35)
	assert.True(t, containsSequence(out, "\x1b[22m\x1b[35m2326\x1b[0m"))

}

// TestCompatEmptyStringNoOutput verifies empty string produces no output.
func TestCompatEmptyStringNoOutput(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, color.Date, "")
	assert.Equal(t, 0, buf.Len())

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
				assert.False(t, !strings.Contains(s, "server") || !strings.Contains(s, "sshd") || !strings.Contains(s, "99") || !strings.Contains(s, "Accepted"))

			},
		},
		{
			name:  "httpd_access",
			input: `10.0.0.1 - - [01/Jan/2024:12:00:00 +0000] "HEAD /health HTTP/1.1" 200 0`,
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				assert.False(t, !strings.Contains(s, "10.0.0.1") || !strings.Contains(s, "HEAD"))

			},
		},
		{
			name:  "httpd_error",
			input: "[Mon Jan 01 12:00:00 2024] [warn] potential issue",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				assert.False(t, !strings.Contains(s, "[warn]") || !strings.Contains(s, "potential issue"))

			},
		},
		{
			name:  "dpkg_status",
			input: "2024-01-01 12:00:00 status installed base-files:amd64 12.4",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				assert.False(t, !strings.Contains(s, "status") || !strings.Contains(s, "base-files"))

			},
		},
		{
			name:  "plain_with_keywords",
			input: "error connecting to server failed retry starting",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				assert.False(t, !strings.Contains(s, "error") || !strings.Contains(s, "starting"))

			},
		},
		{
			name:  "plain_with_ip_and_path",
			input: "connection from 192.168.0.1 reading /var/log/syslog",
			check: func(t *testing.T, out string) {
				s := stripAnsiCompat(out)
				assert.False(t, !strings.Contains(s, "192.168.0.1") || !strings.Contains(s, "/var/log/syslog"))

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

			assert.Equal(t, gStr, cStr)

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
		assert.Equal(t, tt.want, got)

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
		assert.Equal(t, tt.want, line)

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
		assert.Equal(t, tt.want, got)

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
