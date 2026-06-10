package authweb

import (
	"strings"
	"testing"
)

// seqReader is a simple deterministic reader for tests that need varied bytes.
func seqReader(b []byte) (int, error) {
	for i := range b {
		b[i] = byte(i)
	}
	return len(b), nil
}

func TestNewPKCEProducesUrlSafeVerifierAndMatchingChallenge(t *testing.T) {
	v, c, err := NewPKCE(seqReader)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) < 43 || strings.ContainsAny(v, "+/=") {
		t.Fatalf("verifier not url-safe: %q", v)
	}
	if c == "" || c == v {
		t.Fatalf("challenge must be S256 of verifier, got %q", c)
	}
	if c2 := Challenge(v); c2 != c {
		t.Fatalf("Challenge(verifier) mismatch: %q vs %q", c2, c)
	}
}

func TestNewStateIsUrlSafeAndUnique(t *testing.T) {
	s1, err := NewState(nil)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewState(nil)
	if err != nil {
		t.Fatal(err)
	}
	if s1 == s2 {
		t.Fatal("state collision")
	}
	if strings.ContainsAny(s1, "+/=") {
		t.Fatalf("state not url-safe: %q", s1)
	}
}

func TestHashTokenStable(t *testing.T) {
	if HashToken("abc") == nil {
		t.Fatal("nil hash")
	}
	h1 := string(HashToken("abc"))
	h2 := string(HashToken("abc"))
	if h1 != h2 {
		t.Fatal("unstable")
	}
	if string(HashToken("abc")) == string(HashToken("abd")) {
		t.Fatal("collision")
	}
}
