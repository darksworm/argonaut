# Argonaut k9s Integration Demo
# Shows: apps view → sync with watch → open Deployment in k9s → scale replicas
#
# Requires a k3d cluster: make k9s-cluster-up
# Scenario set via ARGONAUT_DEMO_SCENARIO=k9s in Makefile
# Usage: cd demos && make record-k9s

Output demos/k9s.mp4

Set Shell "bash"
Set Theme "TokyoNight"
Set FontSize 29
Set Width 1920
Set Height 1080
Set Padding 0
Set WindowBar ""
Set BorderRadius 0
Set TypingSpeed 50ms

# Launch (default_view = "apps" so we land directly in apps view)
Type "argonaut"
Enter
Sleep 2s

# Navigate to config-server (OutOfSync) — 2nd in alphabetical list
Down
Sleep 200ms

# Sync config-server (watch is on by default)
Type "s"
Sleep 500ms

# Enable prune
Type "p"
Sleep 1s

# Confirm sync — wait for sync + tree to load
Enter
Sleep 2s

# Navigate to the Deployment (first child under the Application header)
Down
Sleep 500ms

# Open in k9s
Type "K"
Sleep 3s

# Scale deployment: 1 → 2 replicas
Type "s"
Sleep 300ms
Backspace
Type "2"
Sleep 300ms
Enter
Sleep 3s

# Exit k9s
Type ":q"
Enter
Sleep 2s

# Back in argonaut — exit
Escape
Sleep 1s
Type "q"
Sleep 1s
