package wordcolor

import (
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
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
	if buf.Len() != 0 {
		t.Error("Process empty should produce no output")
	}
}

func TestProcessNoWordcolor(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "hello world", false, false)
	out := stripAnsi(buf.String())
	if out != "hello world" {
		t.Errorf("Process with wcol=false should output raw text, got %q", out)
	}
}

func TestProcessRepeatMessage(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "last message repeated 10 times", true, false)
	out := buf.String()
	if !strings.Contains(out, "last message repeated 10 times") {
		t.Error("repeat message should appear in output")
	}
}

func TestProcessMark(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "-- MARK --", true, false)
	out := buf.String()
	if !strings.Contains(out, "-- MARK --") {
		t.Error("MARK message should appear in output")
	}
}

func TestProcessOneIPAddress(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "192.168.1.1", false)
	out := stripAnsi(buf.String())
	if out != "192.168.1.1" {
		t.Errorf("IP address should appear, got %q", out)
	}
}

func TestProcessOneHostname(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "www.example.com", false)
	out := stripAnsi(buf.String())
	if out != "www.example.com" {
		t.Errorf("hostname should appear, got %q", out)
	}
}

func TestProcessOneMAC(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "aa:bb:cc:dd:ee:ff", false)
	out := stripAnsi(buf.String())
	if out != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("MAC should appear, got %q", out)
	}
}

func TestProcessOneDirectory(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "/etc/passwd", false)
	out := stripAnsi(buf.String())
	if out != "/etc/passwd" {
		t.Errorf("directory should appear, got %q", out)
	}
}

func TestProcessOneEmail(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "user@example.com", false)
	out := stripAnsi(buf.String())
	if out != "user@example.com" {
		t.Errorf("email should appear, got %q", out)
	}
}

func TestProcessOneURI(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "http://example.com/path", false)
	out := stripAnsi(buf.String())
	if out != "http://example.com/path" {
		t.Errorf("URI should appear, got %q", out)
	}
}

func TestProcessOneVersion(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "2.3.7", false)
	out := stripAnsi(buf.String())
	if out != "2.3.7" {
		t.Errorf("version should appear, got %q", out)
	}
}

func TestProcessOneNumber(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "42", false)
	out := stripAnsi(buf.String())
	if out != "42" {
		t.Errorf("number should appear, got %q", out)
	}
}

func TestProcessOneAddress(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "0x1234abcd", false)
	out := stripAnsi(buf.String())
	if out != "0x1234abcd" {
		t.Errorf("address should appear, got %q", out)
	}
}

func TestProcessOneSignal(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "sigterm", false)
	out := stripAnsi(buf.String())
	if out != "sigterm" {
		t.Errorf("signal should appear, got %q", out)
	}
}

func TestProcessOneSize(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "150mb", false)
	out := stripAnsi(buf.String())
	if out != "150mb" {
		t.Errorf("size should appear, got %q", out)
	}
}

func TestProcessOneTime(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "12:30:45", false)
	out := stripAnsi(buf.String())
	if out != "12:30:45" {
		t.Errorf("time should appear, got %q", out)
	}
}

func TestProcessOneBadWord(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "error", false)
	// Should use Error color (bold red)
	out := buf.String()
	if !strings.Contains(out, "\x1b[1m") {
		t.Error("error word should be bold")
	}
}

func TestProcessOneGoodWord(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "started", false)
	out := stripAnsi(buf.String())
	if out != "started" {
		t.Errorf("good word should appear, got %q", out)
	}
}

func TestProcessOneSystemWord(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "linux", false)
	out := stripAnsi(buf.String())
	if out != "linux" {
		t.Errorf("system word should appear, got %q", out)
	}
}

func TestProcessOnePunctuation(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "(test)", false)
	out := stripAnsi(buf.String())
	if out != "(test)" {
		t.Errorf("punctuation-wrapped word should appear, got %q", out)
	}
}

func TestProcessOneHostIP(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "mail.example.com[192.168.1.1]", false)
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "mail.example.com") {
		t.Errorf("host part should appear, got %q", out)
	}
	if !strings.Contains(out, "192.168.1.1") {
		t.Errorf("IP part should appear, got %q", out)
	}
}

func TestProcessMultipleWords(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "hello world 42", true, false)
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") || !strings.Contains(out, "42") {
		t.Errorf("all words should appear, got %q", out)
	}
}

func TestProcessLocalhost(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.ProcessOne(&buf, "localhost", false)
	out := stripAnsi(buf.String())
	if out != "localhost" {
		t.Errorf("localhost should appear, got %q", out)
	}
}

func TestProcessConsecutiveSpaces(t *testing.T) {
	p, _ := newTestProcessor()
	var buf bytes.Buffer
	p.Process(&buf, "a  b", true, false)
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "a") && !strings.Contains(out, "b") {
		t.Errorf("both words should appear, got %q", out)
	}
}
