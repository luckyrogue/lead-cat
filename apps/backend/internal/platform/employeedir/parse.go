package employeedir

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

type Record struct {
	FullName string
	Email    string
	Dept     string
}

var wantHeader = []string{"full_name", "email", "department"}

func Parse(data []byte) ([]Record, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
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
