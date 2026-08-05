# Captures the :help modal as a still frame (help_raw.png).
# Quantize it before committing: see assets/README.md
Output help_discard.gif

Set FontSize 20
Set Width 1440
Set Height 780
Set Padding 12
Set Framerate 5
Env ARGONAUT_CONFIG "assets/tapes/demo-config.toml"

Hide
Type "./argonaut-demo"
Enter
Wait+Screen@30s /Ready/
Sleep 500ms
Type "?"
Wait+Screen@10s /TREE VIEW/
Sleep 1s
Show
Sleep 1s
Screenshot help_raw.png
Sleep 500ms
