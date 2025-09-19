package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// watchTreeDeliver is used by the watcher goroutine to send messages into Bubble Tea
func (m *Model) watchTreeDeliver(msg model.ResourceTreeStreamMsg) {
	select {
	case m.treeStream <- msg:
		cblog.With("component", "ui").Debug("Tree event delivered to channel", "app", msg.AppName)
	default:
		cblog.With("component", "ui").Warn("Tree channel full, dropping event", "app", msg.AppName)
	}
}

// consumeTreeEvent reads a single tree stream event and returns it as a tea message
func (m *Model) consumeTreeEvent() tea.Cmd {
	return func() tea.Msg {
		if m.treeStream == nil {
			return nil
		}
		ev, ok := <-m.treeStream
		if !ok {
			cblog.With("component", "ui").Debug("Tree stream channel closed")
			return nil
		}
		cblog.With("component", "ui").Debug("Consumed tree event", "app", ev.AppName)
		return ev
	}
}
