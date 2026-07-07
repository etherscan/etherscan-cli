package client

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	addressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	txHashRe  = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	hexRe     = regexp.MustCompile(`^0x[0-9a-fA-F]*$`)
)

func ValidateAddress(name, value string) error {
	if value == "" || addressRe.MatchString(value) {
		return nil
	}
	return fmt.Errorf("%s must be a 0x-prefixed 40-byte hex address", name)
}

func ValidateTxHash(name, value string) error {
	if value == "" || txHashRe.MatchString(value) {
		return nil
	}
	return fmt.Errorf("%s must be a 0x-prefixed 32-byte transaction hash", name)
}

func ValidateHex(name, value string) error {
	if value == "" || hexRe.MatchString(value) {
		return nil
	}
	return fmt.Errorf("%s must be 0x-prefixed hex", name)
}

func ValidateUint(name, value string) error {
	if value == "" {
		return nil
	}
	if _, err := strconv.ParseUint(value, 10, 64); err != nil {
		return fmt.Errorf("%s must be an unsigned integer", name)
	}
	return nil
}

func ValidateDate(name, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%s must use YYYY-MM-DD", name)
	}
	return nil
}

func ValidateSort(value string) error {
	if value == "" || value == "asc" || value == "desc" {
		return nil
	}
	return fmt.Errorf("sort must be asc or desc")
}

func ValidateCommaAddresses(name, value string, max int) error {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	if len(parts) > max {
		return fmt.Errorf("%s accepts at most %d addresses", name, max)
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("%s must not contain empty entries", name)
		}
		if err := ValidateAddress(name, part); err != nil {
			return err
		}
	}
	return nil
}
