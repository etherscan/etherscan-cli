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

// quickStartGroup is a labelled block of examples. Grouping keeps the guide
// scannable as the example list grows: a first-time user reads the labels, not
// six near-identical command lines.
type quickStartGroup struct {
	label string
	items []quickStartExample
}

// quickStartGroups are the handful of commands a first-time user most likely
// wants, in the order they are usually needed. Every command here is real (see
// registry.go / root.go).
var quickStartGroups = []quickStartGroup{
	{"Setup", []quickStartExample{
		{"etherscan login", "Set up your API key"},
	}},
	{"Query", []quickStartExample{
		{"etherscan stats ethprice", "Get ETH price"},
		{"etherscan account balance 0x…", "Address balance"},
		{"etherscan gastracker oracle", "Gas prices"},
	}},
	{"Explore", []quickStartExample{
		{"etherscan tui", "Interactive explorer"},
		{"etherscan --help", "All commands"},
	}},
}

const (
	quickStartTitle  = "Official API Command Line Interface"
	quickStartDocs   = "https://docs.etherscan.io"
	quickStartBullet = "◆"

	// dividerMarker is a placeholder row: the horizontal rule cannot be built
	// until innerWidth is known, so build emits this and the render loop swaps in
	// a correctly sized line.
	dividerMarker = "\x00divider\x00"
)

// renderQuickStart builds the branded Quick Start banner: the Etherscan block-art
// wordmark above a rounded box holding a title, grouped example commands, and a
// docs link. When color is true it is painted with the brand accent (blue) and
// dim (grey) using truecolor ANSI; when false it is plain text with no escape
// sequences, so piped or CI output stays clean.
//
// width and height are the terminal size (<=0 means unknown, i.e. piped, in
// which case the full render is used so redirected output captures everything).
// The banner degrades in three steps to fit a short terminal: the wordmark
// collapses to a one-line "Etherscan", then the box's blank padding rows are
// dropped, then the group labels go. The airy box survives longer than the block
// art because the box is the part carrying information. Narrow terminals degrade
// on a separate axis: the "# comment" column goes first, and only below about 38
// columns — narrower than the shortest command row itself — are rows ellipsised
// to keep them inside the border.
func renderQuickStart(info BuildInfo, color bool, width, height int) string {
	paint := func(hex, s string, bold bool) string {
		if !color {
			return s
		}
		return ansiColor(hex, s, bold)
	}
	// bold emphasises the command text without tinting it, so it stays legible on
	// both light and dark terminal themes.
	bold := func(s string) string {
		if !color {
			return s
		}
		return "\x1b[1m" + s + "\x1b[0m"
	}

	logoWidth := 0
	for _, line := range brand.Logo {
		if w := utf8.RuneCountInString(line); w > logoWidth {
			logoWidth = w
		}
	}

	// Align the "# comment" column to the widest command across every group.
	maxCmd := 0
	for _, g := range quickStartGroups {
		for _, e := range g.items {
			if w := utf8.RuneCountInString(e.cmd); w > maxCmd {
				maxCmd = w
			}
		}
	}

	// build assembles the boxed content. pad controls the blank rows that give the
	// box its vertical rhythm and grouped controls the "Setup / Query / Explore"
	// labels; both are sacrificed, in that order, on a short terminal. commented
	// controls the "# comment" column, which a narrow terminal drops. Blank rows
	// need no special handling downstream because the render loop pads every row
	// out to innerWidth.
	build := func(pad, grouped, commented bool) []string {
		var lines []string
		blank := func() {
			if pad {
				lines = append(lines, "")
			}
		}

		blank()
		head := paint(brand.AccentHex, quickStartBullet, false) + " " + bold("Etherscan CLI")
		if info.Version != "" {
			head += " " + paint(brand.DimHex, "v"+info.Version, false)
		}
		lines = append(lines, " "+head, " "+paint(brand.AccentHex, quickStartTitle, true))
		blank()
		lines = append(lines, dividerMarker)
		blank()
		lines = append(lines, " "+paint(brand.AccentHex, "Quick Start", true), "")
		for i, g := range quickStartGroups {
			if grouped {
				if i > 0 {
					lines = append(lines, "")
				}
				lines = append(lines, "  "+paint(brand.DimHex, g.label, false))
			}
			for _, e := range g.items {
				row := "   " + paint(brand.AccentHex, "$", false) + " " + bold(e.cmd)
				if commented {
					gap := strings.Repeat(" ", maxCmd-utf8.RuneCountInString(e.cmd))
					row += gap + "  " + paint(brand.DimHex, "# "+e.comment, false)
				}
				lines = append(lines, row)
			}
		}
		blank()
		lines = append(lines, dividerMarker)
		blank()
		lines = append(lines, " "+paint(brand.DimHex, "Docs:", false)+" "+paint(brand.AccentHex, quickStartDocs, false))
		blank()
		return lines
	}

	// widest measures the widest visible content row. Divider placeholders are
	// skipped: their width is derived from this result, not the other way round.
	widest := func(lines []string) int {
		max := 0
		for _, line := range lines {
			if line == dividerMarker {
				continue
			}
			if w := visibleWidth(line); w > max {
				max = w
			}
		}
		return max
	}

	// The "# comment" column is what makes rows long, so a narrow terminal drops
	// it before anything gets truncated: the commands are the point of the guide
	// and the comments are a convenience. Dropping it changes no row count, so
	// this is decided before the height tiers below. Unknown width keeps it.
	commented := true
	if width > 0 && widest(build(true, true, true)) > width-4 {
		commented = false
	}

	fullLogo := width <= 0 || width >= logoWidth
	padRows, grouped := true, true
	if height > 0 {
		// Rows consumed: wordmark + the blank line under it + 2 box borders + the
		// content. Two more are reserved for the invoking command line above and
		// the shell prompt that returns below, so the top of the banner is not
		// already scrolled away by the time the user sees it.
		fits := func(logoRows int, pad, grouped bool) bool {
			return logoRows+1+2+len(build(pad, grouped, commented))+2 <= height
		}
		switch {
		case fullLogo && fits(len(brand.Logo), true, true):
			// Everything fits.
		case fits(1, true, true):
			fullLogo = false
		case fits(1, false, true):
			fullLogo, padRows = false, false
		default:
			// Smallest form; a terminal shorter than this simply scrolls. Height
			// degradation only ever drops decoration, never content — narrowing
			// is the axis that can reach the commands.
			fullLogo, padRows, grouped = false, false, false
		}
	}
	lines := build(padRows, grouped, commented)

	// innerWidth is the widest visible content line (dividers excluded), capped so
	// the whole box (content + 2 padding + 2 borders) fits the terminal.
	innerWidth := widest(lines)
	if width > 0 && innerWidth > width-4 {
		innerWidth = width - 4
	}
	if innerWidth < 1 {
		innerWidth = 1
	}

	var b strings.Builder
	// Wordmark, above the box.
	if fullLogo {
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
		} else if visibleWidth(content) > innerWidth {
			// Terminal narrower than even the comment-less rows: cut to the border
			// rather than punching through it.
			content = truncateVisible(content, innerWidth)
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
// ANSI colour escapes. The banner's glyphs (box-drawing, half-blocks, "…", "◆")
// are all single-width, so a rune count of the stripped string is accurate.
func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiEscape.ReplaceAllString(s, ""))
}

// truncateVisible shortens s to at most max terminal cells, marking the cut with
// an ellipsis. Colour escapes cost no cells, so they are copied through rather
// than counted, and a reset is appended when any were passed over so the cut
// cannot leak colour into the rest of the line.
//
// This is the backstop for the box invariant that every row is exactly
// innerWidth wide. The width tiers drop the "# comment" column first, but a
// terminal can always be narrower than even the shortest command.
func truncateVisible(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if visibleWidth(s) <= max {
		return s
	}

	escapes := ansiEscape.FindAllStringIndex(s, -1)
	var b strings.Builder
	sawEscape, visible := false, 0
	for i := 0; i < len(s); {
		if len(escapes) > 0 && escapes[0][0] == i {
			b.WriteString(s[i:escapes[0][1]])
			i = escapes[0][1]
			escapes = escapes[1:]
			sawEscape = true
			continue
		}
		// Leave one cell for the ellipsis.
		if visible == max-1 {
			break
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)
		i += size
		visible++
	}
	b.WriteString("…")
	if sawEscape {
		b.WriteString("\x1b[0m")
	}
	return b.String()
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
