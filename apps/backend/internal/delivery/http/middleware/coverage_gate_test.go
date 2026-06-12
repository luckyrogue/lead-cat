package middleware

import (
	"fmt"
	"os"
	"testing"
)

const minCoverage = 0.50

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
