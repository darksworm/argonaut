package api

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Size constants
const (
	KB = 1024
	MB = 1024 * KB
	
	// Buffer size defaults
	DefaultInitialBufferV2 = 256 * KB   // Start with 256KB
	DefaultMaxBufferV2    = 16 * MB     // Maximum single buffer size
	DefaultMaxAccumulated = 32 * MB     // Maximum accumulated event size
	
	// Growth increments for adaptive strategy
	SmallGrowthIncrement  = 256 * KB    // For buffers < 1MB
	MediumGrowthIncrement = 512 * KB    // For buffers 1-4MB
	LargeGrowthIncrement  = 1 * MB      // For buffers 4-8MB
	XLargeGrowthIncrement = 2 * MB      // For buffers > 8MB
)

// Common SSE error types
var (
	ErrEventTooLarge  = errors.New("SSE event exceeds maximum size")
	ErrBufferOverflow = errors.New("buffer overflow, growing")
	ErrStreamClosed   = errors.New("SSE stream closed")
)

// SSEConfig holds configuration for SSE processing
type SSEConfig struct {
	InitialBuffer  int    // Start size (default 256KB)
	MaxBuffer      int    // Max single buffer (default 16MB)
	MaxAccumulated int    // Max accumulated event (default 32MB)
	GrowthStrategy string // "adaptive" (default) or "fixed"
	
	// Adaptive growth sizes
	SmallGrowth  int // 256KB for < 1MB
	MediumGrowth int // 512KB for 1-4MB
	LargeGrowth  int // 1MB for 4-8MB
	XLargeGrowth int // 2MB for > 8MB
}

// DefaultSSEConfig returns the default SSE configuration
func DefaultSSEConfig() *SSEConfig {
	return &SSEConfig{
		InitialBuffer:  DefaultInitialBufferV2,
		MaxBuffer:      DefaultMaxBufferV2,
		MaxAccumulated: DefaultMaxAccumulated,
		GrowthStrategy: "adaptive",
		SmallGrowth:    SmallGrowthIncrement,
		MediumGrowth:   MediumGrowthIncrement,
		LargeGrowth:    LargeGrowthIncrement,
		XLargeGrowth:   XLargeGrowthIncrement,
	}
}

// SSEMetrics tracks SSE processing metrics
type SSEMetrics struct {
	BufferResizes    int
	MaxEventSize     int
	CurrentBuffer    int
	AccumulatedBytes int
	EventsProcessed  int
}

// Buffer pool for reducing GC pressure
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, DefaultInitialBufferV2)
	},
}

// AccumulatingSSEReader reads SSE events without data loss on buffer overflow
type AccumulatingSSEReader struct {
	stream      io.ReadCloser
	buffer      []byte     // Current read buffer
	accumulated []byte     // Partial event accumulator
	bufferSize  int        // Current buffer size
	config      *SSEConfig
	metrics     *SSEMetrics
	scanner     *bufio.Scanner // For normal-sized events
	useScanner  bool           // Whether to use scanner
}

// NewAccumulatingSSEReader creates a new SSE reader that handles large events gracefully
func NewAccumulatingSSEReader(stream io.ReadCloser, config *SSEConfig) *AccumulatingSSEReader {
	if config == nil {
		config = DefaultSSEConfig()
	}
	
	// Get buffer from pool if it matches default size
	var buffer []byte
	if config.InitialBuffer == DefaultInitialBufferV2 {
		buffer = bufferPool.Get().([]byte)
	} else {
		buffer = make([]byte, config.InitialBuffer)
	}
	
	reader := &AccumulatingSSEReader{
		stream:      stream,
		buffer:      buffer,
		accumulated: make([]byte, 0, config.InitialBuffer),
		bufferSize:  config.InitialBuffer,
		config:      config,
		metrics: &SSEMetrics{
			CurrentBuffer: config.InitialBuffer,
		},
		useScanner: true,
	}
	
	// Create scanner for normal-sized events
	reader.scanner = bufio.NewScanner(stream)
	reader.scanner.Buffer(make([]byte, config.InitialBuffer), config.MaxBuffer)
	reader.scanner.Split(scanSSEEvents)
	
	return reader
}

// scanSSEEvents is a split function for SSE events (double newline terminated)
func scanSSEEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Look for double newline (SSE event boundary)
	if idx := bytes.Index(data, []byte("\n\n")); idx >= 0 {
		return idx + 2, data[:idx+2], nil
	}
	
	// If at EOF with data, return it
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	
	// Need more data
	return 0, nil, nil
}

// calculateNextBufferSize determines the next buffer size using adaptive growth
func (r *AccumulatingSSEReader) calculateNextBufferSize() int {
	current := r.bufferSize
	
	var increment int
	switch {
	case current < 1*MB:
		increment = r.config.SmallGrowth
	case current < 4*MB:
		increment = r.config.MediumGrowth
	case current < 8*MB:
		increment = r.config.LargeGrowth
	default:
		increment = r.config.XLargeGrowth
	}
	
	newSize := current + increment
	if newSize > r.config.MaxBuffer {
		newSize = r.config.MaxBuffer
	}
	
	return newSize
}

// growBuffer increases the buffer size adaptively
func (r *AccumulatingSSEReader) growBuffer() error {
	if r.bufferSize >= r.config.MaxBuffer {
		return fmt.Errorf("%w: current size %d MB, max %d MB", 
			ErrEventTooLarge, r.bufferSize/MB, r.config.MaxBuffer/MB)
	}
	
	newSize := r.calculateNextBufferSize()
	
	// Return old buffer to pool if it was default size
	if len(r.buffer) == DefaultInitialBufferV2 {
		bufferPool.Put(r.buffer)
	}
	
	r.buffer = make([]byte, newSize)
	r.bufferSize = newSize
	r.metrics.BufferResizes++
	r.metrics.CurrentBuffer = newSize
	
	return nil
}

// tryScanner attempts to read an event using the scanner
func (r *AccumulatingSSEReader) tryScanner() ([]byte, error) {
	if !r.useScanner {
		return nil, nil
	}
	
	if r.scanner.Scan() {
		event := r.scanner.Bytes()
		r.metrics.EventsProcessed++
		if len(event) > r.metrics.MaxEventSize {
			r.metrics.MaxEventSize = len(event)
		}
		// Make a copy since scanner reuses the buffer
		result := make([]byte, len(event))
		copy(result, event)
		return result, nil
	}
	
	if err := r.scanner.Err(); err != nil {
		// Check for buffer overflow
		if err == bufio.ErrTooLong || strings.Contains(err.Error(), "token too long") {
			// Switch to direct reading mode
			r.useScanner = false
			// Try to grow buffer for direct reading
			if r.bufferSize < r.config.MaxBuffer {
				r.growBuffer()
			}
			return nil, nil
		}
		return nil, err
	}
	
	// EOF reached
	return nil, io.EOF
}

// readDirectUntilEventEnd reads directly from stream for large events
func (r *AccumulatingSSEReader) readDirectUntilEventEnd() ([]byte, error) {
	for {
		n, err := r.stream.Read(r.buffer)
		if n > 0 {
			r.accumulated = append(r.accumulated, r.buffer[:n]...)
			r.metrics.AccumulatedBytes = len(r.accumulated)
			
			// Check for event boundary
			if idx := bytes.Index(r.accumulated, []byte("\n\n")); idx >= 0 {
				event := make([]byte, idx+2)
				copy(event, r.accumulated[:idx+2])
				
				// Keep remainder for next event
				remainder := r.accumulated[idx+2:]
				r.accumulated = r.accumulated[:0]
				if len(remainder) > 0 {
					r.accumulated = append(r.accumulated, remainder...)
				}
				
				r.metrics.EventsProcessed++
				if len(event) > r.metrics.MaxEventSize {
					r.metrics.MaxEventSize = len(event)
				}
				
				return event, nil
			}
			
			// Check accumulated size limits
			if len(r.accumulated) > r.config.MaxAccumulated {
				return nil, fmt.Errorf("%w: accumulated %d MB exceeds max %d MB",
					ErrEventTooLarge, len(r.accumulated)/MB, r.config.MaxAccumulated/MB)
			}
			
			// Try to grow buffer if we're accumulating a lot
			if len(r.accumulated) > r.bufferSize && r.bufferSize < r.config.MaxBuffer {
				if err := r.growBuffer(); err == nil {
					// Successfully grew buffer
					continue
				}
			}
		}
		
		if err != nil {
			if err == io.EOF && len(r.accumulated) > 0 {
				// Return accumulated data as final event
				event := r.accumulated
				r.accumulated = nil
				return event, io.EOF
			}
			return nil, err
		}
	}
}

// ReadEvent reads the next SSE event, handling large events gracefully
func (r *AccumulatingSSEReader) ReadEvent() ([]byte, error) {
	// First try scanner for normal-sized events
	if r.useScanner {
		event, err := r.tryScanner()
		if event != nil || (err != nil && err != io.EOF) {
			return event, err
		}
		if err == io.EOF {
			// Check if we have accumulated data to return
			if len(r.accumulated) > 0 {
				event := r.accumulated
				r.accumulated = nil
				return event, io.EOF
			}
			return nil, io.EOF
		}
		// Scanner failed with buffer overflow, switch to direct reading
	}
	
	// Fall back to direct reading for large events
	return r.readDirectUntilEventEnd()
}

// ReadLine reads a single line (for line-by-line processing)
func (r *AccumulatingSSEReader) ReadLine() (string, error) {
	// This is a simplified version for reading single lines
	// Used when processing SSE data line by line
	event, err := r.ReadEvent()
	if err != nil {
		return "", err
	}
	
	// Split into lines and return first data line
	lines := bytes.Split(event, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("data: ")) {
			return string(bytes.TrimPrefix(line, []byte("data: "))), nil
		}
	}
	
	return "", nil
}

// Close closes the underlying stream and returns buffer to pool
func (r *AccumulatingSSEReader) Close() error {
	if len(r.buffer) == DefaultInitialBufferV2 {
		bufferPool.Put(r.buffer)
	}
	if r.stream != nil {
		return r.stream.Close()
	}
	return nil
}

// Metrics returns the current SSE processing metrics
func (r *AccumulatingSSEReader) Metrics() SSEMetrics {
	return *r.metrics
}

// StreamResponse contains the SSE stream and response headers
type StreamResponse struct {
	Body    io.ReadCloser
	Headers map[string][]string
}