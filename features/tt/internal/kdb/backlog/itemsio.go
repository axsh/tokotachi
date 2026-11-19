package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// utf8BOM is stripped from every JSON file read (Python "utf-8-sig").
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ParseItems decodes a JSON array of item objects. Empty input yields no items.
// source is used in error messages only.
func ParseItems(raw []byte, source string) ([]Item, error) {
	objs, err := parseRawObjects(raw, source)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(objs))
	for _, m := range objs {
		it, err := NormItem(m)
		if err != nil {
			return nil, bizErr("%s: %v", source, err)
		}
		items = append(items, it)
	}
	return items, nil
}

// readRawInput reads a file ("-" for stdin) without interpreting it.
func readRawInput(path string) ([]byte, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, bizErr("read %s: %v", path, err)
	}
	return raw, nil
}

// parseRawObjects decodes a JSON array of objects (numbers kept as json.Number).
// Empty input yields no objects.
func parseRawObjects(raw []byte, source string) ([]map[string]any, error) {
	raw = bytes.TrimPrefix(raw, utf8BOM)
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return []map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var data any
	if err := dec.Decode(&data); err != nil {
		return nil, bizErr("invalid JSON in %s: %v", source, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, bizErr("invalid JSON in %s: trailing data after top-level value", source)
	}
	arr, ok := data.([]any)
	if !ok {
		return nil, bizErr("%s must contain a JSON array", source)
	}
	objs := make([]map[string]any, 0, len(arr))
	for _, el := range arr {
		m, ok := el.(map[string]any)
		if !ok {
			return nil, bizErr("%s: array element is not an object", source)
		}
		objs = append(objs, m)
	}
	return objs, nil
}

// ReadItems reads a JSON array file. A missing or empty file yields no items;
// path "-" reads stdin.
func ReadItems(path string) ([]Item, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
	}
	if err != nil {
		return nil, bizErr("read %s: %v", path, err)
	}
	return ParseItems(raw, path)
}

// WriteItems writes items in the canonical pretty-printed form, creating
// parent directories as needed.
func WriteItems(path string, items []Item) error {
	return writeFileBytes(path, CanonicalArrayBytes(items))
}

func writeFileBytes(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return bizErr("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return bizErr("write %s: %v", path, err)
	}
	return nil
}

// CanonicalArrayBytes renders items exactly like Python
// json.dumps(items, ensure_ascii=False, indent=2) + "\n" (UTF-8). The byte
// form is hashed into the corpus content_hash, so it must stay stable.
func CanonicalArrayBytes(items []Item) []byte {
	var b strings.Builder
	if len(items) == 0 {
		b.WriteString("[]\n")
		return []byte(b.String())
	}
	b.WriteString("[\n")
	for i, it := range items {
		b.WriteString("  {\n")
		vals := itemValues(it)
		for j, col := range Columns {
			b.WriteString("    ")
			writePyString(&b, col)
			b.WriteString(": ")
			if col == "state_date" && it.StateDate == nil {
				b.WriteString("null")
			} else {
				writePyString(&b, vals[j])
			}
			if j < len(Columns)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("  }")
		if i < len(items)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return []byte(b.String())
}

// CompactArrayBytes renders items as one compact JSON line (dump format).
func CompactArrayBytes(items []Item) ([]byte, error) {
	if items == nil {
		items = []Item{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(items); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// itemValues returns column values in Columns order (state_date as "" when nil).
func itemValues(it Item) []string {
	sd := ""
	if it.StateDate != nil {
		sd = *it.StateDate
	}
	return []string{
		it.Phase, it.Branch, it.LocalID, it.Slug, it.Title, it.State, sd, it.Kind,
		it.AskTo, it.Due, it.Origin, it.Decision, it.Problem, it.CurrentHandling, it.Options,
		it.Questions, it.AfterDecision, it.History, it.CreatedAt, it.UpdatedAt, it.UUID,
	}
}

// writePyString writes s as a JSON string using Python's ensure_ascii=False
// escaping rules: only quote, backslash and control characters are escaped.
func writePyString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r < 0x20:
			fmt.Fprintf(b, `\u%04x`, r)
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	b.WriteByte('"')
}
