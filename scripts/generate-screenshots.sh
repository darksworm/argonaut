#!/bin/bash
# Automated script to launch ArgoCD, run the app, and generate screenshots for README
# Uses tmux-cli for terminal management and screencapture for taking screenshots

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCREENSHOTS_DIR="$PROJECT_DIR/assets/screenshots"
TMUX_SESSION="argonaut-demo"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date '+%H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date '+%H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date '+%H:%M:%S')] $1${NC}"
}

# Create screenshots directory
mkdir -p "$SCREENSHOTS_DIR"

# Array to store background freeze processes
declare -a FREEZE_PIDS=()
declare -a TEMP_FILES=()

# Fast mode flag
FAST_MODE=false

# Color remapping flag - set to true to remap bright colors to regular colors
REMAP_COLORS=false

# Function to capture tmux content and start freeze in background
take_screenshot() {
    local filename="$1"
    local description="$2"
    local session_pane="${3:-$TMUX_SESSION:argonaut-app}"  # Default to named window

    log "Capturing: $description"

    # Add .png extension if not present
    if [[ "$filename" != *.png ]]; then
        filename="${filename}.png"
    fi

    # Use freeze to capture the tmux pane with ANSI colors preserved
    if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        # Capture to temp file with unique name
        local temp_file="/tmp/tmux_capture_${filename}_$$"
        tmux capture-pane -pet "$session_pane" > "$temp_file"

        # Debug: show what we captured
        log "Captured $(wc -l < "$temp_file") lines for $filename"

        # Process colors if remapping is enabled
        local processed_file="$temp_file"
        if [[ "$REMAP_COLORS" == "true" ]]; then
            local remapped_file="/tmp/tmux_capture_remapped_${filename}_$$"
            python3 "$SCRIPT_DIR/ghostty_color_remap.py" "$temp_file" "$remapped_file"
            processed_file="$remapped_file"
            TEMP_FILES+=("$remapped_file")
        fi

        # Use termshot for screenshot generation
        termshot --raw-read "$processed_file" --filename "$SCREENSHOTS_DIR/$filename" &

        local freeze_pid=$!
        FREEZE_PIDS+=($freeze_pid)
        TEMP_FILES+=("$temp_file")

        log "Started freeze process $freeze_pid for $filename"
    else
        error "Tmux session $TMUX_SESSION not found"
        return 1
    fi
}

# Function to wait for all background freeze processes to complete
wait_for_screenshots() {
    log "Waiting for all screenshot renders to complete..."

    for pid in "${FREEZE_PIDS[@]}"; do
        if ps -p $pid > /dev/null 2>&1; then
            log "Waiting for freeze process $pid..."
            wait $pid
        fi
    done

    # Clean up temp files
    for temp_file in "${TEMP_FILES[@]}"; do
        rm -f "$temp_file"
    done

    log "All screenshots completed and temp files cleaned up"
}

# Function to wait for user input
wait_for_user() {
    local message="$1"
    echo -e "${BLUE}$message${NC}"
    read -p "Press Enter to continue..."
}

# Cleanup function
cleanup() {
    log "Cleaning up..."

    # Kill tmux session if it exists
    if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        tmux kill-session -t "$TMUX_SESSION"
        log "Killed tmux session: $TMUX_SESSION"
    fi

    # Note: ArgoCD is left running for continued use
    log "ArgoCD left running - use 'make argocd-down' to stop it manually if needed"
}

# Set trap for cleanup on script exit
trap cleanup EXIT

main() {
    log "Starting automated screenshot generation for Argonaut CLI"
    log "Project directory: $PROJECT_DIR"
    log "Screenshots will be saved to: $SCREENSHOTS_DIR"

    # Change to project directory
    cd "$PROJECT_DIR"

    # Step 1: Check ArgoCD is ready
    if [[ ! -f .argocd-portforward.pid ]]; then
        error "ArgoCD not running. Please start it first with: make argocd-up"
        exit 1
    fi

    log "ArgoCD detected - proceeding with app screenshots"

    # Step 2: Kill existing session if it exists, then create new one
    if tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        log "Killing existing tmux session: $TMUX_SESSION"
        tmux kill-session -t "$TMUX_SESSION"
    fi

    log "Creating new tmux session: $TMUX_SESSION"
    tmux new-session -d -s "$TMUX_SESSION" -c "$PROJECT_DIR" -n "argonaut-app" "bash"

    # Give tmux a moment to fully initialize
    sleep 1

    # Verify session was created
    if ! tmux has-session -t "$TMUX_SESSION" 2>/dev/null; then
        error "Failed to create tmux session"
        exit 1
    fi

    log "Tmux session created successfully with window 'argonaut-app'"

    # Step 3: Launch the TUI app directly with go run
    log "Launching TUI app with go run..."
    # Use full path to go binary
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "/etc/profiles/per-user/ilmars/bin/go run ./cmd/app" Enter
    sleep 5  # Wait for app to fully load

    # Zoom the pane for better screenshot quality
    log "Zooming tmux pane for better screenshot quality..."
    tmux resize-pane -t "$TMUX_SESSION:argonaut-app" -Z

    # Navigate through clusters, namespaces, projects to get to apps view
    log "Navigating to apps view (skipping cluster/namespace/project screenshots)..."

    # Navigate to namespaces view
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Enter"
    sleep 2

    # Navigate to projects view
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Enter"
    sleep 2

    # Navigate to apps view
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Enter"
    sleep 2
    take_screenshot "01-apps-view" "Applications view"

    # Step 7: Select one app and open rollback view first
    log "Selecting one app and opening rollback view..."
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "End"  # Go to last app
    sleep 0.5
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Space"  # Select the app
    sleep 0.5
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" ":rollback" "Enter"
    sleep 2
    take_screenshot "02-rollback-view" "Rollback view"

    # Exit rollback view and clear selection
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Escape"
    sleep 1
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "c"  # Clear selection
    sleep 0.5

    # Step 8: Select multiple apps for resource tree view
    log "Selecting multiple apps and opening resource tree view..."
    # Select first app
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Home"  # Go to first app
    sleep 0.5
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Space"
    sleep 0.5
    # Move down and select second app
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Down"
    sleep 0.5
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Space"
    sleep 0.5
    # Move down and select third app
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Down"
    sleep 0.5
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Space"
    sleep 0.5
    # Open resources view
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" ":resources" "Enter"
    sleep 3  # Tree view might take longer to load
    take_screenshot "03-tree-view" "Resource tree view"

    # Exit tree view and go back to apps view
    log "Exiting tree view and going back to apps view..."
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Escape"  # Exit tree view properly
    sleep 1
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "Backspace"  # Go back to apps view
    sleep 2

    # Show sync confirmation modal (apps should still be selected)
    log "Triggering sync confirmation modal..."
    tmux send-keys -t "$TMUX_SESSION:argonaut-app" "s"
    sleep 1
    take_screenshot "04-sync-confirmation" "Sync confirmation modal"

    # Wait for all background freeze processes to complete
    wait_for_screenshots

    log "All screenshots generated successfully!"
    log "Screenshots saved in: $SCREENSHOTS_DIR"
    log "Files created:"
    ls -1 "$SCREENSHOTS_DIR"/*.png 2>/dev/null | while read -r file; do
        echo "  - $(basename "$file")"
    done || echo "  (Screenshots still rendering...)"

    log "You can now use these screenshots in your README.md"
    log "Don't forget to open https://localhost:8080 to take a screenshot of the ArgoCD web UI as well!"
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    if ! command -v tmux &> /dev/null; then
        missing_deps+=("tmux")
    fi

    if ! command -v termshot &> /dev/null; then
        missing_deps+=("termshot (install with: cargo install termshot)")
    fi

    if ! command -v kubectl &> /dev/null; then
        missing_deps+=("kubectl")
    fi

    if ! command -v k3d &> /dev/null; then
        missing_deps+=("k3d")
    fi

    if ! command -v argocd &> /dev/null; then
        missing_deps+=("argocd")
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error "Missing required dependencies: ${missing_deps[*]}"
        error "Please install the missing dependencies and try again."
        exit 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Generate screenshots for Argonaut CLI README"
            echo ""
            echo "Options:"
            echo "  --help, -h     Show this help message"
            echo "  --fast         Fast mode - use termshot without styling"
            echo "  --remap-colors Remap ANSI colors to match cyberdream theme"
            echo "  --cleanup      Only run cleanup (kill sessions, stop ArgoCD)"
            echo ""
            echo "The script will:"
            echo "  1. Check ArgoCD is running"
            echo "  2. Create tmux session and launch TUI app"
            echo "  3. Navigate through app views automatically"
            echo "  4. Capture screenshots with freeze"
            echo "  5. Save screenshots to assets/screenshots/"
            echo ""
            echo "Screenshots generated:"
            echo "  01-apps-view.png"
            echo "  02-rollback-view.png"
            echo "  03-tree-view.png"
            echo "  04-sync-confirmation.png"
            exit 0
            ;;
        --fast)
            FAST_MODE=true
            ;;
        --remap-colors)
            REMAP_COLORS=true
            ;;
        --cleanup)
            cleanup
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            echo "Use --help for usage information."
            exit 1
            ;;
    esac
    shift
done

# Run the script
check_dependencies
main