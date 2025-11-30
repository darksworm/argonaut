#!/bin/bash
# Mock k9s binary for E2E testing
#
# This script simulates k9s behavior for testing purposes.
# It records arguments to a file and exits with a configurable code.
#
# Environment variables:
#   MOCK_K9S_ARGS_FILE  - File to write received arguments (default: /tmp/mock_k9s_args)
#   MOCK_K9S_EXIT_CODE  - Exit code to return (default: 0)
#   MOCK_K9S_DURATION   - How long to run in seconds (default: 0.3)

# Write received arguments to a file for verification
ARGS_FILE="${MOCK_K9S_ARGS_FILE:-/tmp/mock_k9s_args}"
echo "$@" > "$ARGS_FILE"

# Duration to run (simulates k9s being open)
DURATION="${MOCK_K9S_DURATION:-0.3}"

# Exit code to return
EXIT_CODE="${MOCK_K9S_EXIT_CODE:-0}"

# Clear screen and show mock output (simulates k9s frame)
echo -e "\033[2J\033[H"
echo "=== Mock k9s ==="
echo "Args: $@"
echo ""
echo "This is a mock k9s for E2E testing."
echo "Press q or Ctrl+C to exit (auto-exits after ${DURATION}s)"

# Wait for configured duration
sleep "$DURATION"

exit "$EXIT_CODE"
