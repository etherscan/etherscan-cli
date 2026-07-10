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

// minLandingHeight is the terminal height below which the big banner is dropped in
// favour of the compact one-line wordmark, so the modules/endpoints panels stay on
// screen on short terminals.
const minLandingHeight = 34

// renderBanner returns the landing wordmark. On a terminal both wide and tall
// enough it returns the full pixel-art logo painted in the accent colour;
// otherwise it returns a compact one-line fallback so the banner is never blank.
func renderBanner(width, height int) string {
	if width >= bannerWidth && height >= minLandingHeight {
		lines := make([]string, len(bannerLogo))
		for i, row := range bannerLogo {
			lines[i] = logoSt.Render(row)
		}
		return strings.Join(lines, "\n") + "\n\n" + subSt.Render(bannerTagline)
	}
	return logoSt.Render("Etherscan") + "  " + subSt.Render(bannerTagline)
}
