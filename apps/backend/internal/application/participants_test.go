package application

import (
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  Ivan@Corp.KZ ")
	if err != nil || got != "ivan@corp.kz" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := normalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestFilterEmployees(t *testing.T) {
	all := []postgres.Employee{
		{FullName: "Иван Иванов", Email: "ivan@corp.kz"},
		{FullName: "Пётр Петров", Email: "petr@corp.kz"},
		{FullName: "Анна Сидорова", Email: "anna@corp.kz"},
	}
	if got := filterEmployees(all, "иван"); len(got) != 1 || got[0].Email != "ivan@corp.kz" {
		t.Fatalf("name search: %+v", got)
	}
	if got := filterEmployees(all, "PETR@"); len(got) != 1 || got[0].FullName != "Пётр Петров" {
		t.Fatalf("email search: %+v", got)
	}
	if got := filterEmployees(all, "   "); got != nil {
		t.Fatalf("empty query must yield nil, got %+v", got)
	}
	if got := filterEmployees(all, "zzz"); len(got) != 0 {
		t.Fatalf("no match must yield none, got %+v", got)
	}
}
