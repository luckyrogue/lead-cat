// Package employeedir loads the corporate employee directory from an embedded
// CSV and syncs it into Google-configured workspaces. §1.2 / §9.4
package employeedir

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

// Record is one parsed directory entry. Email is normalized to lower-case.
type Record struct {
	FullName string
	Email    string
	Dept     string
}

var wantHeader = []string{"full_name", "email", "department"}

// Parse reads employees.csv bytes into records. The header row must be exactly
// full_name,email,department (case- and whitespace-insensitive). Blank lines and
// rows with an empty email are skipped; remaining fields are trimmed and the
// email is lower-cased. §9.4
func Parse(data []byte) ([]Record, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // tolerate ragged rows; we validate the header ourselves
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse employees csv: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	head := rows[0]
	if len(head) < len(wantHeader) {
		return nil, fmt.Errorf("invalid header: got %v, want full_name,email,department", head)
	}
	for i, want := range wantHeader {
		// guard: len(head) >= len(wantHeader) checked above, so head[i] is safe.
		if got := strings.ToLower(strings.TrimSpace(head[i])); got != want {
			return nil, fmt.Errorf("invalid header col %d: got %q, want %q", i, got, want)
		}
	}
	var out []Record
	for _, row := range rows[1:] {
		if len(row) < 3 {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(row[1]))
		if email == "" {
			continue
		}
		out = append(out, Record{
			FullName: strings.TrimSpace(row[0]),
			Email:    email,
			Dept:     strings.TrimSpace(row[2]),
		})
	}
	return out, nil
}
