package theme

import (
	"fmt"
	"testing"
)

// ShaColor gives every commit sha a stable palette color derived from its
// content, so the same sha is recognizable wherever it appears.
func TestShaColor_DeterministicPerSha(t *testing.T) {
	p := Default()

	if p.ShaColor("4caae2c0") != p.ShaColor("4caae2c0") {
		t.Error("the same sha must always get the same color")
	}
	if p.ShaColor("4caae2c0e29c5ada8c840af6718bdcf3eeb57baa") != p.ShaColor("4caae2c0") {
		t.Error("a sha and its 8-char prefix must get the same color")
	}
}

func TestShaColor_SpreadsAcrossPaletteBuckets(t *testing.T) {
	p := Default()
	seen := map[string]bool{}
	for _, sha := range []string{"4caae2c0", "a1b2c3d4", "deadbeef", "0f0e0d0c", "12345678", "fedcba98"} {
		seen[fmt.Sprintf("%v", p.ShaColor(sha))] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected shas to spread over multiple palette colors, got %d", len(seen))
	}
}
