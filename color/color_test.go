package color

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/wow-look-at-my/testify/assert"
	"github.com/wow-look-at-my/testify/require"
)

func TestColorConstants(t *testing.T) {
	assert.Equal(t, 0, Date)

	assert.Equal(t, StaticBoldWhite+1, Last)

}

func TestColorName(t *testing.T) {
	tests := []struct {
		c	Color
		want	string
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
		got := ColorName(tt.c)
		assert.Equal(t, tt.want, got)

	}
	got := ColorName(Last)
	assert.Equal(t, "", got)

}

func TestKeywordLookup(t *testing.T) {
	c, ok := KeywordLookup("date")
	assert.False(t, !ok || c != Date)

	c, ok = KeywordLookup("process")
	assert.False(t, !ok || c != Proc)

	// Hidden keywords (static colors)
	c, ok = KeywordLookup("bold_red")
	assert.False(t, !ok || c != StaticBoldRed)

	// Unknown keyword
	_, ok = KeywordLookup("nonexistent")
	assert.False(t, ok)

}

func TestNewTable(t *testing.T) {
	ct := NewTable(true)
	require.NotNil(t, ct)

	assert.True(t, ct.Transparent())

	// Check some default values
	assert.Equal(t, AttrBold|5, ct.Get(Date))

	assert.Equal(t, 2, ct.Get(Proc))

	assert.Equal(t, AttrBold|1, ct.Get(Error))

	assert.Equal(t, ct.Get(Dir), ct.Get(File))

}

func TestNewTableNotTransparent(t *testing.T) {
	ct := NewTable(false)
	assert.False(t, ct.Transparent())

}

func TestSetGet(t *testing.T) {
	ct := NewTable(true)
	ct.Set(Date, 42)
	assert.Equal(t, 42, ct.Get(Date))

}

func TestWriteColored(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "hello")
	out := buf.String()
	assert.Contains(t, out, "hello")

	assert.Contains(t, out, "\x1b[")

	// Bold should be present for Date (AttrBold|5)
	assert.Contains(t, out, "\x1b[1m")

	assert.True(t, strings.HasSuffix(out, "\x1b[0m"))

}

func TestWriteColoredEmpty(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "")
	assert.Equal(t, 0, buf.Len())

}

func TestWriteColoredNonTransparent(t *testing.T) {
	ct := NewTable(false)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Default, "test")
	out := buf.String()
	// Non-transparent should always emit background code
	assert.Contains(t, out, "\x1b[40m")

}

func TestWriteColoredAttributes(t *testing.T) {
	ct := NewTable(true)
	// Set a color with underline
	ct.Set(Date, AttrUnderline|2)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "x")
	out := buf.String()
	assert.Contains(t, out, "\x1b[4m")

	// Set with reverse (emits SGR 5)
	ct.Set(Date, AttrReverse|3)
	buf.Reset()
	ct.WriteColored(&buf, Date, "x")
	out = buf.String()
	assert.Contains(t, out, "\x1b[5m")

	// Set with blink (emits SGR 7)
	ct.Set(Date, AttrBlink|4)
	buf.Reset()
	ct.WriteColored(&buf, Date, "x")
	out = buf.String()
	assert.Contains(t, out, "\x1b[7m")

}

func TestWriteColoredBackground(t *testing.T) {
	ct := NewTable(true)
	// Set color with bg=1 (red), fg=7 (white)
	ct.Set(Date, (1<<8)|7)
	var buf bytes.Buffer
	ct.WriteColored(&buf, Date, "test")
	out := buf.String()
	// Should contain background code (ansiColor[1]+10 = 31+10 = 41)
	assert.Contains(t, out, "\x1b[41m")

}

func TestWriteSpace(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteSpace(&buf)
	assert.Contains(t, buf.String(), " ")

}

func TestWriteNewline(t *testing.T) {
	ct := NewTable(true)
	var buf bytes.Buffer
	ct.WriteNewline(&buf)
	assert.Equal(t, "\n", buf.String())

}

func TestParseLine(t *testing.T) {
	ct := NewTable(true)
	// Simple color
	ct.ParseLine("date red")
	assert.Equal(t, 1, ct.Get(Date))

	// Bold + color
	ct.ParseLine("host bold cyan")
	assert.Equal(t, AttrBold|5, ct.Get(Host))

	// With background
	ct.ParseLine("error bold red on_white")
	expected := AttrBold | 1 | (7 << 8)
	assert.Equal(t, expected, ct.Get(Error))

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
	assert.Equal(t, 2, ct.Get(Date))

	// Keyword with = separator
	ct.ParseLine("date=blue")
	assert.Equal(t, 4, ct.Get(Date))

}

func TestParseLineQuotedKeyword(t *testing.T) {
	ct := NewTable(true)
	ct.Set(Date, AttrBold|5)
	ct.ParseLine("host 'date'")
	assert.Equal(t, ct.Get(Date), ct.Get(Host))

}

func TestParseLineAttributes(t *testing.T) {
	ct := NewTable(true)
	ct.ParseLine("date underline green")
	assert.Equal(t, AttrUnderline|2, ct.Get(Date))

	ct.ParseLine("date reverse yellow")
	assert.Equal(t, AttrReverse|3, ct.Get(Date))

	ct.ParseLine("date blink blue")
	assert.Equal(t, AttrBlink|4, ct.Get(Date))

}

func TestLoadFile(t *testing.T) {
	ct := NewTable(true)
	// Non-existent file should not error
	err := ct.LoadFile("/nonexistent/path/to/file")
	assert.Nil(t, err)

	// Create temp config file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")
	os.WriteFile(path, []byte("date bold red\nhost green\n"), 0644)

	err = ct.LoadFile(path)
	assert.Nil(t, err)

	assert.Equal(t, AttrBold|1, ct.Get(Date))

	assert.Equal(t, 2, ct.Get(Host))

}

func TestLoadFileDirectory(t *testing.T) {
	ct := NewTable(true)
	// Loading a directory should silently succeed (not regular file)
	err := ct.LoadFile(t.TempDir())
	assert.Nil(t, err)

}

func TestAnsiColorSwap(t *testing.T) {
	// Verify cyan/magenta swap
	assert.Equal(t, 36, ansiColor[5])

	assert.Equal(t, 35, ansiColor[6])

}
