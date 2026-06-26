package main

// benchmark_test.go contains Go benchmarks for ccze-go components and a
// throughput comparison against the C ccze binary (when available).
//
// Run benchmarks:
//   go test -bench=. -benchmem
//
// Run only C comparison:
//   go test -bench=BenchmarkVsC -benchtime=3s

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
	"github.com/stretchr/testify/require"
)

// --------------------------------------------------------------------------
// Component benchmarks
// --------------------------------------------------------------------------

func BenchmarkColorTableInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		color.NewTable(true)
	}
}

func BenchmarkColorWriteColored(b *testing.B) {
	ct := color.NewTable(true)
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		ct.WriteColored(&buf, color.Date, "Sep 14 11:45:00")
	}
}

func BenchmarkColorParseLine(b *testing.B) {
	ct := color.NewTable(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct.ParseLine("date bold cyan")
	}
}

func BenchmarkWordcolorProcess(b *testing.B) {
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		wc.Process(&buf, "connection from 192.168.1.1 to http://example.com user@test.com version 2.3.7 error", true, false)
	}
}

func BenchmarkWordcolorProcessOne(b *testing.B) {
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	var buf bytes.Buffer
	words := []string{
		"192.168.1.1", "http://example.com/path", "user@example.com",
		"2.3.7", "/etc/passwd", "error", "linux", "aa:bb:cc:dd:ee:ff",
		"0x1234abcd", "sigterm", "150mb", "12:30:45", "42",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		wc.ProcessOne(&buf, words[i%len(words)], false)
	}
}

// --------------------------------------------------------------------------
// Plugin benchmarks
// --------------------------------------------------------------------------

func BenchmarkSyslogPlugin(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewSyslogPlugin(&buf, ct, wc, false)
	line := "Sep 14 11:45:00 myhost sshd[1234]: Accepted publickey for admin from 10.0.0.5 port 22"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		p.Handle(line)
	}
}

func BenchmarkHTTPDPlugin(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewHTTPDPlugin(&buf, ct, wc, false)
	line := `192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /index.html HTTP/1.0" 200 2326`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		p.Handle(line)
	}
}

func BenchmarkDpkgPlugin(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewDpkgPlugin(&buf, ct, wc, false)
	line := "2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		p.Handle(line)
	}
}

func BenchmarkSquidPlugin(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewSquidPlugin(&buf, ct, wc, false)
	line := "1234567890.123      5 192.168.1.1 TCP_MISS/200 1234 GET http://example.com user DIRECT/93.184.216.34 text/html"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		p.Handle(line)
	}
}

func BenchmarkPostfixPlugin(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	p := plugin.NewPostfixPlugin(&buf, ct, wc, false)
	line := "ABC123: to=<user@example.com>,relay=smtp.example.com[93.184.216.34]:25,delay=0.5,status=sent"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		p.Handle(line)
	}
}

func BenchmarkRegistryRun(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	// Register a few common plugins
	r.Register(plugin.NewSyslogPlugin(&buf, ct, wc, false))
	r.Register(plugin.NewHTTPDPlugin(&buf, ct, wc, false))
	r.Register(plugin.NewDpkgPlugin(&buf, ct, wc, false))
	r.Register(plugin.NewPostfixPlugin(&buf, ct, wc, false))

	line := "Sep 14 11:45:00 myhost sshd[1234]: Connection closed"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		r.Run(line, plugin.TypeFull)
	}
}

func BenchmarkRegistryRunNoMatch(b *testing.B) {
	var buf bytes.Buffer
	ct := color.NewTable(true)
	wc := wordcolor.New(ct)
	r := plugin.NewRegistry()

	r.Register(plugin.NewSyslogPlugin(&buf, ct, wc, false))
	r.Register(plugin.NewHTTPDPlugin(&buf, ct, wc, false))
	r.Register(plugin.NewDpkgPlugin(&buf, ct, wc, false))

	line := "this line matches no plugin format at all"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		r.Run(line, plugin.TypeFull)
	}
}

// --------------------------------------------------------------------------
// Full pipeline benchmarks
// --------------------------------------------------------------------------

func BenchmarkFullPipelineSyslog(b *testing.B) {
	line := "Sep 14 11:45:00 myhost sshd[1234]: Accepted publickey for admin from 10.0.0.5 port 22"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processLine(line)
	}
}

func BenchmarkFullPipelineHTTPD(b *testing.B) {
	line := `192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /index.html HTTP/1.0" 200 2326`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processLine(line)
	}
}

func BenchmarkFullPipelinePlainText(b *testing.B) {
	line := "something failed with error at 0x1234abcd connecting to 192.168.0.1 reading /var/log/syslog version 6.1.0"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processLine(line)
	}
}

func BenchmarkFullPipelineMixed(b *testing.B) {
	lines := []string{
		"Sep 14 11:45:00 myhost sshd[1234]: Connection closed by 192.168.1.100 port 52431",
		`192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /page HTTP/1.0" 200 2326`,
		"[Sun Oct 12 15:30:00 2003] [error] client denied",
		"2023-10-15 14:30:22 status installed libfoo:amd64 1.2.3",
		"something failed with error at 0x1234abcd",
		"Sep 14 11:45:00 mailhost postfix/smtp[1234]: ABC123: to=<user@example.com>,relay=smtp.example.com",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processLine(lines[i%len(lines)])
	}
}

// --------------------------------------------------------------------------
// Throughput benchmark: Go vs C ccze
// --------------------------------------------------------------------------

// benchPipe is a shared helper for the shell-out benchmarks.
// It launches the given binary with "-A", pipes input b.N times, and waits.
func benchPipe(b *testing.B, binPath string, input []byte, totalBytes int64) {
	b.Helper()
	b.SetBytes(totalBytes)
	b.ResetTimer()

	cmd := exec.Command(binPath, "-A")
	stdin, err := cmd.StdinPipe()
	require.Nil(b, err)

	cmd.Stdout = io.Discard
	require.NoError(b, cmd.Start())

	for i := 0; i < b.N; i++ {
		_, err := stdin.Write(input)
		require.Nil(b, err)

	}

	stdin.Close()
	require.NoError(b, cmd.Wait())

}

// buildGoBinary builds the ccze-go binary into a temp dir and returns its path.
func buildGoBinary(b *testing.B) string {
	b.Helper()
	tmp := b.TempDir()
	binPath := tmp + "/ccze-go"
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Stderr = os.Stderr
	require.NoError(b, cmd.Run())

	return binPath
}

func BenchmarkVsCThroughput(b *testing.B) {
	data, err := os.ReadFile("testdata/mixed.log")
	require.Nil(b, err)

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	input := data

	var totalBytes int64
	for _, l := range lines {
		totalBytes += int64(len(l))
	}

	b.Run("InProcess", func(b *testing.B) {
		b.SetBytes(totalBytes)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			ct := color.NewTable(true)
			wc := wordcolor.New(ct)
			r := plugin.NewRegistry()
			registerAllPlugins(r, w, ct, wc, false)

			for _, line := range lines {
				handled, rest := r.Run(line, plugin.TypeFull)
				if rest != "" {
					handled2, rest2 := r.Run(rest, plugin.TypePartial)
					if !handled2 {
						wc.Process(w, rest, true, false)
					} else {
						wc.Process(w, rest2, true, false)
					}
					ct.WriteNewline(w)
				}
				if !handled {
					wc.Process(w, line, true, false)
					ct.WriteNewline(w)
				}
			}
			w.Flush()
		}
	})

	b.Run("Go", func(b *testing.B) {
		binPath := buildGoBinary(b)
		benchPipe(b, binPath, input, totalBytes)
	})

	b.Run("C", func(b *testing.B) {
		cczePath, err := exec.LookPath("ccze")
		if err != nil {
			b.Skip("C ccze binary not found; skipping. Install with: apt install ccze")
		}
		benchPipe(b, cczePath, input, totalBytes)
	})
}

// --------------------------------------------------------------------------
// Benchmark summary helper for CI
// --------------------------------------------------------------------------

// TestBenchmarkSummary runs key benchmarks and prints results in a format
// suitable for CI step summaries. Run with: go test -v -run TestBenchmarkSummary
func TestBenchmarkSummary(t *testing.T) {
	if os.Getenv("CCZE_BENCH_SUMMARY") == "" {
		t.Skip("set CCZE_BENCH_SUMMARY=1 to run benchmark summary")
	}

	data, err := os.ReadFile("testdata/mixed.log")
	require.Nil(t, err)

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	input := data

	var totalBytes int64
	for _, l := range lines {
		totalBytes += int64(len(l))
	}

	// Build the Go binary once
	tmp := t.TempDir()
	goBin := tmp + "/ccze-go"
	cmd := exec.Command("go", "build", "-o", goBin, ".")
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	// In-process benchmark
	inProcResult := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			ct := color.NewTable(true)
			wc := wordcolor.New(ct)
			r := plugin.NewRegistry()
			registerAllPlugins(r, w, ct, wc, false)
			for _, line := range lines {
				handled, rest := r.Run(line, plugin.TypeFull)
				if rest != "" {
					handled2, rest2 := r.Run(rest, plugin.TypePartial)
					if !handled2 {
						wc.Process(w, rest, true, false)
					} else {
						wc.Process(w, rest2, true, false)
					}
					ct.WriteNewline(w)
				}
				if !handled {
					wc.Process(w, line, true, false)
					ct.WriteNewline(w)
				}
			}
			w.Flush()
		}
	})

	// Shell-out to ccze-go
	goShellResult := testing.Benchmark(func(b *testing.B) {
		cmd := exec.Command(goBin, "-A")
		stdin, _ := cmd.StdinPipe()
		cmd.Stdout = io.Discard
		cmd.Start()
		for i := 0; i < b.N; i++ {
			stdin.Write(input)
		}
		stdin.Close()
		cmd.Wait()
	})

	fmt.Printf("## Benchmark Results (%d lines)\n\n", len(lines))
	fmt.Printf("| Metric | In-process | ccze-go | ccze (C) |\n")
	fmt.Printf("|--------|------------|---------|----------|\n")
	fmt.Printf("| ns/op | %d | %d | ", inProcResult.NsPerOp(), goShellResult.NsPerOp())

	// Shell-out to C ccze
	cczePath, err := exec.LookPath("ccze")
	if err != nil {
		fmt.Printf("N/A |\n")
		return
	}

	cShellResult := testing.Benchmark(func(b *testing.B) {
		cmd := exec.Command(cczePath, "-A")
		stdin, _ := cmd.StdinPipe()
		cmd.Stdout = io.Discard
		cmd.Start()
		for i := 0; i < b.N; i++ {
			stdin.Write(input)
		}
		stdin.Close()
		cmd.Wait()
	})

	fmt.Printf("%d |\n", cShellResult.NsPerOp())
	speedup := float64(cShellResult.NsPerOp()) / float64(goShellResult.NsPerOp())
	fmt.Printf("| **Speedup** | - | **%.1fx faster** | baseline |\n", speedup)
}
