# Records the resource tree + events pane walking down to a pod.
# Prep and post-processing: see assets/README.md
Output assets/argonaut_events.gif

Set FontSize 20
Set Width 1440
Set Height 720
Set Padding 12
Set Framerate 10
Env ARGONAUT_CONFIG "assets/tapes/demo-config.toml"

Hide
Type "./argonaut-demo"
Enter
Wait+Screen@30s /Ready/
Sleep 500ms
Type ":resources rollout-bluegreen"
Enter
Wait+Screen@15s /EVENTS/
Sleep 1500ms
Show

Sleep 3s
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 2s
Sleep 3s
