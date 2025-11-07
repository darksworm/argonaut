# Contributing

When contributing to this repository, please first discuss the change you wish to make via issue, email, or any other method with the owners of this repository before making a change.

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
