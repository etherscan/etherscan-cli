package updater

import (
	"fmt"
	"strconv"
	"strings"
)

type semanticVersion struct {
	major, minor, patch uint64
	prerelease          []string
}

func parseVersion(value string) (semanticVersion, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if value == "" || value == "dev" {
		return semanticVersion{}, fmt.Errorf("development builds cannot check for updates")
	}
	if i := strings.IndexByte(value, '+'); i >= 0 {
		value = value[:i]
	}
	var prerelease []string
	if i := strings.IndexByte(value, '-'); i >= 0 {
		prerelease = strings.Split(value[i+1:], ".")
		value = value[:i]
		if len(prerelease) == 0 {
			return semanticVersion{}, fmt.Errorf("invalid version")
		}
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return semanticVersion{}, fmt.Errorf("invalid version")
	}
	parsed := semanticVersion{prerelease: prerelease}
	numbers := []*uint64{&parsed.major, &parsed.minor, &parsed.patch}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semanticVersion{}, fmt.Errorf("invalid version")
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semanticVersion{}, fmt.Errorf("invalid version")
		}
		*numbers[i] = n
	}
	for _, identifier := range prerelease {
		if identifier == "" {
			return semanticVersion{}, fmt.Errorf("invalid version")
		}
		for _, r := range identifier {
			if !(r == '-' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				return semanticVersion{}, fmt.Errorf("invalid version")
			}
		}
	}
	return parsed, nil
}

func compareVersions(left, right semanticVersion) int {
	for _, pair := range [][2]uint64{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if len(left.prerelease) == 0 && len(right.prerelease) == 0 {
		return 0
	}
	if len(left.prerelease) == 0 {
		return 1
	}
	if len(right.prerelease) == 0 {
		return -1
	}
	for i := 0; i < len(left.prerelease) && i < len(right.prerelease); i++ {
		l, r := left.prerelease[i], right.prerelease[i]
		if l == r {
			continue
		}
		ln, lerr := strconv.ParseUint(l, 10, 64)
		rn, rerr := strconv.ParseUint(r, 10, 64)
		switch {
		case lerr == nil && rerr == nil:
			if ln < rn {
				return -1
			}
			return 1
		case lerr == nil:
			return -1
		case rerr == nil:
			return 1
		case l < r:
			return -1
		default:
			return 1
		}
	}
	if len(left.prerelease) < len(right.prerelease) {
		return -1
	}
	if len(left.prerelease) > len(right.prerelease) {
		return 1
	}
	return 0
}

func canonicalVersion(value string) (string, semanticVersion, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(value, "v"))
	parsed, err := parseVersion(trimmed)
	if err != nil {
		return "", semanticVersion{}, err
	}
	return trimmed, parsed, nil
}
