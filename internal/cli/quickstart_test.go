package cli

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/etherscan/etherscan-cli/internal/brand"
)

func TestRenderQuickStartPlain(t *testing.T) {
	out := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 100, 0)

	if strings.Contains(out, "\x1b[") {
		t.Fatalf("plain render must contain no ANSI escapes, got:\n%s", out)
	}

	for _, want := range []string{
		"Quick Start",
		"Official API Command Line Interface",
		"docs.etherscan.io",
		"◆ Etherscan CLI v1.2.3",
		"╭", "╰", "│", // rounded box glyphs (not ANSI)
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plain render missing %q", want)
		}
	}
	for _, g := range quickStartGroups {
		if !strings.Contains(out, g.label) {
			t.Errorf("plain render missing group label %q", g.label)
		}
		for _, e := range g.items {
			if !strings.Contains(out, e.cmd) {
				t.Errorf("plain render missing example command %q", e.cmd)
			}
			if !strings.Contains(out, e.comment) {
				t.Errorf("plain render missing comment %q", e.comment)
			}
		}
	}
}

func TestRenderQuickStartColored(t *testing.T) {
	out := renderQuickStart(BuildInfo{Version: "1.2.3"}, true, 100, 0)

	if !strings.Contains(out, "\x1b[") {
		t.Fatal("colored render should contain ANSI escapes")
	}
	// The accent (#5A8DEE -> 90,141,238) should be present as a truecolor escape.
	accent := "\x1b[38;2;90;141;238m"
	if !strings.Contains(out, accent) {
		t.Errorf("colored render missing brand accent escape, got:\n%q", out)
	}
	// The "$" sigil and the "◆" bullet carry the accent; the docs URL does too
	// while its "Docs:" label stays dim.
	for _, want := range []string{
		accent + "$" + "\x1b[0m",
		accent + quickStartBullet + "\x1b[0m",
		accent + quickStartDocs + "\x1b[0m",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("colored render missing accented %q", stripANSI(want))
		}
	}
	// Command text is bolded rather than tinted, so it stays readable on any theme.
	if !strings.Contains(out, "\x1b[1m"+"etherscan login"+"\x1b[0m") {
		t.Error("colored render should bold the example command text")
	}
	if !strings.Contains(out, "etherscan login") {
		t.Error("colored render missing example command")
	}
}

func TestRenderQuickStartNarrowFallback(t *testing.T) {
	// A width narrower than the full wordmark collapses to the one-line logo.
	out := renderQuickStart(BuildInfo{}, false, 10, 0)
	lines := strings.Split(out, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "Etherscan" {
		t.Errorf("narrow render should start with the one-line Etherscan wordmark, got first line %q", lines[0])
	}
	// The version is omitted entirely when BuildInfo carries none.
	if strings.Contains(out, "Etherscan CLI v") {
		t.Error("empty version should not render a version suffix")
	}
}

func TestRenderQuickStartRetainsLogoAtEveryHeight(t *testing.T) {
	firstLogoRow := brand.Logo[0]
	for _, height := range []int{10, 24, 30, 40, 60} {
		out := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 100, height)
		if !strings.HasPrefix(out, firstLogoRow+"\n") {
			t.Errorf("height %d replaced the full logo; first line = %q", height, strings.Split(out, "\n")[0])
		}
	}
}

func TestRenderSplashInteractiveSpacing(t *testing.T) {
	interactive := renderSplash(BuildInfo{}, true, 100, 40)
	if !strings.HasPrefix(interactive, "\n"+ansiColor(brand.AccentHex, brand.Logo[0], true)) {
		t.Errorf("interactive splash must start with one blank row before the logo, got %q", interactive[:min(len(interactive), 80)])
	}
	if strings.HasPrefix(interactive, "\n\n") {
		t.Fatal("interactive splash started with more than one blank row")
	}

	plain := renderSplash(BuildInfo{}, false, 100, 40)
	if strings.HasPrefix(plain, "\n") {
		t.Fatal("plain splash must not gain a leading blank row")
	}
	if strings.Contains(plain, "\x1b[") {
		t.Fatal("plain splash must not contain ANSI escapes")
	}
}

// When the terminal is too narrow for the full wordmark, the compact banner must
// not overflow an ordinary terminal. 24 rows is the conventional floor; below
// that the smallest form still exceeds the viewport and simply scrolls, which
// TestQuickStartDegradesMonotonically covers instead.
//
// Width 80 deliberately selects the one-line logo while leaving enough room for
// every Quick Start row; narrower box behavior is covered separately.
func TestQuickStartFitsTerminalHeight(t *testing.T) {
	for _, tc := range []struct{ w, h int }{{80, 24}, {80, 30}, {80, 40}, {80, 60}} {
		out := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, tc.w, tc.h)
		// +1 for the line the shell prompt returns on below the banner.
		if got := len(strings.Split(out, "\n")) + 1; got > tc.h {
			t.Errorf("render at %dx%d produced %d lines; want <= %d", tc.w, tc.h, got, tc.h)
		}
		// Degrading must never drop the actual content.
		for _, want := range []string{"Quick Start", "etherscan login", "docs.etherscan.io"} {
			if !strings.Contains(out, want) {
				t.Errorf("render at %dx%d dropped %q", tc.w, tc.h, want)
			}
		}
	}
}

// Shrinking the terminal must never make the banner taller, and every command
// survives all the way down to the smallest form.
func TestQuickStartDegradesMonotonically(t *testing.T) {
	prev := 0
	for h := 10; h <= 60; h++ {
		out := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 100, h)
		got := len(strings.Split(out, "\n"))
		if prev != 0 && got < prev {
			t.Errorf("height %d rendered %d lines, fewer than %d at height %d", h, got, prev, h-1)
		}
		prev = got
		for _, g := range quickStartGroups {
			for _, e := range g.items {
				if !strings.Contains(out, e.cmd) {
					t.Fatalf("height %d dropped command %q", h, e.cmd)
				}
			}
		}
	}
}

// An unknown size (piped output) always renders in full, so redirected output
// captures the whole guide rather than a terminal-shaped subset of it.
func TestQuickStartUnknownSizeRendersFull(t *testing.T) {
	out := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 0, 0)
	if got := len(strings.Split(out, "\n")); got < len(brand.Logo)+20 {
		t.Errorf("piped render collapsed to %d lines; want the full banner", got)
	}
}

func TestVisibleWidthIgnoresANSI(t *testing.T) {
	plain := "Quick Start"
	colored := ansiColor("#5A8DEE", plain, true)
	if got := visibleWidth(colored); got != utf8.RuneCountInString(plain) {
		t.Errorf("visibleWidth(colored)=%d; want %d", got, utf8.RuneCountInString(plain))
	}
	if got := visibleWidth("box·… drawing"); got != utf8.RuneCountInString("box·… drawing") {
		t.Errorf("visibleWidth miscounted single-width runes: %d", got)
	}
}

// Every boxed row must have the same visible width so the right border lines up,
// whether or not the content carries ANSI colour, at every degradation tier
// (padded, unpadded, and ungrouped) that a short terminal can trigger, and at
// every width — including widths narrower than the longest command, where the
// comment column is dropped and rows are ellipsised to the border.
//
// The box must also never be wider than the terminal it is drawn in.
func TestQuickStartBoxAligned(t *testing.T) {
	for _, color := range []bool{false, true} {
		for _, width := range []int{20, 40, 61, 80, 100} {
			for _, height := range []int{0, 40, 30, 24, 20, 10} {
				out := renderQuickStart(BuildInfo{Version: "1.2.3"}, color, width, height)
				var boxWidth = -1
				for _, line := range strings.Split(out, "\n") {
					if !strings.HasPrefix(stripANSI(line), "╭") &&
						!strings.HasPrefix(stripANSI(line), "│") &&
						!strings.HasPrefix(stripANSI(line), "╰") {
						continue // logo / blank lines above the box
					}
					w := visibleWidth(line)
					if boxWidth == -1 {
						boxWidth = w
					} else if w != boxWidth {
						t.Errorf("color=%v %dx%d: box row width %d != %d for %q", color, width, height, w, boxWidth, stripANSI(line))
					}
				}
				if boxWidth == -1 {
					t.Errorf("color=%v %dx%d: no box rows found", color, width, height)
				} else if boxWidth > width {
					t.Errorf("color=%v %dx%d: box is %d cells wide, overflowing the terminal", color, width, height, boxWidth)
				}
			}
		}
	}
}

func stripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }

func TestTruncateVisible(t *testing.T) {
	// A string that already fits is returned untouched, escapes and all.
	colored := ansiColor("#5A8DEE", "etherscan login", true)
	if got := truncateVisible(colored, 20); got != colored {
		t.Errorf("truncateVisible should not alter a string that fits; got %q", got)
	}

	// Plain truncation: max cells total, with the last one spent on the ellipsis.
	got := truncateVisible("etherscan login", 6)
	if got != "ether…" {
		t.Errorf("truncateVisible(plain, 6) = %q; want %q", got, "ether…")
	}
	if w := visibleWidth(got); w != 6 {
		t.Errorf("truncated width = %d; want 6", w)
	}

	// Cutting inside a coloured span must keep the opening escape intact (never
	// emit a partial sequence), stay 6 cells wide, and close with a reset so the
	// colour cannot bleed into the border that follows.
	got = truncateVisible(colored, 6)
	if w := visibleWidth(got); w != 6 {
		t.Errorf("truncated coloured width = %d; want 6", w)
	}
	if stripANSI(got) != "ether…" {
		t.Errorf("truncated coloured text = %q; want %q", stripANSI(got), "ether…")
	}
	if !strings.HasPrefix(got, "\x1b[") {
		t.Errorf("truncated coloured string lost its opening escape: %q", got)
	}
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("truncated coloured string should end with a reset: %q", got)
	}

	// Degenerate widths must not panic or produce something wider than asked.
	if got := truncateVisible("etherscan", 1); got != "…" {
		t.Errorf("truncateVisible(_, 1) = %q; want %q", got, "…")
	}
	for _, max := range []int{0, -1} {
		if got := truncateVisible("etherscan", max); got != "" {
			t.Errorf("truncateVisible(_, %d) = %q; want empty", max, got)
		}
	}
}

// Below the width that fits the "# comment" column the comments are dropped, but
// every command must survive: the command list is the point of the guide.
func TestQuickStartNarrowDropsCommentsNotCommands(t *testing.T) {
	wide := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 100, 0)
	if !strings.Contains(wide, "# Set up your API key") {
		t.Fatal("wide render should carry the comment column")
	}

	narrow := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 44, 0)
	if strings.Contains(narrow, "# Set up your API key") {
		t.Error("narrow render should drop the comment column")
	}
	for _, g := range quickStartGroups {
		for _, e := range g.items {
			if !strings.Contains(narrow, e.cmd) {
				t.Errorf("narrow render dropped command %q", e.cmd)
			}
		}
	}
}

func TestHexRGB(t *testing.T) {
	r, g, b, ok := hexRGB("#5A8DEE")
	if !ok || r != 90 || g != 141 || b != 238 {
		t.Fatalf("hexRGB(#5A8DEE) = %d,%d,%d ok=%v; want 90,141,238 ok=true", r, g, b, ok)
	}
	if _, _, _, ok := hexRGB("nope"); ok {
		t.Error("hexRGB should reject a malformed value")
	}
}
