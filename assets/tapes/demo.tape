# Full feature tour: apps list → search → diff → sync → resource tree +
# events → run a Rollouts action → roll back the history app → themes → help.
# Prep and post-processing: see assets/README.md
Output assets/argonaut_demo.gif

Set FontSize 20
# wider than the stills so the two-pane views (tree + events, history +
# detail) get room; tall enough for the help modal, which clips below ~780
Set Width 1920
Set Height 900
Set Padding 12
Set Framerate 10
Set TypingSpeed 60ms
Env ARGONAUT_CONFIG "assets/tapes/demo-config.toml"

Hide
Type "./argonaut-demo"
Enter
Wait+Screen@30s /Ready/
Sleep 500ms
Show

# apps list
Sleep 2500ms

# search / filter
Type "/canary"
Sleep 400ms
Enter
Sleep 1500ms

# diff (delta pager on the OutOfSync rollout)
Type "d"
Wait+Screen@15s /Desired/
Sleep 2500ms
Type "q"
Sleep 1s

# sync confirmation
Type "s"
Wait+Screen@10s /Yes/
Sleep 2s
Escape
Sleep 800ms

# resource tree + live events pane
Type ":resources rollout-canary"
Sleep 400ms
Enter
Wait+Screen@15s /EVENTS/
Sleep 2s
Type "j"
Sleep 600ms
Type "j"
Sleep 1200ms

# run a Rollouts action: resume the paused canary
Type "a"
Wait+Screen@10s /promote-full/
Sleep 1500ms
Type "resume"
Sleep 1200ms
Enter
Sleep 3500ms
Escape
Sleep 1s

# deployment history: roll back to the previous revision and watch it
Type "/history-demo"
Sleep 400ms
Enter
Sleep 800ms
Type "R"
Wait+Screen@15s /Deployment history/
Wait+Screen@15s /feat:/
Sleep 2s
Type "j"
Sleep 1200ms
Enter
Sleep 1500ms
Type "y"
Wait+Screen@30s /EVENTS/
Sleep 4s
Escape
Sleep 1500ms
Escape
Sleep 600ms
Escape
Sleep 1s

# theme picker
Type ":theme"
Sleep 400ms
Enter
Sleep 1200ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 900ms
Type "k"
Sleep 1200ms
Escape
Sleep 1s

# help
Type "?"
Wait+Screen@10s /TREE VIEW/
Sleep 2500ms
Escape
Sleep 2s
