package api

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strconv"
)

// SSE buffer configuration defaults
const (
	DefaultInitialBuffer = 256 * 1024      // 256KB initial buffer
	DefaultMaxBuffer     = 16 * 1024 * 1024 // 16MB maximum buffer
	DefaultGrowthFactor  = 2.0              // Double the buffer on growth
)

// getSSEBufferConfig returns SSE buffer configuration from environment or defaults
func getSSEBufferConfig() (initial, max int, growth float64) {
	initial = DefaultInitialBuffer
	max = DefaultMaxBuffer
	growth = DefaultGrowthFactor

	if val := os.Getenv("ARGONAUT_SSE_INITIAL_BUFFER"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			initial = parsed
		}
	}

	if val := os.Getenv("ARGONAUT_SSE_MAX_BUFFER"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > initial {
			max = parsed
		}
	}

	if val := os.Getenv("ARGONAUT_SSE_BUFFER_GROWTH"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil && parsed > 1.0 {
			growth = parsed
		}
	}

	return
}

// scanSSELinesOrChunk is a custom split function for bufio.Scanner that handles SSE events.
// It tries to scan complete SSE events (ending with double newline) but will chunk
// large data to avoid the "token too long" error.
//
// This is inspired by the AlbinoDrought solution for handling bufio.ErrTooLong
func scanSSELinesOrChunk(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// First try normal line scanning for SSE
	// SSE events are terminated by a blank line (double newline)
	if idx := bytes.Index(data, []byte("\n\n")); idx >= 0 {
		// We found a complete SSE event
		return idx + 2, data[:idx+2], nil
	}

	// If we're at EOF and have data, return it
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}

	// Check if we're approaching the buffer limit
	// If the data is getting too large, chunk it to prevent ErrTooLong
	const maxChunkSize = 1024 * 1024 // 1MB chunks
	if len(data) >= maxChunkSize {
		// Find the last newline within the chunk to avoid splitting lines
		lastNewline := bytes.LastIndexByte(data[:maxChunkSize], '\n')
		if lastNewline > 0 {
			// Return data up to the last complete line
			return lastNewline + 1, data[:lastNewline+1], nil
		}
		// No newline found, return the chunk anyway to avoid blocking
		return maxChunkSize, data[:maxChunkSize], nil
	}

	// Need more data to find a complete SSE event
	return 0, nil, nil
}

// scanSSELines is a simpler split function that just looks for complete lines
// but handles large lines by chunking them
func scanSSELines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// First try standard line scanning
	advance, token, err = bufio.ScanLines(data, atEOF)
	if advance > 0 || token != nil || err != nil {
		return
	}

	// If buffer is getting large, chunk it to avoid ErrTooLong
	const maxLineLength = 512 * 1024 // 512KB max line length
	if len(data) >= maxLineLength {
		// Return a chunk to prevent buffer overflow
		return maxLineLength, data[:maxLineLength], nil
	}

	return 0, nil, nil
}

// createSSEScanner creates a new scanner configured for SSE streaming with progressive buffer sizing
func createSSEScanner(stream io.Reader, initialSize, maxSize int) *bufio.Scanner {
	scanner := bufio.NewScanner(stream)
	
	// Allocate initial buffer
	buf := make([]byte, initialSize)
	scanner.Buffer(buf, maxSize)
	
	// Use our custom split function that handles large SSE events
	scanner.Split(scanSSELinesOrChunk)
	
	return scanner
}