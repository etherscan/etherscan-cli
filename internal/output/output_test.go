package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRawValuePassthrough(t *testing.T) {
	const wei = "1500000000000000000" // 1.5 in 18-decimal units

	// Scalar results (e.g. account balance) print the raw integer, no unit.
	if got := formatScalar(wei); got != wei {
		t.Fatalf("formatScalar(%q)=%q, want raw value", wei, got)
	}

	// Table "value"/"balance" cells print the raw integer, no "ETH"/symbol.
	for _, col := range []string{"value", "balance"} {
		if got := formatTableCell(col, wei); got != wei {
			t.Fatalf("formatTableCell(%q,%q)=%q, want raw value", col, wei, got)
		}
	}

	// Non-numeric values pass through unchanged.
	if got := formatTableCell("tokenSymbol", "USDC"); got != "USDC" {
		t.Fatalf("formatTableCell tokenSymbol=%q, want USDC", got)
	}
}

// Pretty JSON must print numbers exactly as the API sent them. Decoding into
// float64 rounded 9007199254740993 (2^53+1) to ...992 and rewrote 1.50 as 1.5.
func TestWriteJSONKeepsExactNumbers(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []byte(`{"big":9007199254740993,"dec":1.50,"exp":1e3}`), JSON, false, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"big": 9007199254740993`, `"dec": 1.50`, `"exp": 1e3`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output missing %s:\n%s", want, buf.String())
		}
	}
}

// Token names and symbols are chosen by whoever deploys the contract. An OSC 52
// sequence ("set the clipboard") in one must never reach the terminal raw.
func TestTextOutputStripsTerminalEscapes(t *testing.T) {
	const osc52 = "\x1b]52;c;MHhCQUQ=\x07"
	row := `{"tokenSymbol":"USDC` + jsonEscape(osc52) + `","tokenName":"` + jsonEscape("\x1b[32mUSD Coin\x1b[0m\u009b2J") + `"}`
	cases := []struct {
		name   string
		raw    string
		format Format
	}{
		{"table", "[" + row + "]", Table},
		{"csv", "[" + row + "]", CSV},
		{"scalar", `"1000` + jsonEscape(osc52) + `"`, Table},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := Write(&buf, []byte(tc.raw), tc.format, false, nil); err != nil {
				t.Fatal(err)
			}
			if i := strings.IndexAny(buf.String(), "\x1b\x07\u009b"); i >= 0 {
				t.Fatalf("raw control character at byte %d:\n%q", i, buf.String())
			}
			if !strings.Contains(buf.String(), "\uFFFD") {
				t.Fatalf("expected U+FFFD where the escape was:\n%q", buf.String())
			}
		})
	}
}

// JSON output stays raw data: the encoder escapes control characters itself.
func TestJSONOutputKeepsEscapedControls(t *testing.T) {
	raw := []byte(`{"tokenSymbol":"USDC\u001b]52;c;x\u0007"}`)
	for _, compact := range []bool{false, true} {
		var buf bytes.Buffer
		if err := Write(&buf, raw, JSON, compact, nil); err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(buf.String(), "\x1b\x07") || !strings.Contains(buf.String(), `\u001b]52;c;x\u0007`) {
			t.Fatalf("compact=%v: JSON should keep the escaped form, got %q", compact, buf.String())
		}
	}
}

// Sanitize keeps whitespace controls, so tables must collapse them. A token name such as
// "USDC\rFAKE" would otherwise return the cursor to column 0 and overwrite the address
// printed before it.
func TestTableCollapsesCursorControls(t *testing.T) {
	for _, ctl := range []string{"\r", "\v", "\f", "\n", "\r\n"} {
		raw := `[{"to":"0xabc","tokenName":"USDC` + jsonEscape(ctl) + `FAKE","na` + jsonEscape(ctl) + `me":"x"}]`
		var buf bytes.Buffer
		if err := Write(&buf, []byte(raw), Table, false, nil); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if i := strings.IndexAny(strings.TrimSuffix(out, "\n"), "\r\v\f"); i >= 0 {
			t.Fatalf("%q: cursor control at byte %d:\n%q", ctl, i, out)
		}
		if lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n"); len(lines) != 2 {
			t.Fatalf("%q: want header + 1 row, got %d lines:\n%q", ctl, len(lines), out)
		}
		if !strings.Contains(out, "NA ME") || !strings.Contains(out, "0xabc") || !strings.Contains(out, "USDC FAKE") {
			t.Fatalf("%q: want header \"NA ME\" and cell \"USDC FAKE\":\n%q", ctl, out)
		}
	}
}

func TestSanitizeKeepsWhitespaceAndText(t *testing.T) {
	for _, s := range []string{"pragma solidity ^0.8.0;\r\n\tcontract A {}\n", "page\fbreak\vtab", "👨‍💻.eth", "Ünïcödé", ""} {
		if got := Sanitize(s); got != s {
			t.Errorf("Sanitize(%q) = %q, want unchanged", s, got)
		}
	}
	if got := Sanitize("a\x00b\x7fc\u0085d\xffe"); got != "a\uFFFDb\uFFFDc\uFFFDd\uFFFDe" {
		t.Errorf("Sanitize replaced wrongly: %q", got)
	}
}

// Verified source code often uses CRLF line endings; CSV must keep them intact.
func TestCSVKeepsCRLFSource(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []byte(`[{"SourceCode":"line1\r\nline2\r\n"}]`), CSV, false, nil); err != nil {
		t.Fatal(err)
	}
	if want := "SourceCode\n\"line1\r\nline2\r\n\"\n"; buf.String() != want {
		t.Fatalf("csv = %q, want %q", buf.String(), want)
	}
}

// jsonEscape writes control characters as \u escapes, as an API response would.
func jsonEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			fmt.Fprintf(&b, `\u%04x`, r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
