package scenario

import (
	"testing"
)

func TestValidateDefinition_ok(t *testing.T) {
	raw := PresetCommits()
	if err := ValidateDefinition(raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDefinition_cycle(t *testing.T) {
	raw := []byte(`{"nodes":[{"id":"a","type":"action.telegram.message"},{"id":"b","type":"action.telegram.cat_photo"}],"edges":[{"source":"a","target":"b"},{"source":"b","target":"a"}]}`)
	if err := ValidateDefinition(raw); err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestValidateDefinition_unknownType(t *testing.T) {
	raw := []byte(`{"nodes":[{"id":"a","type":"unknown"}],"edges":[]}`)
	if err := ValidateDefinition(raw); err == nil {
		t.Fatal("expected unknown type error")
	}
}
