package meeting

import (
	"testing"
	"time"
)

func TestGenerateName_Once(t *testing.T) {
	d := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	got := GenerateName("Разработка", "Планёрка", "Иванов А.А.", d, Once)
	want := "Разработка | Планёрка | Иванов А.А. | 2025-06-02"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGenerateName_Weekly(t *testing.T) {
	d := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)
	got := GenerateName("Разработка", "Планёрка", "Иванов А.А.", d, Weekly)
	want := "Разработка | Планёрка | Иванов А.А. | 2025-06-02 | Еженедельно"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
