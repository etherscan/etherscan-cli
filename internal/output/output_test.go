package output

import "testing"

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
