# Captures the resource actions modal on a Rollout (resource_actions_raw.png).
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
Sleep 1500ms
Type ":resources rollout-canary"
Enter
Wait+Screen@15s /EVENTS/
Sleep 1s
Type "j"
Sleep 300ms
Type "j"
Sleep 500ms
Type "a"
Sleep 3s
Show
Sleep 1s
Screenshot resource_actions_raw.png
Sleep 500ms
