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
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
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
		_, err := fmt.Fprintln(w, scalar)
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
	t := tablewriter.NewWriter(w)
	t.SetHeader(cols)
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
	if err := cw.Write(cols); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(cols))
		for i, col := range cols {
			record[i] = row[col]
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
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	// Cell values may contain newlines (contract SourceCode is the worst case). writeTable
	// runs with SetAutoWrapText(false), so an unsanitised newline splits one logical row
	// across several physical lines and misaligns every following column. Collapse runs of
	// whitespace before the truncation rules below, which would otherwise keep newlines
	// that happen to fall inside the retained prefix.
	if strings.ContainsAny(trimmed, "\r\n\t\v\f") {
		trimmed = strings.Join(strings.Fields(trimmed), " ")
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
