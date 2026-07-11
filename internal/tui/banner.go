package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// bannerLogo is traced from IMG_9903.png's actual 88x20 pixel grid. Each source
// pixel was majority-sampled from the mockup's 12px by ~11.4px cells, then pairs
// of rows were collapsed into half-block terminal cells. Rows are left-aligned;
// ragged right edges are harmless because logoSt sets only the foreground colour.
var bannerLogo = []string{
	"     ▄▄██████▄▄",
	"  ▄██████████████▄",
	" ▄███████████▀▀▀██▄      ▄▄▄▄▄  ▄   ▄▄",
	"▄████████▀▀▀█   ███▄    ██▀▀▀  ██▄▄ ██ ▄▄▄    ▄▄▄    ▄ ▄▄ ▄▄▄▄    ▄▄▄     ▄▄▄ ▄  ▄▄ ▄▄▄",
	"████▀▀▀██   █   ██▀     ██▄▄▄  ██▀  ██▀ ▀█▄ ▄█▀ ▀██ ██▀▀ ▄█  ▀█ ▄█▀▀▀██ ▄█▀▀▀▀█  ██▀ ▀██",
	"████   ██   █   ▀ ▄█    ██     ██   ██   ██ ██▀▀▀▀▀ ██    ▀▀▀▄▄ ██      ██    █  ██   ██",
	"▀███   ██   ▀   ▄██▀    ██▄▄▄▄ ▀█▄▄ ██   ██ ▀█▄▄▄█▀ ██   ▀█▄▄▄█ ▀█▄▄▄█▀ ▀█▄▄▄██  ██   ██",
	" ▀█▀   ▀    ▄▄████▀",
	"      ▄▄▄▄███████▀",
	"     ▀▀██████▀▀",
}

const bannerTagline = "Etherscan API V2 · https://docs.etherscan.io"

// bannerHeight / bannerWidth describe the block art; used by the size guard below
// and by tests.
var (
	bannerHeight = len(bannerLogo)
	bannerWidth  = bannerBlockWidth()
)

func bannerBlockWidth() int {
	w := 0
	for _, row := range bannerLogo {
		if rw := lipgloss.Width(row); rw > w {
			w = rw
		}
	}
	return w
}

// bigBannerRows is the number of lines the full banner occupies: the block art
// plus a blank line and the tagline.
var bigBannerRows = bannerHeight + 2

// renderBanner returns the landing wordmark. rows is the line budget the caller
// has left for the banner after accounting for everything else on screen: when
// the terminal is wide enough and the full art fits the budget it returns the
// pixel-art logo painted in the accent colour; otherwise a compact one-line
// fallback so the banner is never blank and never pushes the rest off-screen.
func renderBanner(width, rows int) string {
	if width >= bannerWidth && rows >= bigBannerRows {
		lines := make([]string, len(bannerLogo))
		for i, row := range bannerLogo {
			lines[i] = logoSt.Render(row)
		}
		return strings.Join(lines, "\n") + "\n\n" + subSt.Render(bannerTagline)
	}
	return logoSt.Render("Etherscan") + "  " + subSt.Render(bannerTagline)
}
