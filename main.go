package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"ccze-go/color"
	"ccze-go/plugin"
	"ccze-go/wordcolor"
)

func main() {
	// CLI flags
	pluginsFlag := flag.String("p", "", "comma-separated plugin names to load (empty = all)")
	colorFlag := flag.String("c", "", "comma-separated color overrides like key=boldred")
	remfac := flag.Bool("r", false, "remove syslog-ng facility prefix <N> from start of lines")
	convdate := flag.Bool("C", false, "convert unix timestamps to human-readable dates")
	rcfile := flag.String("F", "", "config file path (overrides default loading)")
	listPlugins := flag.Bool("l", false, "list available plugins and exit")
	_ = flag.Bool("A", false, "ANSI output (default, kept for compat)")
	optionsFlag := flag.String("o", "", "comma-separated options: scroll,noscroll,wordcolor,nowordcolor,lookups,nolookups,transparent,notransparent; modern-log highlighters (opt-in): modern,tags,files,slog,durations,adaptive (and no- variants). Defaults can be set via the CCZE_OPTIONS env var (e.g. CCZE_OPTIONS=modern); -o overrides it.")
	flag.Parse()

	// Parse options. CCZE_OPTIONS provides the baseline (so the default mode can
	// be set once in the environment, e.g. CCZE_OPTIONS=modern); the -o flag is
	// then applied on top and can override it, including via the "no"-prefixed
	// variants (e.g. CCZE_OPTIONS=modern with -o noslog).
	transparent := true
	wcol := true
	slookup := true
	var ext wordcolor.Extensions
	applyOptions(os.Getenv("CCZE_OPTIONS"), &transparent, &wcol, &slookup, &ext)
	applyOptions(*optionsFlag, &transparent, &wcol, &slookup, &ext)

	// Create color table
	ct := color.NewTable(transparent)

	// Load config files
	if *rcfile != "" {
		ct.LoadFile(*rcfile)
	} else {
		ct.LoadFile("/etc/ccze/colorizerc")
		ct.LoadFile("/etc/ccze/cczerc")
		if home, err := os.UserHomeDir(); err == nil {
			ct.LoadFile(home + "/.colorizerc")
			ct.LoadFile(home + "/.cczerc")
		}
	}

	// Apply -c color overrides
	if *colorFlag != "" {
		for _, override := range strings.Split(*colorFlag, ",") {
			override = strings.TrimSpace(override)
			if override == "" {
				continue
			}
			// Convert key=boldred format to "key bold red" for ParseLine
			ct.ParseLine(convertColorOverride(override))
		}
	}

	// Create wordcolor processor
	wc := wordcolor.New(ct)
	wc.SetExtensions(ext)

	// Create buffered writer for stdout
	w := bufio.NewWriter(os.Stdout)

	// Create plugin registry and register all plugins
	registry := plugin.NewRegistry()
	registerAllPlugins(registry, w, ct, wc, *convdate)

	// Filter by -p if specified
	if *pluginsFlag != "" {
		names := strings.Split(*pluginsFlag, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		filterPlugins(registry, names)
	}

	// If -l, list plugins and exit
	if *listPlugins {
		listAllPlugins(registry)
		return
	}

	// Set up signal handler for SIGINT to print ESC[0m reset before exiting
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		<-sigCh
		fmt.Fprint(os.Stdout, "\x1b[0m")
		os.Stdout.Sync()
		os.Exit(0)
	}()

	// Main loop
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Remove facility prefix if requested
		if *remfac {
			if len(line) > 0 && line[0] == '<' {
				if idx := strings.Index(line, ">"); idx >= 0 {
					line = line[idx+1:]
				}
			}
		}

		handled, rest := registry.Run(line, plugin.TypeFull)

		if rest != "" {
			handled2, rest2 := registry.Run(rest, plugin.TypePartial)
			if !handled2 {
				wc.Process(w, rest, wcol, slookup)
			} else {
				wc.Process(w, rest2, wcol, slookup)
			}
			ct.WriteNewline(w)
		}

		if !handled {
			wc.Process(w, line, wcol, slookup)
			ct.WriteNewline(w)
		}

		w.Flush()
	}

	w.Flush()
}

// convertColorOverride converts a -c override string like "date=boldcyan" or
// "error=red" into a format suitable for Table.ParseLine: "date bold cyan".
func convertColorOverride(s string) string {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return s
	}
	key := s[:idx]
	val := s[idx+1:]

	// Try to parse attribute+color from the value
	var attr string
	lower := strings.ToLower(val)

	for _, a := range []string{"bold", "underline", "reverse", "blink"} {
		if strings.HasPrefix(lower, a) {
			attr = a
			val = val[len(a):]
			break
		}
	}

	// Check for on_bgcolor appended
	var bg string
	for _, bgName := range []string{"on_black", "on_red", "on_green", "on_yellow",
		"on_blue", "on_cyan", "on_magenta", "on_white"} {
		if strings.HasSuffix(strings.ToLower(val), bgName) {
			bg = bgName
			val = val[:len(val)-len(bgName)]
			break
		}
	}

	parts := []string{key}
	if attr != "" {
		parts = append(parts, attr)
	}
	if val != "" {
		parts = append(parts, strings.ToLower(val))
	}
	if bg != "" {
		parts = append(parts, bg)
	}
	return strings.Join(parts, " ")
}

// applyOptions parses a comma-separated option string (the same grammar as the
// -o flag and the CCZE_OPTIONS environment variable) and applies it on top of
// the current option state. Unknown tokens are ignored. Calling it more than
// once layers the option sources: later calls override earlier ones, which is
// how the -o flag overrides CCZE_OPTIONS.
func applyOptions(opts string, transparent, wcol, slookup *bool, ext *wordcolor.Extensions) {
	if opts == "" {
		return
	}
	for _, opt := range strings.Split(opts, ",") {
		switch strings.TrimSpace(opt) {
		case "scroll":
			// no-op in raw ansi mode
		case "noscroll":
			// no-op in raw ansi mode
		case "wordcolor":
			*wcol = true
		case "nowordcolor":
			*wcol = false
		case "lookups":
			*slookup = true
		case "nolookups":
			*slookup = false
		case "transparent":
			*transparent = true
		case "notransparent":
			*transparent = false

		// Opt-in "modern log" highlighters (off by default; default output
		// stays byte-for-byte compatible with C ccze).
		case "tags":
			ext.Tags = true
		case "notags":
			ext.Tags = false
		case "files":
			ext.Files = true
		case "nofiles":
			ext.Files = false
		case "slog":
			ext.Slog = true
		case "noslog":
			ext.Slog = false
		case "durations":
			ext.Durations = true
		case "nodurations":
			ext.Durations = false
		case "adaptive":
			ext.Adaptive = true
		case "noadaptive":
			ext.Adaptive = false
		case "modern":
			// Umbrella for the four stable highlighters (not adaptive, which is
			// still experimental and must be opted into by name).
			ext.Tags, ext.Files, ext.Slog, ext.Durations = true, true, true, true
		case "nomodern":
			ext.Tags, ext.Files, ext.Slog, ext.Durations = false, false, false, false
		}
	}
}

// registerAllPlugins registers all 20 plugins in alphabetical order,
// matching the C ccze's scandir/alphasort plugin loading behavior.
func registerAllPlugins(r *plugin.Registry, w *bufio.Writer, ct *color.Table, wc *wordcolor.Processor, convdate bool) {
	r.Register(plugin.NewApmPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewDistccPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewDpkgPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewEximPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewFetchmailPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewFtpstatsPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewHTTPDPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewIcecastPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewOopsPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewPHPPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewPostfixPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewProcmailPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewProFTPDPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewSquidPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewSulogPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewSuperPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewSyslogPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewUlogdPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewVsftpdPlugin(w, ct, wc, convdate))
	r.Register(plugin.NewXferlogPlugin(w, ct, wc, convdate))
}

// filterPlugins removes plugins not in the given name list from the registry.
func filterPlugins(r *plugin.Registry, names []string) {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	r.Filter(nameSet)
}

// listAllPlugins prints available plugins in the C-compatible format.
func listAllPlugins(r *plugin.Registry) {
	fmt.Println("Available plugins:")
	fmt.Println()
	fmt.Printf("%-10s| %-8s| %s\n", "Name", "Type", "Description")
	fmt.Println("------------------------------------------------------------")

	for _, p := range r.Plugins() {
		typeName := "Unknown"
		switch p.Type() {
		case plugin.TypeFull:
			typeName = "Full"
		case plugin.TypePartial:
			typeName = "Partial"
		}
		fmt.Printf("%-10s| %-8s| %s\n", p.Name(), typeName, p.Description())
	}
}
