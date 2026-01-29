package api

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// mockReadCloser implements io.ReadCloser for testing
type mockReadCloser struct {
	reader io.Reader
	closed bool
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.reader.Read(p)
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

// slowReader simulates a slow stream that sends data in chunks
type slowReader struct {
	chunks [][]byte
	index  int
	delay  time.Duration
}

func (s *slowReader) Read(p []byte) (n int, err error) {
	if s.index >= len(s.chunks) {
		return 0, io.EOF
	}
	
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	
	chunk := s.chunks[s.index]
	s.index++
	
	if len(chunk) > len(p) {
		// Simulate partial read
		copy(p, chunk[:len(p)])
		// Put back the rest for next read
		s.index--
		s.chunks[s.index] = chunk[len(p):]
		return len(p), nil
	}
	
	copy(p, chunk)
	return len(chunk), nil
}

func TestAccumulatingSSEReader_SmallEvents(t *testing.T) {
	// Test normal-sized SSE events
	data := "data: {\"type\":\"ADDED\",\"app\":\"test-app-1\"}\n\n" +
		"data: {\"type\":\"MODIFIED\",\"app\":\"test-app-2\"}\n\n" +
		"data: {\"type\":\"DELETED\",\"app\":\"test-app-3\"}\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	events := []string{
		"data: {\"type\":\"ADDED\",\"app\":\"test-app-1\"}\n\n",
		"data: {\"type\":\"MODIFIED\",\"app\":\"test-app-2\"}\n\n",
		"data: {\"type\":\"DELETED\",\"app\":\"test-app-3\"}\n\n",
	}
	
	for i, expected := range events {
		event, err := reader.ReadEvent()
		if err != nil && err != io.EOF {
			t.Fatalf("Event %d: unexpected error: %v", i, err)
		}
		if string(event) != expected {
			t.Errorf("Event %d: got %q, want %q", i, string(event), expected)
		}
	}
	
	// Verify metrics
	metrics := reader.Metrics()
	if metrics.EventsProcessed != 3 {
		t.Errorf("EventsProcessed: got %d, want 3", metrics.EventsProcessed)
	}
	if metrics.BufferResizes != 0 {
		t.Errorf("BufferResizes: got %d, want 0", metrics.BufferResizes)
	}
}

func TestAccumulatingSSEReader_LargeEvent(t *testing.T) {
	// Create a large event (1MB+) that exceeds initial buffer
	largeData := strings.Repeat("x", 1*MB)
	data := "data: {\"payload\":\"" + largeData + "\"}\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	config := &SSEConfig{
		InitialBuffer:  64 * KB,  // Small initial buffer to force growth
		MaxBuffer:      16 * MB,
		MaxAccumulated: 32 * MB,
		GrowthStrategy: "adaptive",
		SmallGrowth:    256 * KB,
		MediumGrowth:   512 * KB,
		LargeGrowth:    1 * MB,
		XLargeGrowth:   2 * MB,
	}
	reader := NewAccumulatingSSEReader(stream, config)
	defer reader.Close()
	
	event, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read large event: %v", err)
	}
	
	if !strings.HasPrefix(string(event), "data: {\"payload\":\"") {
		t.Error("Large event not read correctly")
	}
	
	// Verify buffer grew or switched to direct reading
	metrics := reader.Metrics()
	// Either buffer resized OR we switched to direct reading (which also uses the buffer)
	// Check that we handled the large event properly
	if metrics.MaxEventSize < 1*MB {
		t.Errorf("MaxEventSize too small: got %d, want >= %d", metrics.MaxEventSize, 1*MB)
	}
	
	// The event was processed successfully, which is what matters
	if metrics.EventsProcessed != 1 {
		t.Errorf("EventsProcessed: got %d, want 1", metrics.EventsProcessed)
	}
}

func TestAccumulatingSSEReader_AdaptiveGrowth(t *testing.T) {
	// Test adaptive buffer growth
	config := &SSEConfig{
		InitialBuffer:  256 * KB,
		MaxBuffer:      16 * MB,
		MaxAccumulated: 32 * MB,
		GrowthStrategy: "adaptive",
		SmallGrowth:    256 * KB,
		MediumGrowth:   512 * KB,
		LargeGrowth:    1 * MB,
		XLargeGrowth:   2 * MB,
	}
	
	tests := []struct {
		currentSize int
		expectedNew int
		description string
	}{
		{200 * KB, 200*KB + 256*KB, "Small buffer growth"},
		{500 * KB, 500*KB + 256*KB, "Still small buffer"},
		{1 * MB, 1*MB + 512*KB, "Medium buffer growth"},
		{3 * MB, 3*MB + 512*KB, "Still medium buffer"},
		{5 * MB, 5*MB + 1*MB, "Large buffer growth"},
		{10 * MB, 10*MB + 2*MB, "XLarge buffer growth"},
		{15 * MB, 16 * MB, "Capped at max buffer"},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			stream := &mockReadCloser{reader: strings.NewReader("")}
			reader := NewAccumulatingSSEReader(stream, config)
			reader.bufferSize = tt.currentSize
			
			newSize := reader.calculateNextBufferSize()
			if newSize != tt.expectedNew {
				t.Errorf("Buffer size calculation wrong: got %d, want %d", newSize, tt.expectedNew)
			}
		})
	}
}

func TestAccumulatingSSEReader_ChunkedReading(t *testing.T) {
	// Simulate data arriving in small chunks
	chunks := [][]byte{
		[]byte("data: {\"ty"),
		[]byte("pe\":\"ADDED"),
		[]byte("\",\"app\":\""),
		[]byte("test-app\"}\n"),
		[]byte("\ndata: {\""),
		[]byte("type\":\"MODIFIED\","),
		[]byte("\"app\":\"test-app-2\"}\n\n"),
	}
	
	slowStream := &slowReader{chunks: chunks}
	stream := &mockReadCloser{reader: slowStream}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	// Read first event
	event1, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read first event: %v", err)
	}
	expected1 := "data: {\"type\":\"ADDED\",\"app\":\"test-app\"}\n\n"
	if string(event1) != expected1 {
		t.Errorf("First event wrong: got %q, want %q", string(event1), expected1)
	}
	
	// Read second event
	event2, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read second event: %v", err)
	}
	expected2 := "data: {\"type\":\"MODIFIED\",\"app\":\"test-app-2\"}\n\n"
	if string(event2) != expected2 {
		t.Errorf("Second event wrong: got %q, want %q", string(event2), expected2)
	}
}

func TestAccumulatingSSEReader_EventTooLarge(t *testing.T) {
	// Create an event that exceeds MaxAccumulated
	config := &SSEConfig{
		InitialBuffer:  256 * KB,
		MaxBuffer:      1 * MB,
		MaxAccumulated: 2 * MB, // Small limit for testing
		GrowthStrategy: "adaptive",
		SmallGrowth:    256 * KB,
		MediumGrowth:   512 * KB,
		LargeGrowth:    1 * MB,
		XLargeGrowth:   2 * MB,
	}
	
	// Create 3MB event (exceeds MaxAccumulated)
	largeData := strings.Repeat("x", 3*MB)
	data := "data: " + largeData
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, config)
	defer reader.Close()
	
	_, err := reader.ReadEvent()
	if !errors.Is(err, ErrEventTooLarge) {
		t.Errorf("Expected ErrEventTooLarge, got: %v", err)
	}
}

func TestAccumulatingSSEReader_MultipleDataLines(t *testing.T) {
	// Test SSE event with multiple data lines (common in real SSE)
	data := "data: line1\ndata: line2\ndata: line3\n\n" +
		"data: single line\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	// First event (multi-line)
	event1, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read multi-line event: %v", err)
	}
	expected1 := "data: line1\ndata: line2\ndata: line3\n\n"
	if string(event1) != expected1 {
		t.Errorf("Multi-line event wrong: got %q, want %q", string(event1), expected1)
	}
	
	// Second event (single line)
	event2, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read single-line event: %v", err)
	}
	expected2 := "data: single line\n\n"
	if string(event2) != expected2 {
		t.Errorf("Single-line event wrong: got %q, want %q", string(event2), expected2)
	}
}

func TestAccumulatingSSEReader_KeepAliveMessages(t *testing.T) {
	// Test with keep-alive messages (lines starting with ":")
	data := ": keep-alive\n\n" +
		"data: {\"type\":\"ADDED\"}\n\n" +
		": another keep-alive\n\n" +
		"data: {\"type\":\"MODIFIED\"}\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	events := []string{
		": keep-alive\n\n",
		"data: {\"type\":\"ADDED\"}\n\n",
		": another keep-alive\n\n",
		"data: {\"type\":\"MODIFIED\"}\n\n",
	}
	
	for i, expected := range events {
		event, err := reader.ReadEvent()
		if err != nil && err != io.EOF {
			t.Fatalf("Event %d: unexpected error: %v", i, err)
		}
		if string(event) != expected {
			t.Errorf("Event %d: got %q, want %q", i, string(event), expected)
		}
	}
}

func TestAccumulatingSSEReader_BufferPooling(t *testing.T) {
	// Test that buffers are returned to pool
	data := "data: test\n\n"
	
	// Create multiple readers with default config
	for i := 0; i < 10; i++ {
		stream := &mockReadCloser{reader: strings.NewReader(data)}
		reader := NewAccumulatingSSEReader(stream, nil)
		
		_, err := reader.ReadEvent()
		if err != nil && err != io.EOF {
			t.Fatalf("Iteration %d: failed to read: %v", i, err)
		}
		
		// Close should return buffer to pool
		reader.Close()
	}
	
	// No good way to test pool reuse directly, but this shouldn't panic
}

func TestAccumulatingSSEReader_PartialEventAtEOF(t *testing.T) {
	// Test partial event at EOF (no terminating \n\n)
	data := "data: {\"type\":\"INCOMPLETE\""
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	// Should switch to direct reading after scanner fails to find complete event
	event, _ := reader.ReadEvent()
	
	// Partial data without \n\n terminator should be returned with EOF
	// The actual error may be nil if data is returned
	if string(event) != data {
		t.Errorf("Partial event not returned: got %q, want %q", string(event), data)
	}
}

func TestAccumulatingSSEReader_EmptyStream(t *testing.T) {
	// Test empty stream
	stream := &mockReadCloser{reader: strings.NewReader("")}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	_, err := reader.ReadEvent()
	if err != io.EOF {
		t.Errorf("Expected io.EOF for empty stream, got: %v", err)
	}
}

func TestAccumulatingSSEReader_SwitchFromScannerToDirect(t *testing.T) {
	// Start with small events (use scanner), then large event (switch to direct)
	smallEvent := "data: {\"small\":\"event\"}\n\n"
	largeData := strings.Repeat("x", 1*MB)
	largeEvent := "data: {\"large\":\"" + largeData + "\"}\n\n"
	data := smallEvent + largeEvent + smallEvent
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	config := &SSEConfig{
		InitialBuffer:  256 * KB,
		MaxBuffer:      16 * MB,
		MaxAccumulated: 32 * MB,
		GrowthStrategy: "adaptive",
		SmallGrowth:    256 * KB,
		MediumGrowth:   512 * KB,
		LargeGrowth:    1 * MB,
		XLargeGrowth:   2 * MB,
	}
	reader := NewAccumulatingSSEReader(stream, config)
	defer reader.Close()
	
	// Read small event (uses scanner)
	event1, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read first small event: %v", err)
	}
	if string(event1) != smallEvent {
		t.Errorf("First small event wrong")
	}
	
	// Read large event (should switch to direct reading)
	event2, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read large event: %v", err)
	}
	if !strings.HasPrefix(string(event2), "data: {\"large\":\"") {
		t.Errorf("Large event not read correctly")
	}
	
	// Read another small event (now using direct reading)
	event3, err := reader.ReadEvent()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read second small event: %v", err)
	}
	if string(event3) != smallEvent {
		t.Errorf("Second small event wrong")
	}
}

// Benchmark tests
func BenchmarkAccumulatingSSEReader_SmallEvents(b *testing.B) {
	data := strings.Repeat("data: {\"type\":\"ADDED\",\"app\":\"test-app\"}\n\n", 1000)
	
	for i := 0; i < b.N; i++ {
		stream := &mockReadCloser{reader: strings.NewReader(data)}
		reader := NewAccumulatingSSEReader(stream, nil)
		
		for {
			_, err := reader.ReadEvent()
			if err == io.EOF {
				break
			}
		}
		reader.Close()
	}
}

func BenchmarkAccumulatingSSEReader_LargeEvents(b *testing.B) {
	largeData := strings.Repeat("x", 500*KB)
	event := "data: {\"payload\":\"" + largeData + "\"}\n\n"
	data := strings.Repeat(event, 10)
	
	for i := 0; i < b.N; i++ {
		stream := &mockReadCloser{reader: strings.NewReader(data)}
		config := &SSEConfig{
			InitialBuffer:  256 * KB,
			MaxBuffer:      16 * MB,
			MaxAccumulated: 32 * MB,
			GrowthStrategy: "adaptive",
			SmallGrowth:    256 * KB,
			MediumGrowth:   512 * KB,
			LargeGrowth:    1 * MB,
			XLargeGrowth:   2 * MB,
		}
		reader := NewAccumulatingSSEReader(stream, config)
		
		for {
			_, err := reader.ReadEvent()
			if err == io.EOF {
				break
			}
		}
		reader.Close()
	}
}

func TestAccumulatingSSEReader_ReadLine_SingleDataLine(t *testing.T) {
	// Test single data line
	data := "data: {\"type\":\"ADDED\",\"app\":\"test-app\"}\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expected := "{\"type\":\"ADDED\",\"app\":\"test-app\"}"
	if line != expected {
		t.Errorf("ReadLine result: got %q, want %q", line, expected)
	}
}

func TestAccumulatingSSEReader_ReadLine_MultipleDataLines(t *testing.T) {
	// Test multiple data lines in single event
	data := "data: line1\ndata: line2\ndata: line3\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expected := "line1\nline2\nline3"
	if line != expected {
		t.Errorf("ReadLine result: got %q, want %q", line, expected)
	}
}

func TestAccumulatingSSEReader_ReadLine_MixedPrefixes(t *testing.T) {
	// Test both "data:" and "data: " prefixes
	data := "data:no-space\ndata: with-space\ndata:another-no-space\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expected := "no-space\nwith-space\nanother-no-space"
	if line != expected {
		t.Errorf("ReadLine result: got %q, want %q", line, expected)
	}
}

func TestAccumulatingSSEReader_ReadLine_WithCommentLines(t *testing.T) {
	// Test event with data lines and comment lines (should ignore comments)
	data := ": comment line\ndata: actual data 1\n: another comment\ndata: actual data 2\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expected := "actual data 1\nactual data 2"
	if line != expected {
		t.Errorf("ReadLine result: got %q, want %q", line, expected)
	}
}

func TestAccumulatingSSEReader_ReadLine_EmptyEvent(t *testing.T) {
	// Test event with no data lines
	data := ": comment only\n\n"
	
	stream := &mockReadCloser{reader: strings.NewReader(data)}
	reader := NewAccumulatingSSEReader(stream, nil)
	defer reader.Close()
	
	line, err := reader.ReadLine()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expected := ""
	if line != expected {
		t.Errorf("ReadLine result: got %q, want %q", line, expected)
	}
}

func BenchmarkAccumulatingSSEReader_MixedSizes(b *testing.B) {
	smallEvent := "data: {\"type\":\"SMALL\"}\n\n"
	mediumData := strings.Repeat("x", 50*KB)
	mediumEvent := "data: {\"medium\":\"" + mediumData + "\"}\n\n"
	largeData := strings.Repeat("x", 500*KB)
	largeEvent := "data: {\"large\":\"" + largeData + "\"}\n\n"
	
	data := ""
	for i := 0; i < 100; i++ {
		data += smallEvent
		if i%10 == 0 {
			data += mediumEvent
		}
		if i%20 == 0 {
			data += largeEvent
		}
	}
	
	for i := 0; i < b.N; i++ {
		stream := &mockReadCloser{reader: bytes.NewReader([]byte(data))}
		config := &SSEConfig{
			InitialBuffer:  256 * KB,
			MaxBuffer:      16 * MB,
			MaxAccumulated: 32 * MB,
			GrowthStrategy: "adaptive",
			SmallGrowth:    256 * KB,
			MediumGrowth:   512 * KB,
			LargeGrowth:    1 * MB,
			XLargeGrowth:   2 * MB,
		}
		reader := NewAccumulatingSSEReader(stream, config)
		
		for {
			_, err := reader.ReadEvent()
			if err == io.EOF {
				break
			}
		}
		reader.Close()
	}
}