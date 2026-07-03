package wordcolor

import (
	"testing"

	"ccze-go/color"
	"github.com/stretchr/testify/assert"
)

// TestExtMakeFatal verifies the opt-in GNU make error/fatal highlighter: make
// failure lines (prefixed "make[N]: " / "make: " and marked with "*** ") are
// colored entirely as color.Error, while benign make lines and non-make lines
// are left alone. As with every extension, stripped(output) must equal input.
func TestExtMakeFatal(t *testing.T) {
	p, ct := procWith(Extensions{Make: true})
	errFg := fgSeq(ct, color.Error)

	// Lines that MUST be flagged as make errors (whole line -> Error color).
	fatal := []string{
		`make[1]: *** No rule to make target 'folder/file.cpp', needed by 'folder/file.o'.  Stop.`,
		`make[1]: *** Waiting for unfinished jobs....`,
		`make: *** [Makefile:10: all] Error 1`,
	}
	for _, in := range fatal {
		out := render(p, in)
		assert.Equal(t, in, stripAnsi(out), "make error text must be preserved verbatim: %q", in)
		assert.Contains(t, out, errFg, "make error line should use the Error (bold red) color: %q", in)
	}

	// Lines that MUST NOT be flagged: a benign make line (no "*** " marker), and
	// a non-make line that merely ends in "Stop." (no make prefix). These guard
	// against false positives.
	benign := []string{
		`make[1]: Entering directory '/tmp/ccze/src'`,
		`Please press Stop.`,
	}
	for _, in := range benign {
		out := render(p, in)
		assert.Equal(t, in, stripAnsi(out), "benign line text must be preserved: %q", in)
		assert.NotContains(t, out, errFg, "%q must not be flagged as a make error", in)
	}

	// With the Make extension OFF, even a genuine make error line is not
	// whole-line Error-colored (it falls through to the default per-word path).
	p2, ct2 := procWith(Extensions{})
	out2 := render(p2, `make[1]: *** No rule to make target 'folder/file.cpp', needed by 'folder/file.o'.  Stop.`)
	assert.NotContains(t, out2, fgSeq(ct2, color.Error), "Make-off must not flag make errors")
}

// TestIsMakeError exercises the marker matcher directly, covering both prefix
// forms and the "*** " requirement.
func TestIsMakeError(t *testing.T) {
	yes := []string{
		`make[1]: *** No rule to make target 'x', needed by 'y'.  Stop.`,
		`make[1]: *** Waiting for unfinished jobs....`,
		`make: *** [Makefile:10: all] Error 1`,
		`make[42]: *** anything`,
	}
	for _, in := range yes {
		assert.True(t, isMakeError(in), "expected make error: %q", in)
	}
	no := []string{
		``,
		`make[1]: Entering directory '/tmp/ccze/src'`, // make prefix but no "*** "
		`make: Nothing to be done for 'all'.`,         // make prefix but no "*** "
		`*** stars but no make prefix`,                // marker but not a make line
		`Please press Stop.`,                          // ends in Stop. but not make
		`makefile: *** not a make prefix`,             // "makefile:" is not "make:" / "make["
		`cmake[1]: *** subtly different prefix`,       // does not start with make
	}
	for _, in := range no {
		assert.False(t, isMakeError(in), "expected NOT a make error: %q", in)
	}
}
