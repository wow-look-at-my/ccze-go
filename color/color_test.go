package color

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColorConstants(t *testing.T) {
	if Date != 0 {
		t.Errorf("Date should be 0, got %d", Date)
	}
	if Last != StaticBoldWhite+1 {
		t.Errorf("Last should follow StaticBoldWhite, got %d", Last)
	}
}

func TestColorName(t *testing.T) {
	tests := []struct {
		c    Color
		want string
	}{
		{Date, "date"},
		{Host, "host"},
		{Proc, "process"},
		{PID, "pid"},
		{PIDB, "pid-sqbr"},
		{Default, "default"},
		{Error, "error"},
		{BadWord, "bad"},
		{GoodWord, "good"},
		{PkgStatus, "pkgstatus"},
		{Pkg, "pkg"},
	}
	for _, tt := range tests {
		if got := ColorName(tt.c); got != tt.want {
			t.Errorf("ColorName(%d) = %q, want %q", tt.c, got, tt.want)
		}
	}
	if got := ColorName(Last); got != "" {
		t.Errorf("ColorName(Last) should be empty, got %q", got)
	}
}

func TestKeywordLookup(t *testing.T) {
	c, ok := KeywordLookup("date")
	if !ok || c != Date {
		t.Errorf("KeywordLookup(date) = %d, %v; want Date, true", c, ok)
	}
	c, ok = KeywordLookup("process")
	if !ok || c != Proc {
		t.Errorf("KeywordLookup(process) = %d, %v; want Proc, true", c, ok)
	}
	// Hidden keywords (static colors)
	c, ok = KeywordLookup("bold_red")
	if !ok || c != StaticBoldRed {
		t.Errorf("KeywordLookup(bold_red) = %d, %v; want StaticBoldRed, true", c, ok)
	}
	// Unknown keyword
	_, ok = KeywordLookup("nonexistent")
	if ok {
		t.Error("KeywordLookup(nonexistent) should return false")
	}
}

func TestNewTable(t *testing.T) {
	ct := NewTable(true)
	if ct == nil {
		t.Fatal("NewTable returned nil")
	}
	if !ct.Transparent() {
		t.Error("table should be transparent")
	}
	// Check some default values
	if ct.Get(Date) != AttrBold|5 {
		t.Errorf("Date color = %x, want %x", ct.Get(Date), AttrBold|5)
	}
	if ct.Get(Proc) != 2 {
		t.Errorf("Proc color = %x, want 2", ct.Get(Proc))
	}
	if ct.Get(Error) != AttrBold|1 {
		t.Errorf("Error color = %x, want %x", ct.Get(Error), AttrBold|1)
	}
	if ct.Get(File) != ct.Get(Dir) {
		t.Error("File should equal Dir")
	}
}

func TestNewTableNotTransparent(t *testing.T) {
	ct := NewTable(false)
	if ct.Transparent() {
		t.Error("table should not be transparent")
	}
}

func TestSetGet(t *testing.T) {
	ct := NewTable(true)
	ct.Set(Date, 42)
	if ct.Get(Date) != 42 {
		t.Errorf("Get(Date) = %d after Set(42), want 42", ct.Get(Date))
	}
}

func TestWriteColored(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "hello")
	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Errorf("output should contain 'hello', got %q", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Error("output should contain ESC sequences")
	}
	// Bold should be present for Date (AttrBold|5)
	if !strings.Contains(out, "\x1b[1m") {
		t.Error("output should contain bold ESC for Date")
	}
	if !strings.HasSuffix(out, "\x1b[0m") {
		t.Error("output should end with reset")
	}
}

func TestWriteColoredEmpty(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "")
	if buf.Len() != 0 {
		t.Error("WriteColored with empty string should produce no output")
	}
}

func TestWriteColoredNonTransparent(t *testing.T) {
	ct := NewTable(false)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Default, "test")
	out := buf.String()
	// Non-transparent should always emit background code
	if !strings.Contains(out, "\x1b[40m") {
		t.Errorf("non-transparent output should contain bg code, got %q", out)
	}
}

func TestWriteColoredAttributes(t *testing.T) {
	ct := NewTable(true)
	// Set a color with underline
	ct.Set(Date, AttrUnderline|2)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "x")
	out := buf.String()
	if !strings.Contains(out, "\x1b[4m") {
		t.Error("should contain underline escape")
	}

	// Set with reverse (emits SGR 5)
	ct.Set(Date, AttrReverse|3)
	buf.Reset()
	ct.WriteColored(&buf, Date, "x")
	out = buf.String()
	if !strings.Contains(out, "\x1b[5m") {
		t.Error("AttrReverse should emit SGR 5 (blink)")
	}

	// Set with blink (emits SGR 7)
	ct.Set(Date, AttrBlink|4)
	buf.Reset()
	ct.WriteColored(&buf, Date, "x")
	out = buf.String()
	if !strings.Contains(out, "\x1b[7m") {
		t.Error("AttrBlink should emit SGR 7 (reverse)")
	}
}

func TestWriteColoredBackground(t *testing.T) {
	ct := NewTable(true)
	// Set color with bg=1 (red), fg=7 (white)
	ct.Set(Date, (1<<8)|7)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "test")
	out := buf.String()
	// Should contain background code (ansiColor[1]+10 = 31+10 = 41)
	if !strings.Contains(out, "\x1b[41m") {
		t.Errorf("should contain bg code \\x1b[41m, got %q", out)
	}
}

func TestWriteSpace(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteSpace(&buf)
	if !strings.Contains(buf.String(), " ") {
		t.Error("WriteSpace should produce a space")
	}
}

func TestWriteNewline(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteNewline(&buf)
	if buf.String() != "\n" {
		t.Errorf("WriteNewline should produce newline, got %q", buf.String())
	}
}

func TestParseLine(t *testing.T) {
	ct := NewTable(true)
	// Simple color
	ct.ParseLine("date red")
	if ct.Get(Date) != 1 {
		t.Errorf("after 'date red', Date = %d, want 1", ct.Get(Date))
	}

	// Bold + color
	ct.ParseLine("host bold cyan")
	if ct.Get(Host) != AttrBold|5 {
		t.Errorf("after 'host bold cyan', Host = %x, want %x", ct.Get(Host), AttrBold|5)
	}

	// With background
	ct.ParseLine("error bold red on_white")
	expected := AttrBold | 1 | (7 << 8)
	if ct.Get(Error) != expected {
		t.Errorf("after 'error bold red on_white', Error = %x, want %x", ct.Get(Error), expected)
	}

	// CSS keywords should be skipped
	ct.ParseLine("cssbody #000000")
	// Should not crash

	// Empty line
	ct.ParseLine("")
	// Should not crash

	// Unknown keyword
	ct.ParseLine("nonexistent red")
	// Should not crash

	// Comment
	ct.ParseLine("date green # this is a comment")
	if ct.Get(Date) != 2 {
		t.Errorf("after 'date green', Date = %d, want 2", ct.Get(Date))
	}

	// Keyword with = separator
	ct.ParseLine("date=blue")
	if ct.Get(Date) != 4 {
		t.Errorf("after 'date=blue', Date = %d, want 4", ct.Get(Date))
	}
}

func TestParseLineQuotedKeyword(t *testing.T) {
	ct := NewTable(true)
	ct.Set(Date, AttrBold|5)
	ct.ParseLine("host 'date'")
	if ct.Get(Host) != ct.Get(Date) {
		t.Errorf("after 'host 'date'', Host = %x, want %x", ct.Get(Host), ct.Get(Date))
	}
}

func TestParseLineAttributes(t *testing.T) {
	ct := NewTable(true)
	ct.ParseLine("date underline green")
	if ct.Get(Date) != AttrUnderline|2 {
		t.Errorf("after underline, Date = %x, want %x", ct.Get(Date), AttrUnderline|2)
	}
	ct.ParseLine("date reverse yellow")
	if ct.Get(Date) != AttrReverse|3 {
		t.Errorf("after reverse, Date = %x, want %x", ct.Get(Date), AttrReverse|3)
	}
	ct.ParseLine("date blink blue")
	if ct.Get(Date) != AttrBlink|4 {
		t.Errorf("after blink, Date = %x, want %x", ct.Get(Date), AttrBlink|4)
	}
}

func TestLoadFile(t *testing.T) {
	ct := NewTable(true)
	// Non-existent file should not error
	err := ct.LoadFile("/nonexistent/path/to/file")
	if err != nil {
		t.Errorf("LoadFile on nonexistent should return nil, got %v", err)
	}

	// Create temp config file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	os.WriteFile(path, []byte("date bold red\nhost green\n"), 0644)

	err = ct.LoadFile(path)
	if err != nil {
		t.Errorf("LoadFile returned error: %v", err)
	}
	if ct.Get(Date) != AttrBold|1 {
		t.Errorf("after loading config, Date = %x, want %x", ct.Get(Date), AttrBold|1)
	}
	if ct.Get(Host) != 2 {
		t.Errorf("after loading config, Host = %d, want 2", ct.Get(Host))
	}
}

func TestLoadFileDirectory(t *testing.T) {
	ct := NewTable(true)
	// Loading a directory should silently succeed (not regular file)
	err := ct.LoadFile(t.TempDir())
	if err != nil {
		t.Errorf("LoadFile on directory should return nil, got %v", err)
	}
}

func TestAnsiColorSwap(t *testing.T) {
	// Verify cyan/magenta swap
	if ansiColor[5] != 36 {
		t.Errorf("ansiColor[5] (cyan) = %d, want 36", ansiColor[5])
	}
	if ansiColor[6] != 35 {
		t.Errorf("ansiColor[6] (magenta) = %d, want 35", ansiColor[6])
	}
}
