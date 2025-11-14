# Contributing

When contributing to this repository, please first discuss the change you wish to make via issue, email, or any other method with the owners of this repository before making a change.

If you haven't let us know what/how/why you're working on, your contribution might get rejected, but please don't get discouraged - just get a 👍 from one of the maintainers in the issues section and you're ready to rock and roll.

If you'd like to contribute, but don't know where to start - check the issues labeled "good first issue".

🤖 AI aided contributions are welcome. Just make sure to add proper test coverage!

## How to Contribute a New Theme

To add a new theme to the project:

1. **Add your theme to the presets map** in `pkg/theme/presets.go`:

```go
"your-theme-name": {
    Accent: lipgloss.Color("#your-accent-color"),
    Warning: lipgloss.Color("#your-warning-color"),
    Dim: lipgloss.Color("#your-dim-color"),
    Success: lipgloss.Color("#your-success-color"),
    Danger: lipgloss.Color("#your-danger-color"),
    Progress: lipgloss.Color("#your-progress-color"),
    Unknown: lipgloss.Color("#your-unknown-color"),
    Info: lipgloss.Color("#your-info-color"),
    Text: lipgloss.Color("#your-text-color"),
    Gray: lipgloss.Color("#your-gray-color"),
    SelectedBG: lipgloss.Color("#your-selected-bg"),
    CursorSelectedBG: lipgloss.Color("#your-cursor-selected-bg"),
    CursorBG: lipgloss.Color("#your-cursor-bg"),
    Border: lipgloss.Color("#your-border-color"),
    MutedBG: lipgloss.Color("#your-muted-bg"),
    ShadeBG: lipgloss.Color("#your-shade-bg"),
    DarkBG: lipgloss.Color("#your-dark-bg"),
},
```

2. **Test your theme**:
```bash
go run cmd/app/*.go --theme your-theme-name
```

3. **Submit a pull request** with your theme addition.
