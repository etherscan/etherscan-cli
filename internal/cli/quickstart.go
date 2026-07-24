package cli

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/etherscan/etherscan-cli/internal/brand"
)

// quickStartExample is one line of the Quick Start guide: a runnable command and
// a short description rendered as a dim "# comment".
type quickStartExample struct {
	cmd     string
	comment string
}

// quickStartExamples are the handful of commands a first-time user most likely
// wants. Every command here is real (see registry.go / root.go).
var quickStartExamples = []quickStartExample{
	{"etherscan login", "Set up your API key"},
	{"etherscan stats ethprice", "Get ETH price"},
	{"etherscan account balance 0x…", "Address balance"},
	{"etherscan gastracker oracle", "Gas prices"},
	{"etherscan tui", "Interactive explorer"},
	{"etherscan --help", "All commands"},
}

const (
	quickStartTitle = "Official API Command Line Interface"
	quickStartDocs  = "Docs: https://docs.etherscan.io"
)

// renderQuickStart builds the branded Quick Start banner: the Etherscan block-art
// wordmark above a rounded box holding a title, an aligned list of example
// commands, and a docs link. When color is true it is painted with the brand
// accent (blue) and dim (grey) using truecolor ANSI; when false it is plain text
// with no escape sequences, so piped or CI output stays clean. width is the
// terminal width (<=0 means unknown): the wordmark falls back to a one-line
// "Etherscan" when it would not fit, and the box is capped to the terminal width.
func renderQuickStart(info BuildInfo, color bool, width int) string {
	paint := func(hex, s string, bold bool) string {
		if !color {
			return s
		}
		return ansiColor(hex, s, bold)
	}

	logoWidth := 0
	for _, line := range brand.Logo {
		if w := utf8.RuneCountInString(line); w > logoWidth {
			logoWidth = w
		}
	}

	// Align the "# comment" column to the widest command.
	maxCmd := 0
	for _, e := range quickStartExamples {
		if w := utf8.RuneCountInString(e.cmd); w > maxCmd {
			maxCmd = w
		}
	}

	// Build the inner (boxed) content lines. divider entries are left empty here
	// and sized to innerWidth once it is known.
	const dividerMarker = "\x00divider\x00"
	lines := []string{paint(brand.AccentHex, quickStartTitle, true)}
	if info.Version != "" {
		lines = append(lines, paint(brand.DimHex, "Etherscan CLI "+info.Version, false))
	}
	lines = append(lines, dividerMarker, paint(brand.AccentHex, " Quick Start", true), "")
	for _, e := range quickStartExamples {
		pad := strings.Repeat(" ", maxCmd-utf8.RuneCountInString(e.cmd))
		line := "  " + paint(brand.DimHex, "$", false) + " " + e.cmd + pad + "  " + paint(brand.DimHex, "# "+e.comment, false)
		lines = append(lines, line)
	}
	lines = append(lines, dividerMarker, " "+paint(brand.DimHex, quickStartDocs, false))

	// innerWidth is the widest visible content line (dividers excluded), capped so
	// the whole box (content + 2 padding + 2 borders) fits the terminal.
	innerWidth := 0
	for _, line := range lines {
		if line == dividerMarker {
			continue
		}
		if w := visibleWidth(line); w > innerWidth {
			innerWidth = w
		}
	}
	if width > 0 && innerWidth > width-4 {
		innerWidth = width - 4
	}
	if innerWidth < 1 {
		innerWidth = 1
	}

	var b strings.Builder
	// Wordmark, above the box.
	if width <= 0 || width >= logoWidth {
		for _, line := range brand.Logo {
			b.WriteString(paint(brand.AccentHex, line, true))
			b.WriteByte('\n')
		}
	} else {
		b.WriteString(paint(brand.AccentHex, "Etherscan", true))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	// Rounded box around the content.
	horizontal := strings.Repeat("─", innerWidth+2)
	b.WriteString(paint(brand.DimHex, "╭"+horizontal+"╮", false))
	b.WriteByte('\n')
	for _, line := range lines {
		content := line
		if line == dividerMarker {
			content = paint(brand.DimHex, strings.Repeat("─", innerWidth), false)
		}
		gap := innerWidth - visibleWidth(content)
		if gap < 0 {
			gap = 0
		}
		b.WriteString(paint(brand.DimHex, "│", false))
		b.WriteByte(' ')
		b.WriteString(content)
		b.WriteString(strings.Repeat(" ", gap))
		b.WriteByte(' ')
		b.WriteString(paint(brand.DimHex, "│", false))
		b.WriteByte('\n')
	}
	b.WriteString(paint(brand.DimHex, "╰"+horizontal+"╯", false))

	return b.String()
}

// ansiEscape matches SGR colour sequences so visibleWidth can measure the
// on-screen width of a coloured string.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// visibleWidth returns the number of terminal cells a string occupies, ignoring
// ANSI colour escapes. The banner's glyphs (box-drawing, half-blocks, "…") are
// all single-width, so a rune count of the stripped string is accurate.
func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiEscape.ReplaceAllString(s, ""))
}

// ansiColor wraps s in a truecolor foreground escape (optionally bold). hex is a
// "#rrggbb" string; an unparseable value falls back to no color.
func ansiColor(hex, s string, bold bool) string {
	r, g, b, ok := hexRGB(hex)
	if !ok {
		return s
	}
	prefix := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	if bold {
		prefix = "\x1b[1m" + prefix
	}
	return prefix + s + "\x1b[0m"
}

func hexRGB(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}
