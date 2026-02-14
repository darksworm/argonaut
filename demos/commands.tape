# Argonaut Command Palette Demo
# Shows: command palette navigation, scoping, search, and sort
#
# Usage: cd demos && make build && vhs commands.tape

Output demos/commands.gif

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

# Jump to apps view via command
Type ":"
Sleep 300ms
Type "apps"
Enter
Sleep 1s

# Search for an app
Type "/"
Sleep 300ms
Type "payment"
Sleep 800ms
Enter
Sleep 800ms

# Clear filter
Escape
Sleep 600ms

# Sort by health descending (degraded apps first)
Type ":"
Sleep 300ms
Type "sort health desc"
Enter
Sleep 1s

# Sort by name ascending
Type ":"
Sleep 300ms
Type "sort name asc"
Enter
Sleep 1s

# Navigate to namespaces view
Type ":"
Sleep 300ms
Type "ns"
Enter
Sleep 1s

# Navigate to projects view
Type ":"
Sleep 300ms
Type "proj"
Enter
Sleep 1s

# Navigate to appsets view
Type ":"
Sleep 300ms
Type "appsets"
Enter
Sleep 1s

# Back to clusters
Type ":"
Sleep 300ms
Type "clusters"
Enter
Sleep 1s

# Exit
Type "q"
Sleep 1s
