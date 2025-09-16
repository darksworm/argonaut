package main

import (
    "github.com/charmbracelet/bubbles/v2/table"
    "github.com/charmbracelet/lipgloss/v2"
)

func getTableStyle() table.Styles {
    s := table.DefaultStyles()
    s.Header = s.Header.
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("240")).
        BorderBottom(true).
        Bold(false)
    s.Selected = s.Selected.
        Foreground(lipgloss.Color("229")).
        Background(lipgloss.Color("57")).
        Bold(false)
    return s
}

func newAppsTable() table.Model {
    cols := []table.Column{{Title: "NAME", Width: 40}, {Title: "SYNC", Width: 12}, {Title: "HEALTH", Width: 15}}
    t := table.New(table.WithColumns(cols), table.WithFocused(true), table.WithHeight(10))
    t.SetStyles(getTableStyle())
    return t
}

func newSimpleTable() table.Model {
    cols := []table.Column{{Title: "NAME", Width: 60}}
    t := table.New(table.WithColumns(cols), table.WithFocused(true), table.WithHeight(10))
    t.SetStyles(getTableStyle())
    return t
}

