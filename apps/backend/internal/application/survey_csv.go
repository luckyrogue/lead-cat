package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func sanitizeCSVCell(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

func ResponsesCSV(sv model.Survey, responses []model.SurveyResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{"created_at", "name", "email", "reason", "status"}
	for _, q := range sv.Questions {
		header = append(header, q.Prompt)
	}
	_ = w.Write(header)

	for _, r := range responses {
		byQ := map[string]string{}
		for _, a := range r.Answers {
			byQ[a.QuestionID.String()] = sanitizeCSVCell(answerToString(a.Value))
		}
		row := []string{
			r.CreatedAt.Format("2006-01-02 15:04"),
			sanitizeCSVCell(r.BookerName),
			sanitizeCSVCell(r.BookerEmail),
			sanitizeCSVCell(r.DeclineReason),
			r.Status,
		}
		for _, q := range sv.Questions {
			row = append(row, byQ[q.ID.String()])
		}
		_ = w.Write(row)
	}
	w.Flush()
	return buf.Bytes()
}

func answerToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, "; ")
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = fmt.Sprintf("%v", e)
		}
		return strings.Join(parts, "; ")
	case float64:
		return fmt.Sprintf("%g", t)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}
