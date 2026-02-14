# Argonaut Overview Demo
# Shows: launch → cluster navigation → drill into apps → resource tree → exit
#
# Usage: cd demos && make build && vhs overview.tape

Output demos/overview.gif

Set Shell "bash"
Set FontSize 22
Set Width 1920
Set Height 1080
Set Padding 10
Set TypingSpeed 50ms

# Launch
Type "argonaut"
Enter
Sleep 3s

# We start in clusters view - browse clusters
Down
Sleep 800ms

# Drill into a cluster
Enter
Sleep 1s

# Now in namespaces view - browse
Down
Sleep 600ms
Down
Sleep 600ms

# Drill into a namespace → projects
Enter
Sleep 1s

# Drill into project → apps
Enter
Sleep 1s

# Browse apps
Down
Sleep 500ms
Down
Sleep 500ms

# Open resource tree for selected app
Enter
Sleep 2s

# Navigate the resource tree
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms

# Go back to apps
Escape
Sleep 1s

# Exit
Type "q"
Sleep 1s
