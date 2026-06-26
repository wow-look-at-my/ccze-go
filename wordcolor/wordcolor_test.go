package wordcolor

import (
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
	"github.com/stretchr/testify/assert"
)

func newTestProcessor() (*Processor, *color.Table) {
	ct := color.NewTable(true)
	return New(ct), ct
}

// stripAnsi removes ANSI escape sequences from a string for easier testing.
func stripAnsi(s string) string {
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

func TestProcessEmpty(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "", true, false)
	assert.Equal(t, 0, buf.Len())

}

func TestProcessNoWordcolor(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "hello world", false, false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "hello world", out)

}

func TestProcessRepeatMessage(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "last message repeated 10 times", true, false)
	out := buf.String()
	assert.Contains(t, out, "last message repeated 10 times")

}

func TestProcessMark(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "-- MARK --", true, false)
	out := buf.String()
	assert.Contains(t, out, "-- MARK --")

}

func TestProcessOneIPAddress(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "192.168.1.1", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "192.168.1.1", out)

}

func TestProcessOneHostname(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "www.example.com", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "www.example.com", out)

}

func TestProcessOneMAC(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "aa:bb:cc:dd:ee:ff", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", out)

}

func TestProcessOneDirectory(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "/etc/passwd", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "/etc/passwd", out)

}

func TestProcessOneEmail(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "user@example.com", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "user@example.com", out)

}

func TestProcessOneURI(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "http://example.com/path", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "http://example.com/path", out)

}

func TestProcessOneVersion(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "2.3.7", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "2.3.7", out)

}

func TestProcessOneNumber(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "42", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "42", out)

}

func TestProcessOneAddress(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "0x1234abcd", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "0x1234abcd", out)

}

func TestProcessOneSignal(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "sigterm", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "sigterm", out)

}

func TestProcessOneSize(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "150mb", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "150mb", out)

}

func TestProcessOneTime(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "12:30:45", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "12:30:45", out)

}

func TestProcessOneBadWord(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "error", false)
	// Should use Error color (bold red)
	out := buf.String()
	assert.Contains(t, out, "\x1b[1m")

}

func TestProcessOneGoodWord(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "started", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "started", out)

}

func TestProcessOneSystemWord(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "linux", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "linux", out)

}

func TestProcessOnePunctuation(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "(test)", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "(test)", out)

}

func TestProcessOneHostIP(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "mail.example.com[192.168.1.1]", false)
	out := stripAnsi(buf.String())
	assert.Contains(t, out, "mail.example.com")

	assert.Contains(t, out, "192.168.1.1")

}

func TestProcessMultipleWords(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "hello world 42", true, false)
	out := stripAnsi(buf.String())
	assert.False(t, !strings.Contains(out, "hello") || !strings.Contains(out, "world") || !strings.Contains(out, "42"))

}

func TestProcessLocalhost(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "localhost", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "localhost", out)

}

func TestProcessConsecutiveSpaces(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "a  b", true, false)
	out := stripAnsi(buf.String())
	assert.False(t, !strings.Contains(out, "a") && !strings.Contains(out, "b"))

}
