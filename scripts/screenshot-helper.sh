#!/bin/bash
# Simple screenshot helper for manual workflow
# Takes a screenshot with window selection and saves to assets/screenshots/

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCREENSHOTS_DIR="$PROJECT_DIR/assets/screenshots"

# Create screenshots directory if it doesn't exist
mkdir -p "$SCREENSHOTS_DIR"

# Function to take screenshot using freeze
take_screenshot() {
    local filename="$1"
    local timestamp=$(date +"%Y%m%d_%H%M%S")

    if [[ -z "$filename" ]]; then
        filename="screenshot_${timestamp}"
    fi

    # Add .png extension if not present
    if [[ "$filename" != *.png ]]; then
        filename="${filename}.png"
    fi

    echo "📸 Taking screenshot: $filename"

    # Check if we're in a tmux session
    if [[ -n "$TMUX" ]]; then
        echo "📱 Detected tmux session - capturing current pane"

        # Get current tmux pane content
        local temp_file="/tmp/tmux_capture_$$"
        tmux capture-pane -p > "$temp_file"

        # Use freeze to create styled screenshot
        /Users/ilmars/go/bin/freeze "$temp_file" \
            --output "$SCREENSHOTS_DIR/$filename" \
            --theme "dracula" \
            --background \
            --margin "20" \
            --padding "20" \
            --border.radius "8" \
            --shadow.blur "20" \
            --shadow.x "0" \
            --shadow.y "10" \
            --font.size "14" \
            --width "120" \
            --height "30"

        # Clean up temp file
        rm -f "$temp_file"
    else
        echo "💻 Not in tmux - using interactive terminal capture"
        echo "🎯 Make sure your terminal is ready, then press Enter..."
        read -r

        # Use freeze with execute to capture current terminal state
        /Users/ilmars/go/bin/freeze \
            --execute "echo 'Terminal screenshot captured'" \
            --output "$SCREENSHOTS_DIR/$filename" \
            --theme "dracula" \
            --background \
            --margin "20" \
            --padding "20" \
            --border.radius "8" \
            --shadow.blur "20" \
            --shadow.x "0" \
            --shadow.y "10" \
            --font.size "14" \
            --width "120" \
            --height "30"
    fi

    if [[ -f "$SCREENSHOTS_DIR/$filename" ]]; then
        echo "✅ Screenshot saved: $SCREENSHOTS_DIR/$filename"

        # Show file size for reference
        local size=$(ls -lh "$SCREENSHOTS_DIR/$filename" | awk '{print $5}')
        echo "   File size: $size"

        # Option to open the screenshot
        read -p "Open screenshot? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            open "$SCREENSHOTS_DIR/$filename"
        fi
    else
        echo "❌ Failed to save screenshot"
        exit 1
    fi
}

# Main function
main() {
    echo "📸 Argonaut Screenshot Helper"
    echo "Screenshots will be saved to: $SCREENSHOTS_DIR"
    echo ""

    if [[ $# -eq 0 ]]; then
        # Interactive mode
        echo "Enter filename (without .png extension, or press Enter for timestamp):"
        read -r filename
        take_screenshot "$filename"
    else
        # Command line mode
        take_screenshot "$1"
    fi

    echo ""
    echo "📁 Current screenshots:"
    if ls "$SCREENSHOTS_DIR"/*.png >/dev/null 2>&1; then
        ls -la "$SCREENSHOTS_DIR"/*.png | while read -r line; do
            echo "   $(basename "$(echo "$line" | awk '{print $NF}')")"
        done
    else
        echo "   No screenshots found"
    fi
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        cat << EOF
Usage: $0 [filename]

Take a screenshot with window selection and save to assets/screenshots/

Options:
  --help, -h     Show this help
  --list, -l     List existing screenshots
  --open, -o     Open screenshots directory
  --clean        Remove all screenshots (with confirmation)

Examples:
  $0                    # Interactive mode, prompts for filename
  $0 argocd-setup       # Save as argocd-setup.png
  $0 app-list.png       # Save as app-list.png (extension optional)

The screenshot will be taken using macOS screencapture with window selection.
Click on the terminal window you want to capture when prompted.
EOF
        ;;
    --list|-l)
        echo "📁 Screenshots in $SCREENSHOTS_DIR:"
        if ls "$SCREENSHOTS_DIR"/*.png >/dev/null 2>&1; then
            ls -la "$SCREENSHOTS_DIR"/*.png
        else
            echo "No screenshots found"
        fi
        ;;
    --open|-o)
        open "$SCREENSHOTS_DIR"
        ;;
    --clean)
        if ls "$SCREENSHOTS_DIR"/*.png >/dev/null 2>&1; then
            echo "This will delete all screenshots in $SCREENSHOTS_DIR:"
            ls "$SCREENSHOTS_DIR"/*.png
            echo ""
            read -p "Are you sure? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                rm "$SCREENSHOTS_DIR"/*.png
                echo "✅ All screenshots deleted"
            else
                echo "❌ Cancelled"
            fi
        else
            echo "No screenshots to delete"
        fi
        ;;
    *)
        main "$@"
        ;;
esac