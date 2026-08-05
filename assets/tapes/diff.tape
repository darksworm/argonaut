# Captures the diff pager on an OutOfSync app (diff_raw.png).
# Prep and post-processing: see assets/README.md
Output screenshot_discard.gif

Set FontSize 20
Set Width 1440
Set Height 720
Set Padding 12
Set Framerate 5
Env ARGONAUT_CONFIG "assets/tapes/demo-config.toml"

Hide
Type "./argonaut-demo"
Enter
Wait+Screen@30s /Ready/
Sleep 500ms
Type "j"
Sleep 300ms
Type "d"
Sleep 6s
Show
Sleep 1s
Screenshot diff_raw.png
Sleep 500ms
