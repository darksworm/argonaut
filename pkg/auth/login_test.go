package auth

import (
	"testing"
)

func TestStripProtocol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://foo.com", "foo.com"},
		{"http://foo.com:8080", "foo.com:8080"},
		{"foo.com", "foo.com"},
		{"", ""},
	}
	for _, tc := range tests {
		got := StripProtocol(tc.input)
		if got != tc.want {
			t.Errorf("StripProtocol(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
