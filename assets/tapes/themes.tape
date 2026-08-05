# Records the :theme picker previewing themes over the apps view.
# Prep and post-processing: see assets/README.md
Output assets/argonaut_themes.gif

Set FontSize 20
Set Width 1440
Set Height 720
Set Padding 12
Set Framerate 10
Set TypingSpeed 50ms
Env ARGONAUT_CONFIG "assets/tapes/demo-config.toml"

Hide
Type "./argonaut-demo"
Enter
Wait+Screen@30s /Ready/
Sleep 500ms
Show

Sleep 1500ms
Type ":theme"
Sleep 400ms
Enter
Sleep 1500ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 1200ms
Escape
Sleep 2s
