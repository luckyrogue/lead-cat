package middleware

import (
	"fmt"
	"os"
	"testing"
)

// minCoverage is the floor enforced for this package (formerly
// scripts/check-middleware-coverage.sh).
const minCoverage = 0.50

// TestMain fails the run if statement coverage drops below minCoverage. It only
// fires under `go test -cover`; a plain `go test` leaves testing.Coverage() at
// 0, so the gate is skipped.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		if c := testing.Coverage(); c > 0 && c < minCoverage {
			fmt.Fprintf(os.Stderr, "coverage gate: %.1f%% < %.0f%%\n", c*100, minCoverage*100)
			code = 1
		}
	}
	os.Exit(code)
}
