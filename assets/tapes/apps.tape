# Captures the live apps list (apps_raw.png).
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
Show
Sleep 1500ms
Screenshot apps_raw.png
Sleep 500ms
