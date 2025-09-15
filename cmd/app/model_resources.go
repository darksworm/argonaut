package main

import (
    "context"
    cblog "github.com/charmbracelet/log"
    "github.com/darksworm/argonaut/pkg/api"
    "github.com/darksworm/argonaut/pkg/model"
    tea "github.com/charmbracelet/bubbletea/v2"
)

// loadResourcesForApp creates a command to load resources for the given app
func (m Model) loadResourcesForApp(appName string) tea.Cmd {
    cblog.With("component", "tree").Info("Loading resources", "app", appName)
    return func() tea.Msg {
        if m.state.Server == nil {
            cblog.With("component", "tree").Error("Server not configured while loading resources", "app", appName)
            return ResourcesLoadedMsg{ AppName: appName, Error: "Server not configured" }
        }
        cblog.With("component", "tree").Debug("Creating ApplicationService", "server", m.state.Server.BaseURL)
        appService := api.NewApplicationService(m.state.Server)
        cblog.With("component", "tree").Debug("Calling GetResourceTree", "app", appName)
        tree, err := appService.GetResourceTree(context.Background(), appName, "")
        if err != nil {
            cblog.With("component", "tree").Error("Failed to load resources", "app", appName, "err", err)
            return ResourcesLoadedMsg{ AppName: appName, Error: err.Error() }
        }
        cblog.With("component", "tree").Info("Loaded resources", "count", len(tree.Nodes), "app", appName)
        modelResources := make([]model.ResourceNode, len(tree.Nodes))
        for i, node := range tree.Nodes { modelResources[i] = convertApiToModelResourceNode(node) }
        return ResourcesLoadedMsg{ AppName: appName, Resources: modelResources }
    }
}

// ResourcesLoadedMsg represents the result of loading resources
type ResourcesLoadedMsg struct {
    AppName   string
    Resources []model.ResourceNode
    Error     string
}

// convertApiToModelResourceNode converts api.ResourceNode to model.ResourceNode
func convertApiToModelResourceNode(apiNode api.ResourceNode) model.ResourceNode {
    var health *model.ResourceHealth
    if apiNode.Health != nil {
        health = &model.ResourceHealth{ Status: apiNode.Health.Status, Message: apiNode.Health.Message }
    }
    var networkingInfo *model.NetworkingInfo
    if apiNode.NetworkingInfo != nil {
        targetRefs := make([]model.ResourceRef, len(apiNode.NetworkingInfo.TargetRefs))
        for i, ref := range apiNode.NetworkingInfo.TargetRefs {
            targetRefs[i] = model.ResourceRef{ Group: ref.Group, Kind: ref.Kind, Name: ref.Name, Namespace: ref.Namespace }
        }
        networkingInfo = &model.NetworkingInfo{ TargetLabels: apiNode.NetworkingInfo.TargetLabels, TargetRefs: targetRefs }
    }
    return model.ResourceNode{ Group: apiNode.Group, Kind: apiNode.Kind, Name: apiNode.Name, Namespace: apiNode.Namespace, Version: apiNode.Version, Health: health, NetworkingInfo: networkingInfo }
}

