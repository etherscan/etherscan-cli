package cli

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderQuickStartPlain(t *testing.T) {
	out := renderQuickStart(BuildInfo{Version: "1.2.3"}, false, 100)

	if strings.Contains(out, "\x1b[") {
		t.Fatalf("plain render must contain no ANSI escapes, got:\n%s", out)
	}

	for _, want := range []string{
		"Quick Start",
		"Official API Command Line Interface",
		"docs.etherscan.io",
		"Etherscan CLI 1.2.3",
		"╭", "╰", "│", // rounded box glyphs (not ANSI)
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plain render missing %q", want)
		}
	}
	for _, e := range quickStartExamples {
		if !strings.Contains(out, e.cmd) {
			t.Errorf("plain render missing example command %q", e.cmd)
		}
		if !strings.Contains(out, e.comment) {
			t.Errorf("plain render missing comment %q", e.comment)
		}
	}
}

func TestRenderQuickStartColored(t *testing.T) {
	out := renderQuickStart(BuildInfo{Version: "1.2.3"}, true, 100)

	if !strings.Contains(out, "\x1b[") {
		t.Fatal("colored render should contain ANSI escapes")
	}
	// The accent (#5A8DEE -> 90,141,238) should be present as a truecolor escape.
	if !strings.Contains(out, "\x1b[38;2;90;141;238m") {
		t.Errorf("colored render missing brand accent escape, got:\n%q", out)
	}
	// Content is identical to the plain render once escapes are stripped of the
	// example commands (spot check one).
	if !strings.Contains(out, "etherscan login") {
		t.Error("colored render missing example command")
	}
}

func TestRenderQuickStartNarrowFallback(t *testing.T) {
	// A width narrower than the full wordmark collapses to the one-line logo.
	out := renderQuickStart(BuildInfo{}, false, 10)
	lines := strings.Split(out, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "Etherscan" {
		t.Errorf("narrow render should start with the one-line Etherscan wordmark, got first line %q", lines[0])
	}
	// Version subtitle is omitted when Version is empty.
	if strings.Contains(out, "Etherscan CLI ") {
		t.Error("empty version should not render a version subtitle")
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
// whether or not the content carries ANSI colour.
func TestQuickStartBoxAligned(t *testing.T) {
	for _, color := range []bool{false, true} {
		out := renderQuickStart(BuildInfo{Version: "1.2.3"}, color, 100)
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
				t.Errorf("color=%v: box row width %d != %d for %q", color, w, boxWidth, stripANSI(line))
			}
		}
		if boxWidth == -1 {
			t.Errorf("color=%v: no box rows found", color)
		}
	}
}

func stripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }

func TestHexRGB(t *testing.T) {
	r, g, b, ok := hexRGB("#5A8DEE")
	if !ok || r != 90 || g != 141 || b != 238 {
		t.Fatalf("hexRGB(#5A8DEE) = %d,%d,%d ok=%v; want 90,141,238 ok=true", r, g, b, ok)
	}
	if _, _, _, ok := hexRGB("nope"); ok {
		t.Error("hexRGB should reject a malformed value")
	}
}
