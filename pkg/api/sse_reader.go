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
	bufReader   *bufio.Reader  // Unified reader for both scanning and direct reads
	useScanner  bool           // Whether to use scanner mode
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
	
	// Create unified buffer reader to prevent data loss during transitions
	reader.bufReader = bufio.NewReaderSize(stream, config.InitialBuffer)
	
	return reader
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

// tryScanner attempts to read an event using scanner-like logic with bufio.Reader
func (r *AccumulatingSSEReader) tryScanner() ([]byte, error) {
	if !r.useScanner {
		return nil, nil
	}
	
	// Read event using bufio.Reader directly to avoid data loss
	event, err := r.readEventFromBufReader()
	if err != nil {
		// Check for buffer overflow - switch to direct reading mode
		if err == ErrBufferOverflow {
			r.useScanner = false
			// Try to grow buffer for direct reading
			if r.bufferSize < r.config.MaxBuffer {
				r.growBuffer()
			}
			return nil, nil
		}
		return nil, err
	}
	
	if event != nil {
		r.metrics.EventsProcessed++
		if len(event) > r.metrics.MaxEventSize {
			r.metrics.MaxEventSize = len(event)
		}
	}
	
	return event, nil
}

// readEventFromBufReader reads an event using the bufio.Reader for consistent buffering
func (r *AccumulatingSSEReader) readEventFromBufReader() ([]byte, error) {
	// If we have leftover data from previous event, check it first
	if len(r.accumulated) > 0 {
		if idx := bytes.Index(r.accumulated, []byte("\n\n")); idx >= 0 {
			event := make([]byte, idx+2)
			copy(event, r.accumulated[:idx+2])
			
			// Keep remainder for next event
			remainder := r.accumulated[idx+2:]
			r.accumulated = r.accumulated[:0]
			if len(remainder) > 0 {
				r.accumulated = append(r.accumulated, remainder...)
			}
			
			return event, nil
		}
	}
	
	for {
		n, err := r.bufReader.Read(r.buffer)
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
				
				return event, nil
			}
			
			// Check if we're accumulating too much for scanner mode
			if r.useScanner && len(r.accumulated) > r.config.MaxBuffer/4 {
				return nil, ErrBufferOverflow
			}
			
			// Check accumulated size limits
			if len(r.accumulated) > r.config.MaxAccumulated {
				return nil, fmt.Errorf("%w: accumulated %d MB exceeds max %d MB",
					ErrEventTooLarge, len(r.accumulated)/MB, r.config.MaxAccumulated/MB)
			}
		}
		
		if err != nil {
			if err == io.EOF {
				if len(r.accumulated) > 0 {
					// Return accumulated data as final event
					event := r.accumulated
					r.accumulated = r.accumulated[:0] // Clear properly
					return event, nil // Return nil error so data is accessible
				}
				return nil, io.EOF
			}
			return nil, err
		}
	}
}

// readDirectUntilEventEnd reads directly for large events using the same bufio.Reader
func (r *AccumulatingSSEReader) readDirectUntilEventEnd() ([]byte, error) {
	// Use the same bufReader to maintain consistency and prevent data loss
	return r.readEventFromBufReader()
}

// ReadEvent reads the next SSE event, handling large events gracefully
func (r *AccumulatingSSEReader) ReadEvent() ([]byte, error) {
	// First try scanner-like reading for normal-sized events
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
		// Scanner mode failed with buffer overflow, switched to direct reading
	}
	
	// Use direct reading for large events (still uses same bufReader)
	return r.readDirectUntilEventEnd()
}

// ReadLine reads and concatenates all data lines from an SSE event
func (r *AccumulatingSSEReader) ReadLine() (string, error) {
	// Read the full SSE event
	event, err := r.ReadEvent()
	if err != nil {
		return "", err
	}
	
	// Split into lines and collect all data lines
	lines := bytes.Split(event, []byte("\n"))
	var dataLines []string
	
	for _, line := range lines {
		// Handle both "data:" and "data: " prefixes
		if bytes.HasPrefix(line, []byte("data:")) {
			// Strip "data:" prefix
			dataContent := bytes.TrimPrefix(line, []byte("data:"))
			// Remove single leading space if present
			if len(dataContent) > 0 && dataContent[0] == ' ' {
				dataContent = dataContent[1:]
			}
			dataLines = append(dataLines, string(dataContent))
		}
	}
	
	// Join all data lines with newlines
	if len(dataLines) > 0 {
		return strings.Join(dataLines, "\n"), nil
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