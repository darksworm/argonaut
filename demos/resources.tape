# Argonaut Resources Demo
# Shows: open resource tree → expand/collapse → search/filter → exit
#
# Usage: cd demos && make build && vhs resources.tape

Output demos/resources.gif

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

# Jump to apps view
Type ":"
Sleep 300ms
Type "apps"
Enter
Sleep 1s

# Open resources for first app (payment-api has the richest tree)
Type "r"
Sleep 2s

# Navigate down through the resource tree
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms

# Go back to top
Type "g"
Sleep 600ms

# Search for "pod" in the tree
Type "/"
Sleep 300ms
Type "pod"
Sleep 800ms

# Jump to next match
Enter
Sleep 500ms
Type "n"
Sleep 500ms
Type "n"
Sleep 500ms

# Clear search
Escape
Sleep 800ms

# Go back to apps
Escape
Sleep 1s

# Exit
Type "q"
Sleep 1s
