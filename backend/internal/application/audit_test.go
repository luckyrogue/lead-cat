package application

import (
	"encoding/json"
	"testing"
)

func TestSanitizeAuditDetails_GoogleConfigUpdated(t *testing.T) {
	in := map[string]any{
		"subject":          "admin@example.com",
		"calendar_id":      "primary",
		"has_new_sa_json":  true,
		"google_sa_json":   "leak-secret",
		"random_unrelated": "drop-me",
	}
	out, dropped := sanitizeAuditDetails("google_config_updated", in)
	got := decodeJSON(t, out)
	if _, ok := got["google_sa_json"]; ok {
		t.Fatalf("google_sa_json must be dropped")
	}
	if _, ok := got["random_unrelated"]; ok {
		t.Fatalf("random_unrelated must be dropped")
	}
	if got["subject"] != "admin@example.com" || got["calendar_id"] != "primary" || got["has_new_sa_json"] != true {
		t.Fatalf("whitelist keys lost; got=%v", got)
	}
	if len(dropped) != 2 {
		t.Fatalf("expected 2 dropped keys, got %v", dropped)
	}
}

func TestSanitizeAuditDetails_UnknownAction(t *testing.T) {
	out, _ := sanitizeAuditDetails("totally_unknown", map[string]any{"anything": 1})
	got := decodeJSON(t, out)
	if len(got) != 0 {
		t.Fatalf("unknown action must yield empty details, got %v", got)
	}
}

func decodeJSON(t *testing.T, b json.RawMessage) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
