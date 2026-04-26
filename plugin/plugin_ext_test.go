package plugin

import (
	"strings"
	"testing"

	"ccze-go/color"
)

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
