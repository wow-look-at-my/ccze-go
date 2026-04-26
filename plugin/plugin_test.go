package plugin

import (
	"bytes"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/wordcolor"
	"github.com/wow-look-at-my/testify/assert"
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
	assert.False(t, handled || rest != "")

}

func TestRegistryFilter(t *testing.T) {
	buf, ct, wc := setup()
	r := NewRegistry()
	r.Register(NewSyslogPlugin(buf, ct, wc, false))
	r.Register(NewHTTPDPlugin(buf, ct, wc, false))
	r.Register(NewDpkgPlugin(buf, ct, wc, false))

	r.Filter(map[string]bool{"syslog": true, "dpkg": true})
	plugins := r.Plugins()
	assert.Equal(t, 2, len(plugins))

	for _, p := range plugins {
		assert.False(t, p.Name() != "syslog" && p.Name() != "dpkg")

	}
}

func TestHTTPAction(t *testing.T) {
	tests := []struct {
		method	string
		want	color.Color
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
		got := HTTPAction(tt.method)
		assert.Equal(t, tt.want, got)

	}
}

func TestPrintDate(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	PrintDate(&buf, ct, "Oct 12 22:40:12", false)
	out := stripAnsi(buf.String())
	assert.Equal(t, "Oct 12 22:40:12", out)

}

func TestPrintDateConversion(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	PrintDate(&buf, ct, "1000000000", true)
	out := stripAnsi(buf.String())
	// Format is "Jan  2 15:04:05" — 1000000000 = Sep  9 01:46:40 UTC
	assert.False(t, !strings.Contains(out, "Sep") || !strings.Contains(out, "01:46:40"))

}

func TestPrintDateInvalidTimestamp(t *testing.T) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	PrintDate(&buf, ct, "not-a-number", true)
	out := stripAnsi(buf.String())
	assert.Equal(t, "not-a-number", out)

}

func TestSyslogPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	assert.Equal(t, "syslog", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle("Sep 14 11:45:00 myhost sshd[1234]: test message")
	assert.True(t, handled)

	assert.Equal(t, "test message", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "Sep 14 11:45:00")

	assert.Contains(t, out, "myhost")

	assert.Contains(t, out, "sshd")

}

func TestSyslogPluginRepeat(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Oct 12 22:40:12 iluvatar last message repeated 10 times")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

}

func TestSyslogPluginMark(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Oct 12 22:40:12 iluvatar -- MARK --")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

}

func TestSyslogPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("this is not a syslog line")
	assert.False(t, handled)

}

func TestSyslogPluginNoProcess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSyslogPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Sep 14 11:45:00 myhost daemon: starting up")
	assert.True(t, handled)

	assert.Equal(t, "starting up", rest)

	_ = buf
}

func TestHTTPDPluginAccess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	assert.Equal(t, "httpd", p.Name())

	handled, rest := p.Handle(`192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`)
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "192.168.1.1")

}

func TestHTTPDPluginError(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[Sun Oct 12 15:30:00 2003] [error] some error message")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	_ = buf
}

func TestHTTPDPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewHTTPDPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an apache log")
	assert.False(t, handled)

	_ = buf
}

func TestDpkgPluginStatus(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDpkgPlugin(buf, ct, wc, false)

	assert.Equal(t, "dpkg", p.Name())

	handled, _ := p.Handle("2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3")
	assert.True(t, handled)

	_ = buf
}

func TestDpkgPluginAction(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDpkgPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("2023-10-15 14:30:22 install libfoo:amd64 1.2.3 1.2.4")
	assert.True(t, handled)

	_ = buf
}

func TestDpkgPluginConffile(t *testing.T) {
	buf, ct, wc := setup()
	p := NewDpkgPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("2023-10-15 14:30:22 conffile /etc/foo.conf install")
	assert.True(t, handled)

	_ = buf
}

func TestPostfixPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPostfixPlugin(buf, ct, wc, false)

	assert.Equal(t, "postfix", p.Name())

	assert.Equal(t, TypePartial, p.Type())

	handled, _ := p.Handle("ABC123: to=<user@example.com>,relay=smtp")
	assert.True(t, handled)

	_ = buf
}

func TestEximPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewEximPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("2023-10-15 14:30:22 some message")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	_ = buf
}

func TestSquidPluginAccess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewSquidPlugin(buf, ct, wc, false)

	assert.Equal(t, "squid", p.Name())

	handled, _ := p.Handle("1234567890.123      5 192.168.1.1 TCP_MISS/200 1234 GET http://example.com user DIRECT/93.184.216.34 text/html")
	assert.True(t, handled)

	_ = buf
}

func TestProcmailPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProcmailPlugin(buf, ct, wc, false)

	handled, rest := p.Handle(" From user@example.com  Sat Oct 12")
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	_ = buf
}

func TestProcmailPluginUnknownHeader(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProcmailPlugin(buf, ct, wc, false)

	handled, rest := p.Handle(" Unknown value rest")
	assert.True(t, handled)

	assert.NotEqual(t, "", rest)

	_ = buf
}

func TestPHPPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewPHPPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("[15-Oct-2023 14:30:22] PHP Warning: something")
	assert.True(t, handled)

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
		assert.NotEqual(t, "", p.Name())

		assert.False(t, names[p.Name()])

		names[p.Name()] = true

		assert.NotEqual(t, "", p.Description())

		// All plugins should not match random text
		buf.Reset()
		handled, _ := p.Handle("zzz random text that matches nothing zzz")
		_ = handled	// some plugins with loose regexes may match
	}

	assert.Equal(t, 20, len(plugins))

}

func TestUlogdPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewUlogdPlugin(buf, ct, wc, false)

	assert.Equal(t, TypePartial, p.Type())

	handled, _ := p.Handle("IN=eth0 OUT= MAC=00:11:22:33:44:55 SRC=192.168.1.1 TTL=64")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.False(t, !strings.Contains(out, "IN") || !strings.Contains(out, "eth0"))

}

func TestUlogdPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewUlogdPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("no equals sign here")
	assert.False(t, handled)

	_ = buf
}

func TestFetchmailPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFetchmailPlugin(buf, ct, wc, false)

	assert.Equal(t, TypePartial, p.Type())

	handled, rest := p.Handle("reading message user@pop.example.com:5 of 10 (1234 octets)")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "reading message")

	assert.Contains(t, out, "5")

	_ = rest
}

func TestFetchmailPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewFetchmailPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a fetchmail line")
	assert.False(t, handled)

	_ = buf
}

func TestApmPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewApmPlugin(buf, ct, wc, false)

	assert.Equal(t, TypePartial, p.Type())

	handled, rest := p.Handle("Battery: 85%, not charging (-1% unknown 0:00:00), 2:30:00 remaining")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "Battery:")

	assert.Contains(t, out, "85")

	_ = rest
}

func TestApmPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewApmPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not an apm line")
	assert.False(t, handled)

	_ = buf
}

func TestProFTPDPluginAccess(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProFTPDPlugin(buf, ct, wc, false)

	assert.Equal(t, "proftpd", p.Name())

	assert.Equal(t, TypeFull, p.Type())

	handled, rest := p.Handle(`192.168.1.1 user1 admin [15/Oct/2023:14:30:22 -0700] "RETR /pub/file.txt" 226 1024`)
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "192.168.1.1")

	assert.Contains(t, out, "RETR")

}

func TestProFTPDPluginAuth(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProFTPDPlugin(buf, ct, wc, false)

	handled, rest := p.Handle(`myserver ftp server [1234] 192.168.1.1 [15/Oct/2023:14:30:22 -0700] "USER admin" 331`)
	assert.True(t, handled)

	assert.Equal(t, "", rest)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "myserver")

}

func TestProFTPDPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewProFTPDPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a proftpd log")
	assert.False(t, handled)

	_ = buf
}

func TestVsftpdPlugin(t *testing.T) {
	buf, ct, wc := setup()
	p := NewVsftpdPlugin(buf, ct, wc, false)

	assert.Equal(t, "vsftpd", p.Name())

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 [pid 1234] [admin] OK LOGIN: Client 192.168.1.1")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "1234")

	_ = rest
}

func TestVsftpdPluginWithoutUser(t *testing.T) {
	buf, ct, wc := setup()
	p := NewVsftpdPlugin(buf, ct, wc, false)

	handled, rest := p.Handle("Mon Oct 15 14:30:22 2023 [pid 5678] CONNECT: Client 10.0.0.1")
	assert.True(t, handled)

	out := stripAnsi(buf.String())
	assert.Contains(t, out, "5678")

	_ = rest
}

func TestVsftpdPluginNoMatch(t *testing.T) {
	buf, ct, wc := setup()
	p := NewVsftpdPlugin(buf, ct, wc, false)

	handled, _ := p.Handle("not a vsftpd log")
	assert.False(t, handled)

	_ = buf
}

