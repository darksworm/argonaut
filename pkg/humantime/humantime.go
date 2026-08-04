// Package humantime formats timestamps and durations the way a human would
// say them in a status line: "2m ago", "2 minutes ago", "3m42s".
package humantime

import (
	"fmt"
	"time"
)

// AgoLong formats how long before now t happened, spelled out: "2 minutes ago".
func AgoLong(t, now time.Time) string {
	elapsed := now.Sub(t)
	if elapsed < 0 {
		elapsed = 0
	}
	var n int
	var unit string
	switch {
	case elapsed >= 24*time.Hour:
		n, unit = int(elapsed.Hours()/24), "day"
	case elapsed >= time.Hour:
		n, unit = int(elapsed.Hours()), "hour"
	case elapsed >= time.Minute:
		n, unit = int(elapsed.Minutes()), "minute"
	default:
		n, unit = int(elapsed.Seconds()), "second"
	}
	if n != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s ago", n, unit)
}

// Duration formats an elapsed duration compactly: "6s", "3m42s", "1h2m".
func Duration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// Ago formats how long before now t happened, in compact form: "2m ago".
// Uses the largest whole unit (s/m/h/d); future times clamp to "0s ago".
func Ago(t, now time.Time) string {
	elapsed := now.Sub(t)
	if elapsed < 0 {
		elapsed = 0
	}
	switch {
	case elapsed >= 24*time.Hour:
		return fmt.Sprintf("%dd ago", int(elapsed.Hours()/24))
	case elapsed >= time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	case elapsed >= time.Minute:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	default:
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	}
}
