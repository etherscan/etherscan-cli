package brand

import "testing"

func TestLogoNonEmpty(t *testing.T) {
	if len(Logo) == 0 {
		t.Fatal("Logo must not be empty")
	}
	for i, line := range Logo {
		if line == "" {
			t.Errorf("Logo line %d is empty", i)
		}
	}
}

func TestPaletteHex(t *testing.T) {
	for name, hex := range map[string]string{"AccentHex": AccentHex, "DimHex": DimHex, "GreenHex": GreenHex, "ErrorHex": ErrorHex} {
		if len(hex) != 7 || hex[0] != '#' {
			t.Errorf("%s = %q; want a #rrggbb string", name, hex)
		}
	}
}
