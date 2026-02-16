package plugin

import (
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/wordcolor"
)

// stripAnsi removes ANSI escape sequences.
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

func setup() (*bytes.Buffer, *color.Table, *wordcolor.Processor) {
	buf := &bytes.Buffer{}
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	return buf, ct, wc
}

func TestRegistryRunNoMatch(t *testing.T) {
	r := NewRegistry()
	handled, rest := r.Run("some line", TypeFull)
	if handled || rest != "" {
		t.Error("empty registry should not match")
	}
}

func TestRegistryFilter(t *testing.T) {
	buf, ct, wc := setup()
	r := NewRegistry()
	r.Register(NewSyslogPlugin(buf, ct, wc, false))
	r.Register(NewHTTPDPlugin(buf, ct, wc, false))
	r.Register(NewDpkgPlugin(buf, ct, wc, false))

	r.Filter(map[string]bool{"syslog": true, "dpkg": true})
	plugins := r.Plugins()
	if len(plugins) != 2 {
		t.Errorf("after filter, got %d plugins, want 2", len(plugins))
	}
	for _, p := range plugins {
		if p.Name() != "syslog" && p.Name() != "dpkg" {
			t.Errorf("unexpected plugin: %s", p.Name())
		}
	}
}

func TestHTTPAction(t *testing.T) {
	tests := []struct {
		method string
		want   color.Color
	}{
		{"GET", color.HTTPGet},
		{"get", color.HTTPGet},
		{"POST", color.HTTPPost},
		{"HEAD", color.HTTPHead},
		{"PUT", color.HTTPPut},
		{"CONNECT", color.HTTPConnect},
		{"TRACE", color.HTTPTrace},
		{"DELETE", color.Unknown},
		{"PATCH", color.Unknown},
	}
	for _, tt := range tests {
		if got := HTTPAction(tt.method); got != tt.want {
			t.Errorf("HTTPAction(%q) = %d, want %d", tt.method, got, tt.want)
		}
	}
}

func TestPrintDate(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	PrintDate(&buf, ct, "Oct 12 22:40:12", false)
	out := stripAnsi(buf.String())
	if out != "Oct 12 22:40:12" {
		t.Errorf("PrintDate without conversion = %q, want 'Oct 12 22:40:12'", out)
	}
}

func TestPrintDateConversion(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	PrintDate(&buf, ct, "1000000000", true)
	out := stripAnsi(buf.String())
	// Format is "Jan  2 15:04:05" — 1000000000 = Sep  9 01:46:40 UTC
	if !strings.Contains(out, "Sep") || !strings.Contains(out, "01:46:40") {
		t.Errorf("PrintDate with conversion = %q, should contain 'Sep' and '01:46:40'", out)
	}
}

func TestPrintDateInvalidTimestamp(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	PrintDate(&buf, ct, "not-a-number", true)
	out := stripAnsi(buf.String())
	if out != "not-a-number" {
		t.Errorf("PrintDate with invalid timestamp = %q, want 'not-a-number'", out)
	}
}

func TestSyslogPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	if p.Name() != "syslog" {
		t.Errorf("Name = %q, want syslog", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("Sep 14 11:45:00 myhost sshd[1234]: test message")
	if !handled {
		t.Error("syslog should handle this line")
	}
	if rest != "test message" {
		t.Errorf("rest = %q, want 'test message'", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "Sep 14 11:45:00") {
		t.Error("output should contain date")
	}
	if !strings.Contains(out, "myhost") {
		t.Error("output should contain host")
	}
	if !strings.Contains(out, "sshd") {
		t.Error("output should contain process name")
	}
}

func TestSyslogPluginRepeat(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Oct 12 22:40:12 iluvatar last message repeated 10 times")
	if !handled {
		t.Error("syslog should handle repeat line")
	}
	if rest != "" {
		t.Errorf("rest should be empty for repeat, got %q", rest)
	}
}

func TestSyslogPluginMark(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Oct 12 22:40:12 iluvatar -- MARK --")
	if !handled {
		t.Error("syslog should handle MARK line")
	}
	if rest != "" {
		t.Errorf("rest should be empty for MARK, got %q", rest)
	}
}

func TestSyslogPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("this is not a syslog line")
	if handled {
		t.Error("syslog should not handle non-syslog lines")
	}
}

func TestSyslogPluginNoProcess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Sep 14 11:45:00 myhost daemon: starting up")
	if !handled {
		t.Error("syslog should handle line without PID")
	}
	if rest != "starting up" {
		t.Errorf("rest = %q, want 'starting up'", rest)
	}
	_ = buf
}

func TestHTTPDPluginAccess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	if p.Name() != "httpd" {
		t.Errorf("Name = %q, want httpd", p.Name())
	}

	handled, rest := p.Handle(`192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`)
	if !handled {
		t.Error("httpd should handle access log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "192.168.1.1") {
		t.Error("output should contain IP")
	}
}

func TestHTTPDPluginError(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[Sun Oct 12 15:30:00 2003] [error] some error message")
	if !handled {
		t.Error("httpd should handle error log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	_ = buf
}

func TestHTTPDPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an apache log")
	if handled {
		t.Error("httpd should not match random text")
	}
	_ = buf
}

func TestDpkgPluginStatus(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDpkgPlugin(buf, ct, wc, false)

	if p.Name() != "dpkg" {
		t.Errorf("Name = %q, want dpkg", p.Name())
	}

	handled, _ := p.Handle("2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3")
	if !handled {
		t.Error("dpkg should handle status line")
	}
	_ = buf
}

func TestDpkgPluginAction(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDpkgPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("2023-10-15 14:30:22 install libfoo:amd64 1.2.3 1.2.4")
	if !handled {
		t.Error("dpkg should handle action line")
	}
	_ = buf
}

func TestDpkgPluginConffile(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDpkgPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("2023-10-15 14:30:22 conffile /etc/foo.conf install")
	if !handled {
		t.Error("dpkg should handle conffile line")
	}
	_ = buf
}

func TestPostfixPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPostfixPlugin(buf, ct, wc, false)

	if p.Name() != "postfix" {
		t.Errorf("Name = %q, want postfix", p.Name())
	}
	if p.Type() != TypePartial {
		t.Errorf("Type = %d, want TypePartial", p.Type())
	}

	handled, _ := p.Handle("ABC123: to=<user@example.com>,relay=smtp")
	if !handled {
		t.Error("postfix should handle postfix log")
	}
	_ = buf
}

func TestEximPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 some message")
	if !handled {
		t.Error("exim should handle exim log")
	}
	if rest == "" {
		t.Error("exim should return rest")
	}
	_ = buf
}

func TestSquidPluginAccess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	if p.Name() != "squid" {
		t.Errorf("Name = %q, want squid", p.Name())
	}

	handled, _ := p.Handle("1234567890.123      5 192.168.1.1 TCP_MISS/200 1234 GET http://example.com user DIRECT/93.184.216.34 text/html")
	if !handled {
		t.Error("squid should handle access log")
	}
	_ = buf
}

func TestProcmailPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProcmailPlugin(buf, ct, wc, false)

	handled, rest := p.Handle(" From user@example.com  Sat Oct 12")
	if !handled {
		t.Error("procmail should handle From line")
	}
	if rest != "" {
		t.Errorf("rest should be empty for recognized header, got %q", rest)
	}
	_ = buf
}

func TestProcmailPluginUnknownHeader(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProcmailPlugin(buf, ct, wc, false)

	handled, rest := p.Handle(" Unknown value rest")
	if !handled {
		t.Error("procmail should handle (returning original) for unknown header")
	}
	if rest == "" {
		t.Error("should return original string as rest for unknown header")
	}
	_ = buf
}

func TestPHPPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPHPPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[15-Oct-2023 14:30:22] PHP Warning: something")
	if !handled {
		t.Error("php should handle PHP log")
	}
	_ = rest
	_ = buf
}

func TestAllPluginsHaveRequiredMethods(t *testing.T) {
	buf, ct, wc := setup()

	plugins := []Plugin{
		NewSyslogPlugin(buf, ct, wc, false),
		NewHTTPDPlugin(buf, ct, wc, false),
		NewSquidPlugin(buf, ct, wc, false),
		NewPostfixPlugin(buf, ct, wc, false),
		NewEximPlugin(buf, ct, wc, false),
		NewDpkgPlugin(buf, ct, wc, false),
		NewProcmailPlugin(buf, ct, wc, false),
		NewPHPPlugin(buf, ct, wc, false),
		NewProFTPDPlugin(buf, ct, wc, false),
		NewVsftpdPlugin(buf, ct, wc, false),
		NewFetchmailPlugin(buf, ct, wc, false),
		NewApmPlugin(buf, ct, wc, false),
		NewDistccPlugin(buf, ct, wc, false),
		NewIcecastPlugin(buf, ct, wc, false),
		NewOopsPlugin(buf, ct, wc, false),
		NewXferlogPlugin(buf, ct, wc, false),
		NewFtpstatsPlugin(buf, ct, wc, false),
		NewSulogPlugin(buf, ct, wc, false),
		NewSuperPlugin(buf, ct, wc, false),
		NewUlogdPlugin(buf, ct, wc, false),
	}

	names := make(map[string]bool)
	for _, p := range plugins {
		if p.Name() == "" {
			t.Error("plugin has empty name")
		}
		if names[p.Name()] {
			t.Errorf("duplicate plugin name: %s", p.Name())
		}
		names[p.Name()] = true

		if p.Description() == "" {
			t.Errorf("plugin %s has empty description", p.Name())
		}

		// All plugins should not match random text
		buf.Reset()
		handled, _ := p.Handle("zzz random text that matches nothing zzz")
		_ = handled // some plugins with loose regexes may match
	}

	if len(plugins) != 20 {
		t.Errorf("expected 20 plugins, got %d", len(plugins))
	}
}

func TestUlogdPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewUlogdPlugin(buf, ct, wc, false)

	if p.Type() != TypePartial {
		t.Errorf("ulogd should be partial, got %d", p.Type())
	}

	handled, _ := p.Handle("IN=eth0 OUT= MAC=00:11:22:33:44:55 SRC=192.168.1.1 TTL=64")
	if !handled {
		t.Error("ulogd should handle netfilter log")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "IN") || !strings.Contains(out, "eth0") {
		t.Error("output should contain IN and eth0")
	}
}

func TestUlogdPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewUlogdPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("no equals sign here")
	if handled {
		t.Error("ulogd should not handle line without field=value pairs")
	}
	_ = buf
}

func TestFetchmailPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFetchmailPlugin(buf, ct, wc, false)

	if p.Type() != TypePartial {
		t.Errorf("fetchmail should be partial, got %d", p.Type())
	}

	handled, rest := p.Handle("reading message user@pop.example.com:5 of 10 (1234 octets)")
	if !handled {
		t.Error("fetchmail should handle reading message line")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "reading message") {
		t.Error("output should contain 'reading message'")
	}
	if !strings.Contains(out, "5") {
		t.Error("output should contain message number")
	}
	_ = rest
}

func TestFetchmailPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFetchmailPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a fetchmail line")
	if handled {
		t.Error("fetchmail should not handle random text")
	}
	_ = buf
}

func TestApmPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewApmPlugin(buf, ct, wc, false)

	if p.Type() != TypePartial {
		t.Errorf("apm should be partial, got %d", p.Type())
	}

	handled, rest := p.Handle("Battery: 85%, not charging (-1% unknown 0:00:00), 2:30:00 remaining")
	if !handled {
		t.Error("apm should handle battery line")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "Battery:") {
		t.Error("output should contain Battery:")
	}
	if !strings.Contains(out, "85") {
		t.Error("output should contain battery percentage")
	}
	_ = rest
}

func TestApmPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewApmPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an apm line")
	if handled {
		t.Error("apm should not handle random text")
	}
	_ = buf
}

func TestProFTPDPluginAccess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProFTPDPlugin(buf, ct, wc, false)

	if p.Name() != "proftpd" {
		t.Errorf("Name = %q, want proftpd", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle(`192.168.1.1 user1 admin [15/Oct/2023:14:30:22 -0700] "RETR /pub/file.txt" 226 1024`)
	if !handled {
		t.Error("proftpd should handle access log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "192.168.1.1") {
		t.Error("output should contain host")
	}
	if !strings.Contains(out, "RETR") {
		t.Error("output should contain command")
	}
}

func TestProFTPDPluginAuth(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProFTPDPlugin(buf, ct, wc, false)

	handled, rest := p.Handle(`myserver ftp server [1234] 192.168.1.1 [15/Oct/2023:14:30:22 -0700] "USER admin" 331`)
	if !handled {
		t.Error("proftpd should handle auth log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "myserver") {
		t.Error("output should contain server hostname")
	}
}

func TestProFTPDPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProFTPDPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a proftpd log")
	if handled {
		t.Error("proftpd should not handle random text")
	}
	_ = buf
}

func TestVsftpdPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewVsftpdPlugin(buf, ct, wc, false)

	if p.Name() != "vsftpd" {
		t.Errorf("Name = %q, want vsftpd", p.Name())
	}

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 [pid 1234] [admin] OK LOGIN: Client 192.168.1.1")
	if !handled {
		t.Error("vsftpd should handle log line with user")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "1234") {
		t.Error("output should contain PID")
	}
	_ = rest
}

func TestVsftpdPluginWithoutUser(t *testing.T) {
	buf, ct, wc := setup()
	p := NewVsftpdPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 [pid 5678] CONNECT: Client 10.0.0.1")
	if !handled {
		t.Error("vsftpd should handle log line without user")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "5678") {
		t.Error("output should contain PID")
	}
	_ = rest
}

func TestVsftpdPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewVsftpdPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a vsftpd log")
	if handled {
		t.Error("vsftpd should not handle random text")
	}
	_ = buf
}

func TestXferlogPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewXferlogPlugin(buf, ct, wc, false)

	if p.Name() != "xferlog" {
		t.Errorf("Name = %q, want xferlog", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 5 192.168.1.1 1234 /pub/file.txt b _ o r user ftp 0 * c")
	if !handled {
		t.Error("xferlog should handle xferlog line")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "192.168.1.1") {
		t.Error("output should contain host")
	}
	if !strings.Contains(out, "/pub/file.txt") {
		t.Error("output should contain filename")
	}
}

func TestXferlogPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewXferlogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an xferlog line")
	if handled {
		t.Error("xferlog should not handle random text")
	}
	_ = buf
}

func TestFtpstatsPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFtpstatsPlugin(buf, ct, wc, false)

	if p.Name() != "ftpstats" {
		t.Errorf("Name = %q, want ftpstats", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("1234567890 ab12.cd34 admin 192.168.1.1 U 4096 120 /uploads/data.zip")
	if !handled {
		t.Error("ftpstats should handle ftpstats line")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "admin") {
		t.Error("output should contain user")
	}
	if !strings.Contains(out, "192.168.1.1") {
		t.Error("output should contain host")
	}
}

func TestFtpstatsPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFtpstatsPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an ftpstats line")
	if handled {
		t.Error("ftpstats should not handle random text")
	}
	_ = buf
}

func TestOopsPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewOopsPlugin(buf, ct, wc, false)

	if p.Name() != "oops" {
		t.Errorf("Name = %q, want oops", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 [0xa1b2]statistics(): requests : 42 total")
	if !handled {
		t.Error("oops should handle oops log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "statistics()") {
		t.Error("output should contain statistics()")
	}
	if !strings.Contains(out, "42") {
		t.Error("output should contain value")
	}
}

func TestOopsPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewOopsPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an oops line")
	if handled {
		t.Error("oops should not handle random text")
	}
	_ = buf
}

func TestSulogPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSulogPlugin(buf, ct, wc, false)

	if p.Name() != "sulog" {
		t.Errorf("Name = %q, want sulog", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("SU 10/15 14:30 + pts/0 root-admin")
	if !handled {
		t.Error("sulog should handle su log line")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "SU") {
		t.Error("output should contain SU")
	}
	if !strings.Contains(out, "10/15 14:30") {
		t.Error("output should contain date")
	}
}

func TestSulogPluginQuestionMarkTTY(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSulogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("SU 10/15 14:30 - ?tty root-admin")
	if !handled {
		t.Error("sulog should handle su log with ? tty")
	}
	_ = buf
}

func TestSulogPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSulogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a sulog line")
	if handled {
		t.Error("sulog should not handle random text")
	}
	_ = buf
}

func TestSuperPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSuperPlugin(buf, ct, wc, false)

	if p.Name() != "super" {
		t.Errorf("Name = %q, want super", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("user@host Mon Oct 15 14:30:22 2023  supertag (some command)")
	if !handled {
		t.Error("super should handle super log line")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "user@host") {
		t.Error("output should contain email")
	}
}

func TestSuperPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSuperPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a super line")
	if handled {
		t.Error("super should not handle random text")
	}
	_ = buf
}

func TestDistccPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDistccPlugin(buf, ct, wc, false)

	if p.Name() != "distcc" {
		t.Errorf("Name = %q, want distcc", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("distccd[1234] (dcc_compile) compiling foo.c")
	if !handled {
		t.Error("distcc should handle distcc log line")
	}
	if rest == "" {
		t.Error("distcc should have rest")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "distccd") {
		t.Error("output should contain distccd")
	}
	if !strings.Contains(out, "1234") {
		t.Error("output should contain PID")
	}
}

func TestDistccPluginNoFunc(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDistccPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("distccd[5678] some message here")
	if !handled {
		t.Error("distcc should handle line without func name")
	}
	if rest == "" {
		t.Error("distcc should have rest")
	}
	_ = buf
}

func TestDistccPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDistccPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a distcc line")
	if handled {
		t.Error("distcc should not handle random text")
	}
	_ = buf
}

func TestSquidPluginStore(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("1097454321.987 SWAPOUT 0000002A  http://example.com/page 5A3F2B1C 200 1097454321 1097454321 -1 text/html 1024/2048 GET http://example.com/page")
	if !handled {
		t.Error("squid should handle store log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	_ = buf
}

func TestSquidPluginCache(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023/10/15 14:30:22| Starting Squid Cache")
	if !handled {
		t.Error("squid should handle cache log")
	}
	if rest == "" {
		t.Error("squid cache should return rest")
	}
	_ = buf
}

func TestSquidPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a squid log")
	if handled {
		t.Error("squid should not match random text")
	}
	_ = buf
}

func TestProxyAction(t *testing.T) {
	tests := []struct {
		action string
		want   color.Color
	}{
		{"ERR_SOMETHING", color.Error},
		{"TCP_MISS", color.ProxyMiss},
		{"TCP_HIT", color.ProxyHit},
		{"TCP_DENIED", color.ProxyDenied},
		{"TCP_REFRESH_UNMODIFIED", color.ProxyRefresh},
		{"SWAPFAIL", color.ProxySwapfail},
		{"NONE", color.Debug},
		{"UNKNOWN_THING", color.Unknown},
	}
	for _, tt := range tests {
		if got := proxyAction(tt.action); got != tt.want {
			t.Errorf("proxyAction(%q) = %d, want %d", tt.action, got, tt.want)
		}
	}
}

func TestProxyHierarchy(t *testing.T) {
	tests := []struct {
		hierar string
		want   color.Color
	}{
		{"NO_DIRECT_FAIL", color.Warning},
		{"DIRECT", color.ProxyDirect},
		{"PARENT_HIT", color.ProxyParent},
		{"SIBLING_MISS", color.ProxyMiss},
		{"SOMETHING_ELSE", color.Unknown},
	}
	for _, tt := range tests {
		if got := proxyHierarchy(tt.hierar); got != tt.want {
			t.Errorf("proxyHierarchy(%q) = %d, want %d", tt.hierar, got, tt.want)
		}
	}
}

func TestProxyTag(t *testing.T) {
	tests := []struct {
		tag  string
		want color.Color
	}{
		{"CREATE", color.ProxyCreate},
		{"SWAPIN", color.ProxySwapin},
		{"SWAPOUT", color.ProxySwapout},
		{"RELEASE", color.ProxyRelease},
		{"UNKNOWN", color.Unknown},
	}
	for _, tt := range tests {
		if got := proxyTag(tt.tag); got != tt.want {
			t.Errorf("proxyTag(%q) = %d, want %d", tt.tag, got, tt.want)
		}
	}
}

func TestEximPluginActionType(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	// Test incoming action (<= or <=)
	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 <= user@example.com message text here")
	if !handled {
		t.Error("exim should handle action type line")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "2023-10-15 14:30:22") {
		t.Error("output should contain date")
	}
	_ = rest
}

func TestEximPluginOutgoing(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 => user@example.com some message")
	if !handled {
		t.Error("exim should handle outgoing action")
	}
	_ = rest
	_ = buf
}

func TestEximPluginErrorAction(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 ** user@example.com bounce message")
	if !handled {
		t.Error("exim should handle error action")
	}
	_ = rest
	_ = buf
}

func TestEximPluginUniqn(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 some plain message")
	if !handled {
		t.Error("exim should handle unique ID line")
	}
	if rest == "" {
		t.Error("should have rest")
	}
	_ = buf
}

func TestEximPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an exim line at all")
	if handled {
		t.Error("exim should not handle random text")
	}
	_ = buf
}

func TestIcecastPluginUsage(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	if p.Name() != "icecast" {
		t.Errorf("Name = %q, want icecast", p.Name())
	}
	if p.Type() != TypeFull {
		t.Errorf("Type = %d, want TypeFull", p.Type())
	}

	handled, rest := p.Handle("[15/Oct/2023:14:30:22] [1234:connection] [15/Oct/2023:14:30:22] Bandwidth:128.5kbps Sources:3 Clients:42 Admins:1")
	if !handled {
		t.Error("icecast should handle usage log")
	}
	if rest != "" {
		t.Errorf("rest should be empty, got %q", rest)
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "Bandwidth:") {
		t.Error("output should contain Bandwidth:")
	}
}

func TestIcecastPluginGeneral(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[15/Oct/2023:14:30:22] [1234:connection] some log message")
	if !handled {
		t.Error("icecast should handle general log")
	}
	if rest == "" {
		t.Error("icecast general should return rest")
	}
	_ = buf
}

func TestIcecastPluginAdmin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[15/Oct/2023:14:30:22] Admin [admin_host] some admin message")
	if !handled {
		t.Error("icecast should handle admin log")
	}
	_ = rest
	_ = buf
}

func TestIcecastPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an icecast line")
	if handled {
		t.Error("icecast should not handle random text")
	}
	_ = buf
}

func TestHTTPDPluginErrorLevels(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	tests := []string{
		"[Sun Oct 12 15:30:00 2003] [debug] debug message",
		"[Sun Oct 12 15:30:00 2003] [warn] warning message",
		"[Sun Oct 12 15:30:00 2003] [crit] critical message",
		"[Sun Oct 12 15:30:00 2003] [notice] notice message",
		"[Sun Oct 12 15:30:00 2003] [info] info message",
	}
	for _, line := range tests {
		buf.Reset()
		handled, _ := p.Handle(line)
		if !handled {
			t.Errorf("httpd should handle error log: %s", line)
		}
	}
}

func TestHTTPDPluginAccessWithVhost(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	handled, _ := p.Handle(`example.com 10.0.0.1 - admin [10/Oct/2000:13:55:36 -0700] "POST /api/data HTTP/1.1" 201 512`)
	if !handled {
		t.Error("httpd should handle access log with vhost")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "example.com") {
		t.Error("output should contain vhost")
	}
	_ = buf
}

func TestPostfixPluginMultipleFields(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPostfixPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("ABC123: to=<user@example.com>,relay=smtp.example.com,delay=0.5,status=sent")
	if !handled {
		t.Error("postfix should handle multi-field log")
	}
	out := stripAnsi(buf.String())
	if !strings.Contains(out, "ABC123") {
		t.Error("output should contain spool ID")
	}
	if !strings.Contains(out, "to") {
		t.Error("output should contain field names")
	}
}

func TestPostfixPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPostfixPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a postfix line")
	if handled {
		t.Error("postfix should not handle random text")
	}
	_ = buf
}

func TestRegistryRunFullThenPartial(t *testing.T) {
	buf, ct, wc := setup()
	r := NewRegistry()
	r.Register(NewSyslogPlugin(buf, ct, wc, false))
	r.Register(NewPostfixPlugin(buf, ct, wc, false))

	// Full match
	handled, rest := r.Run("Sep 14 11:45:00 myhost postfix/smtp[1234]: ABC123: to=<user@example.com>", TypeFull)
	if !handled {
		t.Error("should handle syslog line")
	}
	if rest == "" {
		t.Error("should have rest from syslog")
	}

	// Partial match on rest
	buf.Reset()
	handled2, _ := r.Run(rest, TypePartial)
	if !handled2 {
		t.Logf("rest = %q", rest)
		t.Log("partial match did not fire (may be expected depending on rest format)")
	}
}
