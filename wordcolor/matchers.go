// Package wordcolor — matchers.go.
//
// Hand-written scanners for host/IP and email/message-id tokens, split out of
// wordcolor.go to keep files focused and under the toolchain's size limit.
// These replace the original C regexes (regHost, regHostIP, regEmail,
// regEmail2, regMsgID) and are part of the default C-compatible pipeline.
package wordcolor

import "strings"

// ---------------------------------------------------------------------------
// Host / IP matching  (replaces regHost and regHostIP)
// ---------------------------------------------------------------------------

// matchIPv4: \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}
// Exactly 4 dot-separated groups of 1-3 digits. Scans in place: this runs
// for every word (via matchHost, the first cascade entry), so it must not
// allocate like the previous strings.Split version did.
func matchIPv4(s string) bool {
	parts := 0
	i := 0
	n := len(s)
	for {
		start := i
		for i < n && isDigit(s[i]) {
			i++
		}
		if i-start < 1 || i-start > 3 {
			return false
		}
		parts++
		if i == n {
			return parts == 4
		}
		if s[i] != '.' || parts == 4 {
			return false
		}
		i++
	}
}

// matchHostname: ([a-z0-9-_]+\.)+[a-z]{2,3}
// Labels of [a-z0-9-_]+ separated by dots, ending in a 2-3 letter TLD.
// Scans in place (no strings.Split) - it runs for every word via matchHost.
func matchHostname(s string) bool {
	idx := strings.LastIndexByte(s, '.')
	if idx < 1 {
		return false
	}
	tld := s[idx+1:]
	if len(tld) < 2 || len(tld) > 3 {
		return false
	}
	for i := 0; i < len(tld); i++ {
		if !isLowerAlpha(tld[i]) {
			return false
		}
	}
	// Dot-separated labels before the TLD: each non-empty, all [a-z0-9-_].
	labelStart := 0
	for i := 0; i < idx; i++ {
		c := s[i]
		if c == '.' {
			if i == labelStart {
				return false // empty label
			}
			labelStart = i + 1
			continue
		}
		if !isLowerAlnum(c) && c != '-' && c != '_' {
			return false
		}
	}
	return labelStart < idx // final label before the TLD dot is non-empty
}

// matchIPv6Like: (\w*::\w+)+
func matchIPv6Like(s string) bool {
	if !strings.Contains(s, "::") {
		return false
	}
	i := 0
	matched := false
	for i < len(s) {
		// \w*
		for i < len(s) && isWordChar(s[i]) {
			i++
		}
		// ::
		if i+1 >= len(s) || s[i] != ':' || s[i+1] != ':' {
			return matched && i == len(s)
		}
		i += 2
		// \w+ (at least one)
		start := i
		for i < len(s) && isWordChar(s[i]) {
			i++
		}
		if i == start {
			return false
		}
		matched = true
	}
	return matched
}

// matchHostCore checks if s (without port) is a valid host.
func matchHostCore(s string) bool {
	if s == "localhost" {
		return true
	}
	return matchIPv4(s) || matchHostname(s) || matchIPv6Like(s)
}

// matchHost: the full regHost pattern, with optional :port suffix.
// ^(((IPv4)|(hostname)|(localhost)|(IPv6))(:\d{1,5})?)$
func matchHost(s string) bool {
	if len(s) == 0 {
		return false
	}
	if matchHostCore(s) {
		return true
	}
	// Try splitting off a trailing :port
	idx := strings.LastIndex(s, ":")
	if idx <= 0 || s[idx-1] == ':' {
		return false
	}
	port := s[idx+1:]
	if len(port) < 1 || len(port) > 5 || !allDigits(port) {
		return false
	}
	return matchHostCore(s[:idx])
}

// matchHostIPCore is like matchHostCore but with the looser hostname
// pattern from regHostIP: ([a-z0-9-_.]+)+ instead of requiring a TLD.
func matchHostIPCore(s string) bool {
	if s == "localhost" || matchIPv4(s) || matchIPv6Like(s) {
		return true
	}
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !isLowerAlnum(c) && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// matchHostIP: like regHostIP — host pattern followed by '['.
func matchHostIP(s string) bool {
	idx := strings.Index(s, "[")
	if idx <= 0 {
		return false
	}
	host := s[:idx]
	if matchHostIPCore(host) {
		return true
	}
	// Try with port
	cidx := strings.LastIndex(host, ":")
	if cidx > 0 && host[cidx-1] != ':' {
		port := host[cidx+1:]
		if len(port) >= 1 && len(port) <= 5 && allDigits(port) {
			return matchHostIPCore(host[:cidx])
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Email / MsgID matching  (replaces regEmail, regEmail2, regMsgID)
// ---------------------------------------------------------------------------

// matchEmail: ^[a-z0-9-_=+]+@([a-z0-9-_.]+)+  (prefix match)
func matchEmail(s string) bool {
	atIdx := strings.Index(s, "@")
	if atIdx < 1 {
		return false
	}
	for i := 0; i < atIdx; i++ {
		c := s[i]
		if !isLowerAlnum(c) && c != '-' && c != '_' && c != '=' && c != '+' {
			return false
		}
	}
	domain := s[atIdx+1:]
	if len(domain) == 0 {
		return false
	}
	for i := 0; i < len(domain); i++ {
		c := domain[i]
		if !isLowerAlnum(c) && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// matchEmail2: (\.[a-z]{2,4})+$
func matchEmail2(s string) bool {
	i := len(s)
	matched := false
	for i > 0 {
		// Scan backwards for .tld segment
		j := i - 1
		for j >= 0 && s[j] != '.' {
			j--
		}
		if j < 0 {
			break
		}
		tld := s[j+1 : i]
		if len(tld) < 2 || len(tld) > 4 {
			break
		}
		ok := true
		for k := 0; k < len(tld); k++ {
			if !isLowerAlpha(tld[k]) {
				ok = false
				break
			}
		}
		if !ok {
			break
		}
		matched = true
		i = j
	}
	return matched
}

// matchMsgID: ^[a-z0-9-_.$=+]+@([a-z0-9-_.]+)+(\.[a-z]+)+  (prefix match)
func matchMsgID(s string) bool {
	atIdx := strings.Index(s, "@")
	if atIdx < 1 {
		return false
	}
	for i := 0; i < atIdx; i++ {
		c := s[i]
		if !isLowerAlnum(c) && c != '-' && c != '_' && c != '.' && c != '$' && c != '=' && c != '+' {
			return false
		}
	}
	domain := s[atIdx+1:]
	if len(domain) == 0 {
		return false
	}
	for i := 0; i < len(domain); i++ {
		c := domain[i]
		if !isLowerAlnum(c) && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	// Must contain .[a-z]+ at end
	lastDot := strings.LastIndex(domain, ".")
	if lastDot < 0 || lastDot == len(domain)-1 {
		return false
	}
	tld := domain[lastDot+1:]
	for i := 0; i < len(tld); i++ {
		if !isLowerAlpha(tld[i]) {
			return false
		}
	}
	return true
}
