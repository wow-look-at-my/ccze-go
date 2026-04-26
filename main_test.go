package main

import (
	"bufio"
	"bytes"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
	"github.com/wow-look-at-my/testify/assert"
)

func TestConvertColorOverride(t *testing.T) {
	tests := []struct {
		input	string
		want	string
	}{
		{"date=boldcyan", "date bold cyan"},
		{"error=red", "error red"},
		{"host=underlinegreen", "host underline green"},
		{"warning=yellow", "warning yellow"},
		{"noequals", "noequals"},
		{"date=boldredon_blue", "date bold red on_blue"},
		{"date=reversewhite", "date reverse white"},
		{"date=blinkmagenta", "date blink magenta"},
	}
	for _, tt := range tests {
		got := convertColorOverride(tt.input)
		assert.Equal(t, tt.want, got)

	}
}

func TestRegisterAllPlugins(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	registerAllPlugins(r, w, ct, wc, false)

	plugins := r.Plugins()
	assert.Equal(t, 20, len(plugins))

}

func TestFilterPlugins(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	registerAllPlugins(r, w, ct, wc, false)
	filterPlugins(r, []string{"syslog", "httpd"})

	plugins := r.Plugins()
	assert.Equal(t, 2, len(plugins))

}

func TestListAllPlugins(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	registerAllPlugins(r, w, ct, wc, false)

	// listAllPlugins writes to stdout; just verify it doesn't panic
	// We can't easily capture stdout in a test, but at least exercise the code
	listAllPlugins(r)
}
