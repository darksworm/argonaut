#!/bin/bash

echo "=== ArgoCD Apps - Go Implementation Demo ==="
echo ""

# Check if binary exists
if [ ! -f "bin/a9s" ]; then
    echo "Building application..."
    go build -o bin/a9s ./cmd/app
    if [ $? -ne 0 ]; then
        echo "Build failed!"
        exit 1
    fi
    echo "Build successful!"
    echo ""
fi

# Show usage information
echo "🚀 ArgoCD Apps TUI - Ready to run!"
echo ""
echo "📋 Usage Options:"
echo ""
echo "1. Demo Mode (shows UI with demo server - will show auth error):"
echo "   ./bin/a9s"
echo ""
echo "2. Real ArgoCD Server:"
echo "   export ARGOCD_SERVER=\"https://your-argocd-server.com\""
echo "   export ARGOCD_TOKEN=\"your-argocd-token\""
echo "   ./bin/a9s"
echo ""
echo "🎮 Keyboard Shortcuts:"
echo "   j/k or ↑/↓   - Navigate up/down"
echo "   space        - Select/deselect items"
echo "   enter        - Drill down to next view"
echo "   /            - Search mode"
echo "   :            - Command mode"
echo "   s            - Sync selected apps (in apps view)"
echo "   r            - Refresh data"
echo "   ?            - Show help"
echo "   esc          - Cancel/clear"
echo "   q or ctrl+c  - Quit"
echo ""
echo "📁 Logs are written to: logs/a9s.log"
echo ""

# Check for environment variables
if [ -n "$ARGOCD_SERVER" ] && [ -n "$ARGOCD_TOKEN" ]; then
    echo "✅ Found ArgoCD configuration:"
    echo "   Server: $ARGOCD_SERVER"
    echo "   Token: [configured]"
    echo ""
    echo "Starting with real ArgoCD data..."
else
    echo "ℹ️  No ArgoCD configuration found (ARGOCD_SERVER/ARGOCD_TOKEN)"
    echo "   Will run in demo mode to show UI"
    echo ""
    echo "Starting in demo mode..."
fi

echo ""
echo "Starting application..."
./bin/a9s