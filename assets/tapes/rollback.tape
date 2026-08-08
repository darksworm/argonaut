# Captures the two-pane deployment history browser (rollback_raw.png).
# Needs the seeded history app: `make argocd-history`.
# Prep and post-processing: see assets/README.md
Output screenshot_discard.gif

Set FontSize 20
Set Width 1440
# taller than the other tapes: the capped 10-row list leaves the extra
# height to the detail pane, which needs it for the COMMIT/SOURCE sections
Set Height 960
Set Padding 12
Set Framerate 5
Env ARGONAUT_CONFIG "assets/tapes/demo-config.toml"

Hide
Type "./argonaut-demo"
Enter
Wait+Screen@30s /Ready/
Sleep 500ms
Type "/history-demo"
Enter
Sleep 300ms
Type "R"
Wait+Screen@15s /Deployment history/
# commit subjects load per revision after the list appears
Wait+Screen@15s /feat:/
Sleep 1s
Show
Sleep 1s
Screenshot rollback_raw.png
Sleep 500ms
