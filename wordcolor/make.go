// Package wordcolor — make.go.
//
// This file holds the opt-in GNU make error/fatal highlighter. GNU make
// prefixes messages from a sub-make with "make[N]: " (and top-level make with
// "make: ") and marks every error and fatal condition with the "*** " marker.
// Matching that signature identifies the whole family of make failure lines,
// e.g.:
//
//	make[1]: *** No rule to make target 'x', needed by 'y'.  Stop.
//	make[1]: *** Waiting for unfinished jobs....
//	make: *** [Makefile:10: all] Error 1
//
// Like every extension here it only ADDS color: the line's bytes are emitted
// verbatim (just wrapped in the error color) so stripped output equals the
// input. It is gated behind Extensions.Make and is never reached on the default
// C-compatible path. The whole-line handler lives in wordcolor.go's Process.
package wordcolor

import "strings"

// isMakeError reports whether line is a GNU make error/fatal line. GNU make
// prefixes messages from a sub-make with "make[N]: " (and top-level make with
// "make: ") and marks every error and fatal condition with the "*** " marker.
// That signature identifies the whole family of make failure lines, e.g.:
//
//	make[1]: *** No rule to make target 'x', needed by 'y'.  Stop.
//	make[1]: *** Waiting for unfinished jobs....
//	make: *** [Makefile:10: all] Error 1
func isMakeError(line string) bool {
	if !strings.HasPrefix(line, "make:") && !strings.HasPrefix(line, "make[") {
		return false
	}
	return strings.Contains(line, "*** ")
}
