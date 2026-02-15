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
	"os"
	"os/exec"
	"strings"
	"testing"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
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

func BenchmarkVsCThroughput(b *testing.B) {
	data, err := os.ReadFile("testdata/mixed.log")
	if err != nil {
		b.Fatalf("failed to read testdata/mixed.log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	b.Run("Go", func(b *testing.B) {
		var totalBytes int64
		for _, l := range lines {
			totalBytes += int64(len(l))
		}

		b.ResetTimer()
		b.SetBytes(totalBytes)

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

	b.Run("C", func(b *testing.B) {
		ccze, err := exec.LookPath("ccze")
		if err != nil {
			b.Skip("C ccze binary not found; skipping. Install with: apt install ccze")
		}

		input := string(data)
		var totalBytes int64
		for _, l := range lines {
			totalBytes += int64(len(l))
		}

		b.ResetTimer()
		b.SetBytes(totalBytes)

		for i := 0; i < b.N; i++ {
			cmd := exec.Command(ccze, "-A")
			cmd.Stdin = strings.NewReader(input)
			cmd.Stdout = &bytes.Buffer{}
			if err := cmd.Run(); err != nil {
				b.Fatalf("C ccze failed: %v", err)
			}
		}
	})
}

// BenchmarkVsCLargeFile benchmarks processing a larger synthetic log file.
func BenchmarkVsCLargeFile(b *testing.B) {
	// Generate a large synthetic log: repeat the testdata lines
	data, err := os.ReadFile("testdata/mixed.log")
	if err != nil {
		b.Fatalf("failed to read testdata/mixed.log: %v", err)
	}
	baseLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	// Repeat to get ~1000 lines
	var largeLines []string
	for len(largeLines) < 1000 {
		largeLines = append(largeLines, baseLines...)
	}
	largeLines = largeLines[:1000]
	largeInput := strings.Join(largeLines, "\n") + "\n"

	var totalBytes int64
	for _, l := range largeLines {
		totalBytes += int64(len(l))
	}

	b.Run("Go_1000lines", func(b *testing.B) {
		b.SetBytes(totalBytes)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			ct := color.NewTable(true)
			wc := wordcolor.New(ct)
			r := plugin.NewRegistry()
			registerAllPlugins(r, w, ct, wc, false)

			for _, line := range largeLines {
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

	b.Run("C_1000lines", func(b *testing.B) {
		ccze, err := exec.LookPath("ccze")
		if err != nil {
			b.Skip("C ccze binary not found; skipping. Install with: apt install ccze")
		}

		b.SetBytes(totalBytes)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			cmd := exec.Command(ccze, "-A")
			cmd.Stdin = strings.NewReader(largeInput)
			cmd.Stdout = &bytes.Buffer{}
			if err := cmd.Run(); err != nil {
				b.Fatalf("C ccze failed: %v", err)
			}
		}
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
	if err != nil {
		t.Fatalf("failed to read testdata/mixed.log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	// Benchmark Go pipeline
	goResult := testing.Benchmark(func(b *testing.B) {
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

	goNsPerOp := goResult.NsPerOp()
	goLinesPerSec := float64(len(lines)) / (float64(goNsPerOp) / 1e9)

	fmt.Printf("## Benchmark Results\n\n")
	fmt.Printf("| Metric | Go | C |\n")
	fmt.Printf("|--------|-----|---|\n")
	fmt.Printf("| Lines processed | %d | %d |\n", len(lines), len(lines))
	fmt.Printf("| Go ns/iteration | %d | - |\n", goNsPerOp)
	fmt.Printf("| Go lines/sec | %.0f | ", goLinesPerSec)

	// Try C benchmark
	ccze, err := exec.LookPath("ccze")
	if err != nil {
		fmt.Printf("N/A (not installed) |\n")
		return
	}

	cResult := testing.Benchmark(func(b *testing.B) {
		input := string(data)
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(ccze, "-A")
			cmd.Stdin = strings.NewReader(input)
			cmd.Stdout = &bytes.Buffer{}
			cmd.Run()
		}
	})

	cNsPerOp := cResult.NsPerOp()
	cLinesPerSec := float64(len(lines)) / (float64(cNsPerOp) / 1e9)
	speedup := float64(cNsPerOp) / float64(goNsPerOp)

	fmt.Printf("%.0f |\n", cLinesPerSec)
	fmt.Printf("| C ns/iteration | - | %d |\n", cNsPerOp)
	fmt.Printf("| **Speedup** | **%.1fx faster** | baseline |\n", speedup)
}
