package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// watchTreeDeliver is used by the watcher goroutine to send messages into Bubble Tea
func (m Model) watchTreeDeliver(msg model.ResourceTreeStreamMsg) {
	select {
	case m.treeStream <- msg:
	default:
	}
}

// consumeTreeEvent reads a single tree stream event and returns it as a tea message
func (m Model) consumeTreeEvent() tea.Cmd {
	return func() tea.Msg {
		if m.treeStream == nil {
			return nil
		}
		ev, ok := <-m.treeStream
		if !ok {
			return nil
		}
		return ev
	}
}
