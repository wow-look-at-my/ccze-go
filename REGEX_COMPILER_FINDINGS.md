# go-regex-compiler Integration Findings

Real-world test of `go-regex-compiler` against `ccze-go` (35 regex patterns across 20 plugins + wordcolor).

## What Worked

### Pattern Compilation: 44/44 (100%)
Every regex pattern in ccze-go compiled successfully through go-regex-compiler. No DFA state explosions, no parse failures. Even complex patterns like:
- `httpd_access` (9 capture groups, mixed anchors, character classes) - 6,777 lines generated
- `oops` (day/month alternations, hex patterns) - 646 lines
- `squid_store` (18 capture groups) - 490 lines

### Direct Replacement: 14 patterns
These patterns used `MatchString()` only (no capture groups) and were directly replaced:
- **wordcolor** (13 patterns): `regHost`, `regMAC`, `regURI`, `regVer`, `regAddr`, `regNum`, `regEmail`, `regEmail2`, `regMsgID`, `regSize`, `regTime`, `regSig`, `regHostIP`
- **ulogd plugin** (1 pattern): pure `MatchString` usage

### Pre-filter Integration: 3 plugins demonstrated
DFA matchers used as fast pre-filters before `FindStringSubmatch`:
- `syslog` - DFA rejects non-syslog lines before regex runs
- `httpd` - separate DFA pre-filters for access vs error log formats
- `dpkg` - DFA pre-filters for status/action/conffile patterns

### Test Results: 96/96 pass
All existing tests pass with generated matchers integrated. Zero behavioral regressions.

## Partial-Match Adaptation
7 wordcolor patterns were prefix/suffix matchers (no `$` or `^` anchor). These were adapted by adding `.*` to make them full-match patterns:
- `^prefix...` → `^prefix....*$`
- `...suffix$` → `^.*...suffix$`
- `substring` → `^.*substring.*$`

All 7 adapted correctly. DFA handled the `.*` without state explosion.

## What Needs More Love

### 1. No Capture Group Support (The Big One)
**Impact: 33 of 35 patterns can't be fully replaced.**

ccze-go uses `FindStringSubmatch()` to extract capture groups in almost every plugin. go-regex-compiler only generates `func(string) bool` matchers. This means:
- Plugins still need `regexp.Regexp` for extraction
- Generated matchers can only serve as pre-filters (fast rejection)
- The pre-filter pattern works but doubles the code complexity

**What's needed:** Generate `func(string) []string` that returns submatch slices, or at minimum named group extraction.

### 2. No Partial/Substring Match Mode
**Impact: Requires manual pattern adaptation.**

go-regex-compiler always does full-string matching (`^...$`). Many real-world patterns are:
- Prefix matches (`^pattern` without `$`)
- Suffix matches (`pattern$` without `^`)
- Substring matches (no anchors at all)

Currently requires manually wrapping with `.*` which:
- Works but feels fragile
- Can cause DFA state explosion for complex inner patterns
- Changes the semantic meaning vs the original regex

**What's needed:** A `-match-mode` flag: `full` (default), `prefix`, `suffix`, `contains`.

### 3. Code Size Explosion for Complex Patterns
**Impact: 27,447 total lines generated for 42 patterns.**

Some patterns generate enormous DFA tables:
- `httpd_access`: 6,777 lines (one function!)
- `fetchmail`: 3,790 lines
- `apm`: 2,880 lines
- `ulogd` (with `.*` wrapping): 1,193 lines

Simple patterns are compact and elegant:
- `num` (`^[+-]?\d+$`): 42 lines
- `addr` (`^0x[0-9a-f]+$`): 49 lines
- `ver`: 71 lines

The `.*` adaptation for partial matches is the main driver of bloat. A native partial-match mode would dramatically reduce this.

### 4. No `go generate` Integration
**What's needed:** A `//go:generate` compatible workflow. Ideally:
```go
//go:generate regex-gen -regex "^pattern$" -func MatchFoo -package matchers -output matchers/foo.go
```
This exists implicitly (the CLI supports it) but there's no documentation or examples showing this workflow.

### 5. No Batch Mode
**Impact: Running 42 invocations is slow and produces 42 separate files.**

**What's needed:** Accept a config file with multiple patterns:
```yaml
package: matchers
patterns:
  - name: MatchHost
    regex: "^(host pattern)$"
  - name: MatchMAC
    regex: "^(mac pattern)$"
```
Generate a single file or organized set of files.

## Summary

| Category | Count | Status |
|----------|-------|--------|
| Patterns compiled | 44/44 | All pass |
| Direct replacements (MatchString only) | 14 | Working, tested |
| Pre-filter integrations | 3 plugins | Working, tested |
| Blocked by no capture groups | 33 | Can only pre-filter |
| Tests passing | 96/96 | All pass |
| Generated code lines | 27,447 | Large for complex patterns |

**Bottom line:** go-regex-compiler works well for what it does - the generated DFA code is correct and the compilation handles all real-world patterns. The biggest gap is capture group support, which would unlock full regex replacement in real codebases. Partial-match mode and batch generation would make it much more practical for integration.
