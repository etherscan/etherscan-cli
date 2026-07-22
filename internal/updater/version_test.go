package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.4", "1.2.3", 1},
		{"1.3.0", "1.2.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.1.0", "1.1.0-rc.5", 1},
		{"1.1.0-rc.5", "1.1.0-rc.4", 1},
		{"1.1.0-rc.1", "1.1.0", -1},
		{"1.1.0-alpha.2", "1.1.0-alpha.10", -1},
	}
	for _, tt := range tests {
		t.Run(tt.left+"_"+tt.right, func(t *testing.T) {
			left, err := parseVersion(tt.left)
			if err != nil {
				t.Fatal(err)
			}
			right, err := parseVersion(tt.right)
			if err != nil {
				t.Fatal(err)
			}
			if got := compareVersions(left, right); got != tt.want {
				t.Fatalf("compareVersions() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseVersionRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "dev", "1.2", "01.2.3", "1.2.3-", "1.2.x"} {
		if _, err := parseVersion(value); err == nil {
			t.Fatalf("parseVersion(%q) accepted an invalid value", value)
		}
	}
}
