// Package brand is the single source of truth for the Etherscan CLI's visual
// identity: the block-art wordmark and the accent palette. It carries no
// dependencies so both the interactive TUI (internal/tui) and the plain CLI
// (internal/cli) can share the exact same logo and colours without duplicating
// the art or drifting apart.
package brand

// Logo is the block-art "Etherscan" wordmark, traced from the source mockup's
// pixel grid with pairs of rows collapsed into half-block terminal cells. Rows
// are left-aligned; ragged right edges are harmless because callers colour only
// the foreground.
var Logo = []string{
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

// Tagline is the one-line subtitle shown under the wordmark.
const Tagline = "Etherscan API V2 · https://docs.etherscan.io"

// Palette hex values. AccentHex is the brand blue used for the logo, titles and
// selection; DimHex is the muted grey used for subtitles, descriptions and the
// "# comment" hints; GreenHex marks a present API key.
const (
	AccentHex = "#5A8DEE"
	DimHex    = "#8A8A8A"
	GreenHex  = "#98C379"
)
