# Argonaut Sync Demo
# Shows: navigate to apps → multi-select → sync with options → watch tree
#
# Usage: cd demos && make build && vhs sync.tape

Output demos/sync.gif

Set Shell "bash"
Set FontSize 22
Set Width 1920
Set Height 1080
Set Padding 10
Set TypingSpeed 50ms

# Launch directly into apps view via command
Type "argonaut"
Enter
Sleep 3s

# Use command palette to jump to apps view
Type ":"
Sleep 300ms
Type "apps"
Enter
Sleep 1s

# Select first app with Space
Sleep 300ms
Space
Sleep 400ms

# Move down and select second app
Down
Sleep 300ms
Space
Sleep 400ms

# Trigger sync
Type "s"
Sleep 1s

# Toggle prune option
Type "p"
Sleep 600ms

# Toggle watch tree option
Type "w"
Sleep 600ms

# Confirm sync
Enter
Sleep 3s

# We're now in tree view watching resources
# Navigate the tree
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms

Sleep 2s

# Exit
Escape
Sleep 500ms
Type "q"
Sleep 1s
