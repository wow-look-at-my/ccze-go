// Package wordcolor — extensions.go.
//
// This file holds the opt-in "modern log" highlighters (tags, files, slog
// key=value, durations; the Unreal and make highlighters live in unreal.go and
// make.go) and their plumbing. They are split out from wordcolor.go to keep
// each file focused (and under the toolchain's file-size limit). Every helper here is reached only when its Extensions flag is set;
// the default pipeline never touches them and stays byte-for-byte compatible
// with C ccze. All of them only ADD color — the visible text is unchanged.
package wordcolor

import (
	"io"
	"strings"

	"ccze-go/color"
)

// The two simplest, opt-in boolean matchers used here (matchDuration and
// isSlogKey) are compiled from regexes by go-regex-compiler into DFA
// byte-scanners (duration_match_gen.go, slogkey_match_gen.go) instead of being
// hand-written, keeping the pattern declarative. They are off the default hot
// path, so the regex form costs nothing on the C-compatible benchmark. The
// generated files are committed as vendored source; regenerate them with:
//
//	go install github.com/wow-look-at-my/go-regex-compiler/cmd/go-regex-compiler@latest
//	go-regex-compiler --regex '[+-]?([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+' \
//	  --func matchDuration --package wordcolor --match full --output wordcolor/duration_match_gen.go
//	go-regex-compiler --regex '[A-Za-z][A-Za-z0-9_.-]*' \
//	  --func isSlogKey --package wordcolor --match full --output wordcolor/slogkey_match_gen.go
//	gofmt -w wordcolor/duration_match_gen.go wordcolor/slogkey_match_gen.go
//
// (A //go:generate directive is intentionally NOT used: go-toolchain gates
// go:generate behind an approval hash and would also need the generator
// installed in CI. Vendoring the output avoids that CI coupling. The
// host/IP/email/number/etc. matchers stay hand-written: they are on the hot
// path and a few need field extraction, which the generator does not do — see
// matchers.go.)

// Extensions toggles the opt-in "modern log" highlighters. They are all off by
// default so the standard pipeline stays byte-for-byte compatible with the C
// ccze it ports (the golden/compat test suite depends on this). They are
// enabled via the -o CLI option (see main.go) and only ever ADD color: every
// input byte is still emitted verbatim, just wrapped in extra ANSI.
type Extensions struct {
	Tags      bool // [INFO]/[component] bracket tags, level-aware coloring
	Files     bool // filenames and relative paths (foo.go, src/app.py, ./x)
	Slog      bool // logfmt / slog key=value structured pairs
	Durations bool // Go-style durations: 30.2s, 1m30s, 100ms, 1.5h
	Unreal    bool // Unreal Engine log lines: [time][frame]Category: Verbosity:
	Adaptive  bool // adaptive recurring-structure recognition (see adaptive.go)
	Make      bool // GNU make error/fatal lines: "make[N]: *** ...  Stop." (see make.go)
}

// any reports whether at least one extension is enabled. When false, ProcessOne
// takes exactly the same path as the original C-compatible implementation.
func (e Extensions) any() bool {
	return e.Tags || e.Files || e.Slog || e.Durations || e.Unreal || e.Adaptive || e.Make
}

// SetExtensions enables the opt-in highlighters on the processor. It (re)builds
// any per-extension state, so it is safe to call once after construction.
func (p *Processor) SetExtensions(e Extensions) {
	p.ext = e
	if e.Adaptive {
		p.ada = newAdaptive()
	} else {
		p.ada = nil
	}
}

// matchDuration (Go-style durations: 30.2s, 1m30s, 100ms, 1.5h, 500us, 300µs,
// 2h45m, -5m) is generated from its regex into duration_match_gen.go by
// go-regex-compiler (see the //go:generate directives above).

// fileExtensions is a curated set of unambiguous file extensions used only for
// BARE names (no slash). It deliberately omits strings that are common domain
// TLDs (com, org, net, io, co, dev, ...) so that hostnames are not mistaken for
// files. Paths that contain a slash use the looser fileExt check instead.
var fileExtensions = map[string]bool{
	"go": true, "mod": true, "sum": true, "txt": true, "log": true, "json": true,
	"yaml": true, "yml": true, "toml": true, "ini": true, "conf": true, "cfg": true,
	"md": true, "rst": true, "csv": true, "tsv": true, "xml": true, "html": true,
	"htm": true, "css": true, "scss": true, "js": true, "jsx": true, "ts": true,
	"tsx": true, "py": true, "rb": true, "rs": true, "c": true, "cc": true,
	"cpp": true, "cxx": true, "h": true, "hpp": true, "java": true, "kt": true,
	"php": true, "pl": true, "lua": true, "sql": true, "sh": true, "bash": true,
	"zsh": true, "fish": true, "pem": true, "crt": true, "cert": true, "cer": true,
	"der": true, "csr": true, "pub": true, "key": true, "p12": true, "pfx": true,
	"jks": true, "pid": true, "sock": true, "lock": true, "gz": true, "bz2": true,
	"xz": true, "zst": true, "tar": true, "tgz": true, "zip": true, "7z": true,
	"rar": true, "png": true, "jpg": true, "jpeg": true, "gif": true, "svg": true,
	"ico": true, "webp": true, "bmp": true, "pdf": true, "db": true, "sqlite": true,
	"sqlite3": true, "bak": true, "tmp": true, "swp": true, "out": true, "err": true,
	"o": true, "a": true, "so": true, "dll": true, "exe": true, "bin": true,
	"deb": true, "rpm": true, "apk": true, "properties": true, "env": true,
	"service": true, "socket": true, "target": true, "tf": true, "tfvars": true,
}

// baseName returns the final path component of s.
func baseName(s string) string {
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// fileExt returns the (lowercase) extension of a basename when it has a
// plausible one — a '.' that is neither the first nor the last byte, followed
// by 1-8 [a-z0-9] characters — otherwise "". This is the loose check used for
// slash-bearing paths, where the slash already signals "this is a path".
func fileExt(base string) string {
	dot := strings.LastIndexByte(base, '.')
	if dot <= 0 || dot == len(base)-1 {
		return ""
	}
	ext := base[dot+1:]
	if len(ext) > 8 {
		return ""
	}
	for i := 0; i < len(ext); i++ {
		if !isLowerAlnum(ext[i]) {
			return ""
		}
	}
	return ext
}

// looksLikePath reports whether s is clearly a filesystem path: it starts with
// /, ./, ../ or ~/, or it is a relative path (contains a slash) whose basename
// carries a plausible extension. This avoids treating tokens like "and/or" as
// paths while still catching "src/app/main.go".
func looksLikePath(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '/' {
		return true
	}
	if strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~/") {
		return true
	}
	if i := strings.IndexByte(s, '/'); i > 0 {
		return fileExt(baseName(s)) != ""
	}
	return false
}

// isBareFile reports whether lword is a bare filename (no slash) whose
// extension is in the curated unambiguous set. Used to color e.g. main.go or
// config.yaml as a file before the host matcher would claim them.
func isBareFile(lword string) bool {
	if lword == "" || strings.IndexByte(lword, '/') >= 0 {
		return false
	}
	return fileExtensions[fileExt(lword)]
}

// isSlogKey reports whether s is a plausible logfmt/slog key: a leading letter
// then [A-Za-z0-9_.-] (which keeps CLI flags like --x=y and numeric junk from
// being treated as keys). It is generated from its regex into
// slogkey_match_gen.go by go-regex-compiler (see the //go:generate directives
// above).

// tagColor maps a bracketed tag's inner text to a color. Common log levels get
// semantic colors; anything else gets the keyword color.
func tagColor(inner string) color.Color {
	switch strings.ToUpper(inner) {
	case "ERROR", "ERR", "FATAL", "CRIT", "CRITICAL", "PANIC", "ALERT", "EMERG", "EMERGENCY":
		return color.Error
	case "WARN", "WARNING":
		return color.Warning
	case "INFO", "INFORMATION", "NOTICE":
		return color.GoodWord
	case "DEBUG", "TRACE", "VERBOSE", "FINE":
		return color.Debug
	default:
		return color.Keyword
	}
}

// classifyValue returns the color for a slog value token by running it through
// the inline type matchers (durations, host, ip, size, version, time, number,
// ...). It returns color.Default when nothing matches. lword must be the
// lowercased value.
func (p *Processor) classifyValue(lword string) color.Color {
	switch {
	case lword == "":
		return color.Default
	case p.ext.Durations && matchDuration(lword):
		return color.GetTime
	case p.ext.Files && isBareFile(lword):
		// e.g. file=main.go — prefer File over the hostname syntax it also
		// matches. Gated on Files so it only applies when that extension is on.
		return color.File
	case matchHost(lword):
		return color.Host
	case matchMAC(lword):
		return color.MAC
	case looksLikePath(lword):
		if fileExt(baseName(lword)) != "" {
			return color.File
		}
		return color.Dir
	case matchEmail(lword) && matchEmail2(lword):
		return color.Email
	case matchURI(lword):
		return color.URI
	case matchSize(lword):
		return color.Size
	case matchVer(lword):
		return color.Version
	case matchTime(lword):
		return color.Date
	case matchAddr(lword):
		return color.Address
	case matchNum(lword):
		return color.Numbers
	default:
		return color.Default
	}
}

// renderTag emits a bracketed tag: the bytes of pre before its trailing '[',
// then a colored "[", the level-colored inner text, a colored "]", and the
// bytes of post after its leading ']'. Caller guarantees pre ends with '[' and
// post starts with ']'. Concatenated output text equals pre+inner+post.
func (p *Processor) renderTag(w io.Writer, pre, inner, post string) {
	p.ct.WriteColored(w, color.Default, pre[:len(pre)-1])
	p.ct.WriteColored(w, color.PIDB, "[")
	p.ct.WriteColored(w, tagColor(inner), inner)
	p.ct.WriteColored(w, color.PIDB, "]")
	p.ct.WriteColored(w, color.Default, post[1:])
}

// counterParts reports whether word is a "N/M" counter (e.g. "22/43") and
// returns its two digit runs. The brackets have already been split into
// pre/post by the caller, so word is just the inner "N/M" text. Because
// IndexByte finds the FIRST slash, a second slash (e.g. "22/4/3") lands in b
// and fails allDigits(b), so multi-slash inputs correctly return false. Uses a
// manual byte scan (via allDigits) rather than regexp: this is a hot per-word
// path. allDigits is the existing helper in wordcolor.go.
func counterParts(word string) (a, b string, ok bool) {
	slash := strings.IndexByte(word, '/')
	if slash <= 0 || slash >= len(word)-1 {
		return "", "", false
	}
	a, b = word[:slash], word[slash+1:]
	if !allDigits(a) || !allDigits(b) {
		return "", "", false
	}
	return a, b, true
}

// renderCounter emits a bracketed "N/M" counter, mirroring renderTag: the bytes
// of pre before its trailing '[', then a bracket-colored "[", the numbers-colored
// first digit run, a bracket-colored "/", the numbers-colored second digit run,
// a bracket-colored "]", and the bytes of post after its leading ']'. Caller
// guarantees pre ends with '[' and post starts with ']'. Every input byte is
// emitted exactly once, so concatenated output text equals pre+a+"/"+b+post.
func (p *Processor) renderCounter(w io.Writer, pre, a, b, post string) {
	p.ct.WriteColored(w, color.Default, pre[:len(pre)-1])
	p.ct.WriteColored(w, color.PIDB, "[")
	p.ct.WriteColored(w, color.Numbers, a)
	p.ct.WriteColored(w, color.PIDB, "/")
	p.ct.WriteColored(w, color.Numbers, b)
	p.ct.WriteColored(w, color.PIDB, "]")
	p.ct.WriteColored(w, color.Default, post[1:])
}

// renderKeyValue handles a slog/logfmt token of the form key=value. It returns
// false (writing nothing) when the token is not a key=value pair, so the caller
// can fall through to normal processing. The key is colored as a field, '=' is
// neutral, and the value is type-classified. Output text equals pre+word+post.
func (p *Processor) renderKeyValue(w io.Writer, pre, word, post string) bool {
	eq := strings.IndexByte(word, '=')
	if eq <= 0 {
		return false
	}
	// Skip '==' (base64 padding, equality tests) — not a kv separator.
	if eq+1 < len(word) && word[eq+1] == '=' {
		return false
	}
	key := word[:eq]
	if !isSlogKey(key) {
		return false
	}
	val := word[eq+1:]

	p.ct.WriteColored(w, color.Default, pre)
	p.ct.WriteColored(w, color.Field, key)
	p.ct.WriteColored(w, color.Default, "=")
	p.renderValue(w, val)
	p.ct.WriteColored(w, color.Default, post)
	return true
}

// renderValue colors a single slog value, honoring surrounding matched quotes.
// Output text always equals val.
func (p *Processor) renderValue(w io.Writer, val string) {
	if val == "" {
		return
	}
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
		q := val[:1]
		inner := val[1 : len(val)-1]
		p.ct.WriteColored(w, color.Default, q)
		p.ct.WriteColored(w, p.classifyValue(strings.ToLower(inner)), inner)
		p.ct.WriteColored(w, color.Default, q)
		return
	}
	p.ct.WriteColored(w, p.classifyValue(strings.ToLower(val)), val)
}
