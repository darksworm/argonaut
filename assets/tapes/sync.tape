# Captures the sync confirmation modal (sync_raw.png).
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
Type "s"
Wait+Screen@10s /Yes/
Show
Sleep 1s
Screenshot sync_raw.png
Sleep 500ms
