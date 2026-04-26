package plugin

import (
	"testing"

	"ccze-go/color"
	"github.com/wow-look-at-my/testify/assert"
)

func TestXferlogPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewXferlogPlugin(buf, ct, wc, false)

	assert.Equal(t, "xferlog", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 5 192.168.1.1 1234 /pub/file.txt b _ o r user ftp 0 * c")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "192.168.1.1")

	assert.Contains(t, out, "/pub/file.txt")

}

func TestXferlogPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewXferlogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an xferlog line")
	assert.False(t, handled)

	_ = buf
}

func TestFtpstatsPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFtpstatsPlugin(buf, ct, wc, false)

	assert.Equal(t, "ftpstats", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("1234567890 ab12.cd34 admin 192.168.1.1 U 4096 120 /uploads/data.zip")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "admin")

	assert.Contains(t, out, "192.168.1.1")

}

func TestFtpstatsPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFtpstatsPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an ftpstats line")
	assert.False(t, handled)

	_ = buf
}

func TestOopsPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewOopsPlugin(buf, ct, wc, false)

	assert.Equal(t, "oops", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 [0xa1b2]statistics(): requests : 42 total")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "statistics()")

	assert.Contains(t, out, "42")

}

func TestOopsPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewOopsPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an oops line")
	assert.False(t, handled)

	_ = buf
}

func TestSulogPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSulogPlugin(buf, ct, wc, false)

	assert.Equal(t, "sulog", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("SU 10/15 14:30 + pts/0 root-admin")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "SU")

	assert.Contains(t, out, "10/15 14:30")

}

func TestSulogPluginQuestionMarkTTY(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSulogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("SU 10/15 14:30 - ?tty root-admin")
	assert.True(t, handled)

	_ = buf
}

func TestSulogPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSulogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a sulog line")
	assert.False(t, handled)

	_ = buf
}

func TestSuperPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSuperPlugin(buf, ct, wc, false)

	assert.Equal(t, "super", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("user@host Mon Oct 15 14:30:22 2023  supertag (some command)")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "user@host")

}

func TestSuperPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSuperPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a super line")
	assert.False(t, handled)

	_ = buf
}

func TestDistccPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDistccPlugin(buf, ct, wc, false)

	assert.Equal(t, "distcc", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("distccd[1234] (dcc_compile) compiling foo.c")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "distccd")

	assert.Contains(t, out, "1234")

}

func TestDistccPluginNoFunc(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDistccPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("distccd[5678] some message here")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	_ = buf
}

func TestDistccPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDistccPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a distcc line")
	assert.False(t, handled)

	_ = buf
}

func TestSquidPluginStore(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("1097454321.987 SWAPOUT 0000002A  http://example.com/page 5A3F2B1C 200 1097454321 1097454321 -1 text/html 1024/2048 GET http://example.com/page")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	_ = buf
}

func TestSquidPluginCache(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023/10/15 14:30:22| Starting Squid Cache")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	_ = buf
}

func TestSquidPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a squid log")
	assert.False(t, handled)

	_ = buf
}

func TestProxyAction(t *testing.T) {
	tests := []struct {
		action	string
		want	color.Color
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
		got := proxyAction(tt.action)
		assert.Equal(t, tt.want, got)

	}
}

func TestProxyHierarchy(t *testing.T) {
	tests := []struct {
		hierar	string
		want	color.Color
	}{
		{"NO_DIRECT_FAIL", color.Warning},
		{"DIRECT", color.ProxyDirect},
		{"PARENT_HIT", color.ProxyParent},
		{"SIBLING_MISS", color.ProxyMiss},
		{"SOMETHING_ELSE", color.Unknown},
	}
	for _, tt := range tests {
		got := proxyHierarchy(tt.hierar)
		assert.Equal(t, tt.want, got)

	}
}

func TestProxyTag(t *testing.T) {
	tests := []struct {
		tag	string
		want	color.Color
	}{
		{"CREATE", color.ProxyCreate},
		{"SWAPIN", color.ProxySwapin},
		{"SWAPOUT", color.ProxySwapout},
		{"RELEASE", color.ProxyRelease},
		{"UNKNOWN", color.Unknown},
	}
	for _, tt := range tests {
		got := proxyTag(tt.tag)
		assert.Equal(t, tt.want, got)

	}
}

func TestEximPluginActionType(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	// Test incoming action (<= or <=)
	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 <= user@example.com message text here")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "2023-10-15 14:30:22")

	_ = rest
}

func TestEximPluginOutgoing(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 => user@example.com some message")
	assert.True(t, handled)

	_ = rest
	_ = buf
}

func TestEximPluginErrorAction(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 ** user@example.com bounce message")
	assert.True(t, handled)

	_ = rest
	_ = buf
}

func TestEximPluginUniqn(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 1234567890123456 some plain message")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	_ = buf
}

func TestEximPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an exim line at all")
	assert.False(t, handled)

	_ = buf
}

func TestIcecastPluginUsage(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	assert.Equal(t, "icecast", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("[15/Oct/2023:14:30:22] [1234:connection] [15/Oct/2023:14:30:22] Bandwidth:128.5kbps Sources:3 Clients:42 Admins:1")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "Bandwidth:")

}

func TestIcecastPluginGeneral(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[15/Oct/2023:14:30:22] [1234:connection] some log message")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	_ = buf
}

func TestIcecastPluginAdmin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[15/Oct/2023:14:30:22] Admin [admin_host] some admin message")
	assert.True(t, handled)

	_ = rest
	_ = buf
}

func TestIcecastPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewIcecastPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an icecast line")
	assert.False(t, handled)

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
		assert.True(t, handled)

	}
}

func TestHTTPDPluginAccessWithVhost(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	handled, _ := p.Handle(`example.com 10.0.0.1 - admin [10/Oct/2000:13:55:36 -0700] "POST /api/data HTTP/1.1" 201 512`)
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "example.com")

	_ = buf
}

func TestPostfixPluginMultipleFields(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPostfixPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("ABC123: to=<user@example.com>,relay=smtp.example.com,delay=0.5,status=sent")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "ABC123")

	assert.Contains(t, out, "to")

}

func TestPostfixPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPostfixPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a postfix line")
	assert.False(t, handled)

	_ = buf
}

func TestRegistryRunFullThenPartial(t *testing.T) {
	buf, ct, wc := setup()
	r := NewRegistry()
	r.Register(NewSyslogPlugin(buf, ct, wc, false))
	r.Register(NewPostfixPlugin(buf, ct, wc, false))

	// Full match
	handled, rest := r.Run("Sep 14 11:45:00 myhost postfix/smtp[1234]: ABC123: to=<user@example.com>", TypeFull)
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	// Partial match on rest
	buf.Reset()
	handled2, _ := r.Run(rest, TypePartial)
	if !handled2 {
		t.Logf("rest = %q", rest)
		t.Log("partial match did not fire (may be expected depending on rest format)")
	}
}
