package api

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanSSELines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple SSE event",
			input:    "data: test1\n\ndata: test2\n\n",
			expected: []string{"data: test1\n\n", "data: test2\n\n"},
		},
		{
			name:     "Large line that would trigger chunking",
			input:    "data: " + strings.Repeat("x", 600*1024) + "\n\n",
			expected: []string{}, // Will be handled specially
		},
		{
			name:     "Multiple small events",
			input:    "data: event1\n\ndata: event2\n\ndata: event3\n\n",
			expected: []string{"data: event1\n\n", "data: event2\n\n", "data: event3\n\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			scanner := createSSEScanner(reader, 256*1024, 16*1024*1024)

			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Scanner error: %v", err)
			}

			// For the large line test, we just check that we got some output without error
			if strings.Contains(tt.name, "Large line") {
				if len(lines) == 0 {
					t.Error("Expected to get chunked output for large line")
				}
				return
			}

			// For other tests, check exact output
			if len(lines) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(lines))
				t.Errorf("Got lines: %v", lines)
				return
			}

			for i, line := range lines {
				if line != tt.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestGetSSEBufferConfig(t *testing.T) {
	// Test defaults
	initial, max, growth := getSSEBufferConfig()
	
	if initial != DefaultInitialBuffer {
		t.Errorf("Expected initial buffer %d, got %d", DefaultInitialBuffer, initial)
	}
	
	if max != DefaultMaxBuffer {
		t.Errorf("Expected max buffer %d, got %d", DefaultMaxBuffer, max)
	}
	
	if growth != DefaultGrowthFactor {
		t.Errorf("Expected growth factor %f, got %f", DefaultGrowthFactor, growth)
	}
}

func TestScanSSELinesOrChunk(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		atEOF       bool
		wantAdvance int
		wantToken   []byte
		wantErr     error
	}{
		{
			name:        "Complete SSE event",
			data:        []byte("data: test\n\ndata: next"),
			atEOF:       false,
			wantAdvance: 12,
			wantToken:   []byte("data: test\n\n"),
			wantErr:     nil,
		},
		{
			name:        "Incomplete SSE event needs more data",
			data:        []byte("data: test\n"),
			atEOF:       false,
			wantAdvance: 0,
			wantToken:   nil,
			wantErr:     nil,
		},
		{
			name:        "EOF with remaining data",
			data:        []byte("data: final"),
			atEOF:       true,
			wantAdvance: 11,
			wantToken:   []byte("data: final"),
			wantErr:     nil,
		},
		{
			name:        "Large data triggers chunking",
			data:        bytes.Repeat([]byte("x"), 2*1024*1024),
			atEOF:       false,
			wantAdvance: 1024 * 1024,
			wantToken:   bytes.Repeat([]byte("x"), 1024*1024),
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advance, token, err := scanSSELinesOrChunk(tt.data, tt.atEOF)

			if advance != tt.wantAdvance {
				t.Errorf("advance = %d, want %d", advance, tt.wantAdvance)
			}

			if !bytes.Equal(token, tt.wantToken) {
				t.Errorf("token = %q, want %q", token, tt.wantToken)
			}

			if err != tt.wantErr {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}