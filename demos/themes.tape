# Argonaut Themes Demo
# Shows: theme selector with live preview cycling through themes
#
# Usage: cd demos && make build && vhs themes.tape

Output demos/themes.gif

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

# Jump to apps so themes look more interesting with data
Type ":"
Sleep 300ms
Type "apps"
Enter
Sleep 1s

# Open theme selector
Type ":"
Sleep 300ms
Type "theme"
Enter
Sleep 1s

# Cycle through themes with live preview
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 800ms
Down
Sleep 800ms

# Select this theme
Enter
Sleep 1s

# Exit
Type "q"
Sleep 1s
