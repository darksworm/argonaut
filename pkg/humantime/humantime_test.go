package humantime

import (
	"testing"
	"time"
)

var now = time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)

func TestAgo(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds", now.Add(-45 * time.Second), "45s ago"},
		{"minutes floor", now.Add(-2*time.Minute - 30*time.Second), "2m ago"},
		{"hours floor", now.Add(-90 * time.Minute), "1h ago"},
		{"days floor", now.Add(-26 * time.Hour), "1d ago"},
		{"future clamps to zero", now.Add(3 * time.Second), "0s ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Ago(tt.at, now); got != tt.want {
				t.Errorf("Ago(%v) = %q, want %q", tt.at, got, tt.want)
			}
		})
	}
}

func TestAgoLong(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds", now.Add(-45 * time.Second), "45 seconds ago"},
		{"one minute singular", now.Add(-1 * time.Minute), "1 minute ago"},
		{"minutes floor", now.Add(-2*time.Minute - 30*time.Second), "2 minutes ago"},
		{"one hour singular", now.Add(-90 * time.Minute), "1 hour ago"},
		{"days", now.Add(-49 * time.Hour), "2 days ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AgoLong(tt.at, now); got != tt.want {
				t.Errorf("AgoLong(%v) = %q, want %q", tt.at, got, tt.want)
			}
		})
	}
}

func TestAge(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds", now.Add(-45 * time.Second), "45s"},
		{"minutes", now.Add(-2 * time.Minute), "2m"},
		{"hours", now.Add(-90 * time.Minute), "1h"},
		{"days", now.Add(-72 * time.Hour), "3d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Age(tt.at, now); got != tt.want {
				t.Errorf("Age(%v) = %q, want %q", tt.at, got, tt.want)
			}
		})
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"seconds only", 6 * time.Second, "6s"},
		{"minutes and seconds", 3*time.Minute + 42*time.Second, "3m42s"},
		{"whole minutes keep zero seconds", 2 * time.Minute, "2m0s"},
		{"hours and minutes", time.Hour + 2*time.Minute, "1h2m"},
		{"negative clamps to zero", -5 * time.Second, "0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Duration(tt.d); got != tt.want {
				t.Errorf("Duration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
