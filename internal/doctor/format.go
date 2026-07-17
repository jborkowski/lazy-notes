package doctor

import (
	"fmt"
	"io"
	"strings"
)

// WriteReport prints a human-readable doctor report.
func WriteReport(w io.Writer, r Report) {
	width := 0
	for _, c := range r.Checks {
		if n := len(c.Name); n > width {
			width = n
		}
	}
	for _, c := range r.Checks {
		mark := statusMark(c.Severity)
		fmt.Fprintf(w, "%s  %-*s  %s\n", mark, width, c.Name, c.Detail)
		if c.Fix != "" && (c.Severity == Fail || c.Severity == Warn) {
			fmt.Fprintf(w, "      fix: %s\n", c.Fix)
		}
	}

	var fails, warns, oks, skips int
	for _, c := range r.Checks {
		switch c.Severity {
		case Fail:
			fails++
		case Warn:
			warns++
		case OK:
			oks++
		case Skip:
			skips++
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "summary: %d ok, %d warn, %d fail, %d skip\n", oks, warns, fails, skips)
	switch {
	case fails > 0:
		fmt.Fprintln(w, "result: FAIL — fix the items above, then re-run: lazy-notes doctor")
	case warns > 0:
		fmt.Fprintln(w, "result: WARN — usable, but review warnings")
	default:
		fmt.Fprintln(w, "result: OK — ready to sync")
	}
}

func statusMark(s Severity) string {
	switch s {
	case OK:
		return "[ok]  "
	case Warn:
		return "[warn]"
	case Fail:
		return "[fail]"
	case Skip:
		return "[skip]"
	default:
		return "[" + strings.ToLower(string(s)) + "]"
	}
}
