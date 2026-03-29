package main

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func strp(s string) *string {
	return &s
}

func TestRenderContextBlock_UsesTreeAppNamespaceProjectWhenScopeEmpty(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.state.Server = &model.Server{BaseURL: "https://argo.example.com"}
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeApp = &model.TreeAppInfo{
		Name:          "app-a",
		DestNamespace: strp("payments"),
		Project:       strp("billing"),
	}

	out := stripANSI(m.renderContextBlock(false))
	if !strings.Contains(out, "Namespace: payments") {
		t.Fatalf("expected Namespace from tree app info, got:\n%s", out)
	}
	if !strings.Contains(out, "Project: billing") {
		t.Fatalf("expected Project from tree app info, got:\n%s", out)
	}
}

func TestRenderContextBlock_ScopeOverridesTreeAppFallback(t *testing.T) {
	m := NewModel(nil)
	m.ready = true
	m.state.Server = &model.Server{BaseURL: "https://argo.example.com"}
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeApp = &model.TreeAppInfo{
		Name:          "app-a",
		DestNamespace: strp("payments"),
		Project:       strp("billing"),
	}
	m.state.Selections.ScopeNamespaces = model.StringSetFromSlice([]string{"scoped-ns"})
	m.state.Selections.ScopeProjects = model.StringSetFromSlice([]string{"scoped-proj"})

	out := stripANSI(m.renderContextBlock(false))
	if !strings.Contains(out, "Namespace: scoped-ns") {
		t.Fatalf("expected explicit namespace scope to win, got:\n%s", out)
	}
	if !strings.Contains(out, "Project: scoped-proj") {
		t.Fatalf("expected explicit project scope to win, got:\n%s", out)
	}
	if strings.Contains(out, "Namespace: payments") {
		t.Fatalf("did not expect app namespace fallback when explicit scope exists, got:\n%s", out)
	}
	if strings.Contains(out, "Project: billing") {
		t.Fatalf("did not expect app project fallback when explicit scope exists, got:\n%s", out)
	}
}
