package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ANSI styles (no external color deps).
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	cyan    = "\033[36m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	red     = "\033[31m"
	white   = "\033[37m"
)

// colorize wraps s when stdout is a TTY.
func colorize(enabled bool, code, s string) string {
	if !enabled {
		return s
	}
	return code + s + reset
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// ATLAS wordmark — block-style Unicode for terminals that support it.
const atlasBanner = `
  █████╗ ████████╗██╗      █████╗ ███████╗
 ██╔══██╗╚══██╔══╝██║     ██╔══██╗██╔════╝
 ███████║   ██║   ██║     ███████║███████╗
 ██╔══██║   ██║   ██║     ██╔══██║╚════██║
 ██║  ██║   ██║   ███████╗██║  ██║███████║
 ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚══════╝
`

func printBanner(w io.Writer, baseURL, model string, color bool) {
	line := colorize(color, dim+cyan, "  ──────────────────────────────────────────────────")
	fmt.Fprintln(w, line)

	// Gradient-ish: alternate cyan/blue/magenta across banner lines.
	codes := []string{cyan, blue, magenta, cyan, blue, magenta}
	for i, raw := range strings.Split(strings.TrimRight(atlasBanner, "\n"), "\n") {
		if strings.TrimSpace(raw) == "" {
			fmt.Fprintln(w)
			continue
		}
		code := codes[i%len(codes)]
		fmt.Fprintln(w, colorize(color, bold+code, raw))
	}

	tagline := colorize(color, dim+white, "  AI Agent Platform  ·  local inference console")
	fmt.Fprintln(w, tagline)
	fmt.Fprintln(w, line)

	meta := fmt.Sprintf("  %s  %s    %s  %s",
		colorize(color, dim, "gateway"),
		colorize(color, cyan, baseURL),
		colorize(color, dim, "model"),
		colorize(color, magenta, model),
	)
	fmt.Fprintln(w, meta)
	fmt.Fprintln(w, line)
	fmt.Fprintln(w)
}
