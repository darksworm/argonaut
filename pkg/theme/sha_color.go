package theme

import (
	"hash/fnv"
	"image/color"
	"strings"
)

// ShaColor returns a stable palette color for a commit sha so the same sha
// is recognizable wherever it appears. Hashing only the 8-char prefix keeps
// a sha and its shortened form on the same color. The buckets deliberately
// exclude Danger (alarm) and the dim/neutral tones (unreadable).
func (p Palette) ShaColor(sha string) color.Color {
	buckets := []color.Color{p.Info, p.Accent, p.Success, p.Warning}
	key := strings.ToLower(sha)
	if len(key) > 8 {
		key = key[:8]
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return buckets[h.Sum32()%uint32(len(buckets))]
}
