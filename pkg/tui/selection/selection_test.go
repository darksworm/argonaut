package selection

import (
	"testing"
)

func TestSelection_NewAndClear(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if !s.IsEmpty() {
		t.Error("New selection should be empty")
	}

	s.SetStart(Position{Row: 1, Col: 2})
	if s.IsEmpty() {
		t.Error("Selection should not be empty after SetStart")
	}

	s.Clear()
	if !s.IsEmpty() {
		t.Error("Selection should be empty after Clear")
	}
}

func TestSelection_SetStartAndEnd(t *testing.T) {
	s := New()
	s.SetStart(Position{Row: 1, Col: 5})

	if !s.Active {
		t.Error("Selection should be active after SetStart")
	}
	if s.Start.Row != 1 || s.Start.Col != 5 {
		t.Errorf("Start position wrong: got %+v", s.Start)
	}

	s.SetEnd(Position{Row: 3, Col: 10})
	if s.End.Row != 3 || s.End.Col != 10 {
		t.Errorf("End position wrong: got %+v", s.End)
	}
}

func TestSelection_Finalize(t *testing.T) {
	s := New()

	// Empty selection (start == end) should return false
	s.SetStart(Position{Row: 1, Col: 5})
	s.SetEnd(Position{Row: 1, Col: 5})
	if s.Finalize() {
		t.Error("Finalize should return false when start == end")
	}
	if s.HasContent {
		t.Error("HasContent should be false when start == end")
	}

	// Non-empty selection should return true
	s.SetStart(Position{Row: 1, Col: 5})
	s.SetEnd(Position{Row: 2, Col: 10})
	if !s.Finalize() {
		t.Error("Finalize should return true when start != end")
	}
	if !s.HasContent {
		t.Error("HasContent should be true when start != end")
	}
	if s.Active {
		t.Error("Selection should not be active after Finalize")
	}
}

func TestSelection_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		start    Position
		end      Position
		wantS    Position
		wantE    Position
	}{
		{
			name:  "already normalized",
			start: Position{Row: 1, Col: 5},
			end:   Position{Row: 3, Col: 10},
			wantS: Position{Row: 1, Col: 5},
			wantE: Position{Row: 3, Col: 10},
		},
		{
			name:  "inverted rows",
			start: Position{Row: 3, Col: 10},
			end:   Position{Row: 1, Col: 5},
			wantS: Position{Row: 1, Col: 5},
			wantE: Position{Row: 3, Col: 10},
		},
		{
			name:  "same row inverted cols",
			start: Position{Row: 2, Col: 15},
			end:   Position{Row: 2, Col: 5},
			wantS: Position{Row: 2, Col: 5},
			wantE: Position{Row: 2, Col: 15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			s.Start = tt.start
			s.End = tt.end
			gotS, gotE := s.Normalize()
			if gotS != tt.wantS {
				t.Errorf("Normalize() start = %+v, want %+v", gotS, tt.wantS)
			}
			if gotE != tt.wantE {
				t.Errorf("Normalize() end = %+v, want %+v", gotE, tt.wantE)
			}
		})
	}
}

func TestSelection_Contains(t *testing.T) {
	s := New()
	s.Start = Position{Row: 1, Col: 5}
	s.End = Position{Row: 3, Col: 10}
	s.HasContent = true

	tests := []struct {
		name string
		row  int
		col  int
		want bool
	}{
		{"before start row", 0, 5, false},
		{"start row before col", 1, 4, false},
		{"at start", 1, 5, true},
		{"middle of selection", 2, 7, true},
		{"at end", 3, 10, true},
		{"end row after col", 3, 11, false},
		{"after end row", 4, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Contains(tt.row, tt.col); got != tt.want {
				t.Errorf("Contains(%d, %d) = %v, want %v", tt.row, tt.col, got, tt.want)
			}
		})
	}
}

func TestSelection_ExtractText(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		start  Position
		end    Position
		want   string
	}{
		{
			name:  "single line partial",
			lines: []string{"Hello, World!"},
			start: Position{Row: 0, Col: 7},
			end:   Position{Row: 0, Col: 12},
			want:  "World",
		},
		{
			name:  "multi line",
			lines: []string{"Line 1", "Line 2", "Line 3"},
			start: Position{Row: 0, Col: 5},
			end:   Position{Row: 2, Col: 4},
			want:  "1\nLine 2\nLine",
		},
		{
			name:  "full line",
			lines: []string{"Test line"},
			start: Position{Row: 0, Col: 0},
			end:   Position{Row: 0, Col: 9},
			want:  "Test line",
		},
		{
			name:  "empty selection",
			lines: []string{"Test"},
			start: Position{Row: 0, Col: 2},
			end:   Position{Row: 0, Col: 2},
			want:  "",
		},
		{
			name:  "bounds beyond line length",
			lines: []string{"Short"},
			start: Position{Row: 0, Col: 0},
			end:   Position{Row: 0, Col: 100},
			want:  "Short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			s.Start = tt.start
			s.End = tt.end
			s.HasContent = tt.start != tt.end
			s.Active = tt.start != tt.end

			got := s.ExtractText(tt.lines)
			if got != tt.want {
				t.Errorf("ExtractText() = %q, want %q", got, tt.want)
			}
		})
	}
}
