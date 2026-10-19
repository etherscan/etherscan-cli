package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
)

type Format string

const (
	Table Format = "table"
	JSON  Format = "json"
	CSV   Format = "csv"
)

// DefaultFormat matches the Etherscan API, which serves application/json for every
// endpoint. Tables remain available via -o table.
const DefaultFormat = JSON

// ParseFormat validates a format name from --output or default_output. Unknown values are
// rejected rather than falling through to the table renderer, which is what Write would
// otherwise do: it tests for JSON and CSV, so anything else silently rendered as a table.
func ParseFormat(value string) (Format, error) {
	switch f := Format(strings.ToLower(strings.TrimSpace(value))); f {
	case Table, JSON, CSV:
		return f, nil
	default:
		return "", fmt.Errorf("unknown output format %q (use json, table, or csv)", value)
	}
}

func Write(w io.Writer, raw json.RawMessage, format Format, compact bool, columns []string) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if format == JSON {
		if compact {
			_, err := w.Write(raw)
			if err == nil {
				_, err = fmt.Fprintln(w)
			}
			return err
		}
		// UseNumber keeps numeric literals exact: decoding into float64 would round
		// integers above 2^53 and rewrite decimals such as 1.50 as 1.5.
		var v any
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			return err
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(v)
	}
	rows, scalar, err := Rows(raw)
	if err != nil {
		return err
	}
	if scalar != "" {
		_, err := fmt.Fprintln(w, Sanitize(scalar))
		return err
	}
	if format == CSV {
		return writeCSV(w, rows, columns)
	}
	return writeTable(w, rows, columns)
}

func WriteRows(w io.Writer, rows []map[string]string, format Format, columns []string) error {
	if format == CSV {
		return writeCSV(w, rows, columns)
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	if format == JSON {
		return Write(w, raw, JSON, false, columns)
	}
	return writeTable(w, rows, columns)
}

func Rows(raw json.RawMessage) ([]map[string]string, string, error) {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, "", err
	}
	switch typed := v.(type) {
	case []any:
		rows := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			if row, ok := stringifyMap(item); ok {
				rows = append(rows, row)
			}
		}
		return rows, "", nil
	case map[string]any:
		row, _ := stringifyMap(typed)
		return []map[string]string{row}, "", nil
	case string:
		return nil, formatScalar(typed), nil
	case json.Number:
		return nil, typed.String(), nil
	default:
		return nil, fmt.Sprint(typed), nil
	}
}

func writeTable(w io.Writer, rows []map[string]string, preferred []string) error {
	if len(rows) == 0 {
		return nil
	}
	cols := columns(rows, preferred)
	header := make([]string, len(cols))
	for i, col := range cols {
		header[i] = collapseWhitespace(Sanitize(col))
	}
	t := tablewriter.NewWriter(w)
	t.SetHeader(header)
	t.SetAutoWrapText(false)
	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	t.SetBorder(false)
	t.SetCenterSeparator("")
	t.SetColumnSeparator("")
	t.SetRowSeparator("")
	t.SetHeaderLine(false)
	t.SetTablePadding("  ")
	t.SetNoWhiteSpace(true)
	for _, row := range rows {
		record := make([]string, len(cols))
		for i, col := range cols {
			record[i] = formatTableCell(col, row[col])
		}
		t.Append(record)
	}
	t.Render()
	return nil
}

func writeCSV(w io.Writer, rows []map[string]string, preferred []string) error {
	if len(rows) == 0 {
		return nil
	}
	cols := columns(rows, preferred)
	cw := csv.NewWriter(w)
	header := make([]string, len(cols))
	for i, col := range cols {
		header[i] = Sanitize(col)
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(cols))
		for i, col := range cols {
			record[i] = Sanitize(row[col])
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func columns(rows []map[string]string, preferred []string) []string {
	seen := map[string]bool{}
	for _, row := range rows {
		for key := range row {
			seen[key] = true
		}
	}
	cols := []string{}
	for _, col := range preferred {
		if seen[col] {
			cols = append(cols, col)
			delete(seen, col)
		}
	}
	if len(cols) > 0 {
		return cols
	}
	rest := make([]string, 0, len(seen))
	for col := range seen {
		rest = append(rest, col)
	}
	sort.Strings(rest)
	return append(cols, rest...)
}

func stringifyMap(value any) (map[string]string, bool) {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	row := map[string]string{}
	for k, v := range raw {
		switch typed := v.(type) {
		case nil:
			row[k] = ""
		case string:
			row[k] = typed
		case json.Number:
			row[k] = typed.String()
		default:
			encoded, err := json.Marshal(typed)
			if err == nil {
				row[k] = string(encoded)
			} else {
				row[k] = fmt.Sprint(typed)
			}
		}
	}
	return row, true
}

func formatScalar(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	if unix, err := strconv.ParseInt(trimmed, 10, 64); err == nil && unix > 946684800 && unix < 4102444800 {
		return fmt.Sprintf("%s (%s)", trimmed, time.Unix(unix, 0).UTC().Format(time.RFC3339))
	}
	return value
}

func formatTableCell(column, value string) string {
	value = Sanitize(value)
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	// Collapse before the truncation rules below, which would otherwise keep newlines that
	// happen to fall inside the retained prefix.
	if collapsed := collapseWhitespace(trimmed); collapsed != trimmed {
		trimmed = collapsed
		value = trimmed
	}
	switch strings.ToLower(column) {
	case "timestamp", "timestampunix", "time_stamp":
		if unix, err := strconv.ParseInt(trimmed, 10, 64); err == nil && unix > 946684800 && unix < 4102444800 {
			return time.Unix(unix, 0).UTC().Format(time.RFC3339)
		}
	}
	if strings.HasPrefix(trimmed, "0x") && len(trimmed) > 42 {
		return trimmed[:18] + "..." + trimmed[len(trimmed)-10:]
	}
	if len(trimmed) > 80 {
		return trimmed[:77] + "..."
	}
	return value
}

// collapseWhitespace turns each run of whitespace in a table cell or header into one
// space. Cell values may contain newlines (contract SourceCode is the worst case), and
// writeTable runs with SetAutoWrapText(false), so a newline would split one logical row
// across several physical lines and misalign every following column. It also covers the
// whitespace controls Sanitize keeps: a token name such as "USDC\rFAKE" would otherwise
// return the cursor to column 0 and overwrite the columns printed before it.
func collapseWhitespace(s string) string {
	if !strings.ContainsAny(s, "\r\n\t\v\f") {
		return s
	}
	return strings.Join(strings.Fields(s), " ")
}

// Sanitize replaces control characters in API-derived text with U+FFFD before it is
// written to a terminal. Token names and symbols are chosen by whoever deploys the
// contract, so a raw ESC could smuggle in an escape sequence such as OSC 52, which
// many terminals honour as "overwrite the clipboard". Whitespace controls (tab,
// newline, vertical tab, form feed, carriage return) are kept: they cannot start an
// escape sequence and occur in real data such as contract source code. Table output
// additionally collapses them to spaces (see collapseWhitespace). JSON output
// does not need this, because the encoder escapes every control character.
func Sanitize(s string) string {
	if utf8.ValidString(s) && strings.IndexFunc(s, unsafeControl) < 0 {
		return s
	}
	// strings.Map also turns invalid UTF-8 bytes into U+FFFD, which covers a lone
	// 8-bit C1 byte such as 0x9b (CSI).
	return strings.Map(func(r rune) rune {
		if unsafeControl(r) {
			return utf8.RuneError
		}
		return r
	}, s)
}

func unsafeControl(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r':
		return false
	}
	return unicode.IsControl(r)
}
