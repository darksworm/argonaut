# ADR-0001: Custom SSE Reader Implementation

## Status

Accepted

## Context

Argonaut integrates with ArgoCD's Server-Sent Events (SSE) API endpoints for real-time streaming of application changes and resource tree updates. During implementation, we encountered critical issues with large SSE events that existing Go SSE libraries could not handle adequately:

### The Problem

1. **Large Payload Size**: ArgoCD sends SSE events up to 10-32MB containing complete application manifests and resource trees
2. **Authentication Constraints**: Reconnection on buffer overflow causes false authentication errors in ArgoCD
3. **Data Loss**: Standard `bufio.Scanner` approach loses buffered data when switching to direct reading for large events
4. **Performance Requirements**: Need to handle high-throughput streams without memory leaks

### Standard Library Analysis

We evaluated popular Go SSE libraries:

**r3labs/sse:**
- Default 64KB buffer limit
- Hard fails or reconnects on buffer overflow
- Designed for typical web payloads (< 64KB)

**tmaxmax/go-sse:**
- Default 64KB buffer limit  
- Configurable via `Buffer(buf []byte, maxSize int)`
- Still relies on reconnection for error recovery

### Architectural Constraints

Our specific requirements that standard libraries don't address:

1. **Zero reconnection**: Authentication issues prevent reconnection on overflow
2. **Massive payload support**: Must handle 32MB events without failure
3. **No data loss**: Scanner-to-direct-read transitions must preserve all buffered data
4. **Memory efficiency**: Buffer pooling and adaptive growth to minimize GC pressure

## Decision

We will implement a custom `AccumulatingSSEReader` with the following characteristics:

### Core Features

1. **Adaptive Buffer Growth**: Starts at 256KB, grows incrementally (256KB → 2MB) based on event size
2. **Hybrid Reading Strategy**: 
   - Use `bufio.Reader` for consistent buffering (no data loss)
   - Scanner mode for normal events (< buffer threshold)
   - Direct reading mode for large events (> buffer threshold)
3. **Buffer Pooling**: Reuse buffers to reduce GC pressure
4. **Zero Data Loss**: Unified `bufio.Reader` prevents data loss during mode transitions
5. **Comprehensive Error Handling**: Clear error types (`ErrEventTooLarge`, `ErrBufferOverflow`)

### Technical Implementation

```go
type AccumulatingSSEReader struct {
    stream      io.ReadCloser
    buffer      []byte
    accumulated []byte
    bufReader   *bufio.Reader  // Unified reader prevents data loss
    config      *SSEConfig
    useScanner  bool
}
```

### Configuration Options

- **InitialBuffer**: 256KB starting size
- **MaxBuffer**: 16MB maximum single buffer
- **MaxAccumulated**: 32MB maximum event size
- **Growth Strategy**: Adaptive increments (256KB, 512KB, 1MB, 2MB)

## Consequences

### Positive

- **Eliminates false authentication errors** from reconnection
- **Handles 32MB events** without failure or data loss
- **Memory efficient** with buffer pooling and adaptive growth
- **High performance** with hybrid reading strategy
- **Comprehensive metrics** for monitoring and debugging

### Negative

- **Custom code maintenance**: ~350 lines of SSE handling code vs library dependency
- **Testing complexity**: Extensive test coverage required (587 lines of tests)
- **Domain expertise**: Team needs to understand SSE specification details

### Alternatives Considered

1. **Patch existing libraries**: Would require complex modifications to disable reconnection and handle large buffers
2. **Use multiple libraries**: Complexity of integrating different libraries for different event sizes
3. **Request ArgoCD changes**: Not feasible - would require upstream changes affecting all ArgoCD users

## Implementation Details

### File Structure

- `pkg/api/sse_reader.go`: Core implementation (353 lines)
- `pkg/api/sse_reader_test.go`: Comprehensive tests (587 lines)
- `pkg/api/applications.go`: Integration with ArgoCD streaming endpoints

### Memory Characteristics

- **Initial memory**: 256KB buffer
- **Maximum memory**: 32MB for largest events  
- **GC efficiency**: Buffer pooling for default-sized buffers
- **Adaptive growth**: Prevents excessive memory allocation for small events

### Error Handling

- `ErrEventTooLarge`: Event exceeds 32MB limit
- `ErrBufferOverflow`: Internal buffer management error
- Graceful EOF handling for partial events

## Notes

This decision was made after discovering that our ~1,224 lines of custom SSE code (including tests) addresses specific architectural constraints that would be complex to achieve with existing libraries. The custom implementation is justified by the unique requirements of ArgoCD integration where reconnection is not an acceptable failure mode.

## References

- [ArgoCD API Documentation](https://argo-cd.readthedocs.io/en/stable/developer-guide/api-docs/)
- [HTML5 SSE Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [r3labs/sse Library](https://github.com/r3labs/sse)
- [tmaxmax/go-sse Library](https://github.com/tmaxmax/go-sse)
- Issue #189: False authentication errors on large SSE events