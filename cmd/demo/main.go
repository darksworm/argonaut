// Package main provides a demo orchestrator for Argonaut.
// It starts a mock ArgoCD server with curated, realistic-looking data,
// creates temporary config files, and runs the argonaut binary.
// Designed for use with VHS (charmbracelet/vhs) tape recordings.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// appVersion is set at build time via -ldflags.
var appVersion string

// k9sTokyoNightSkin is the Tokyo Night skin for k9s.
// Source: https://github.com/axkirillov/k9s-tokyonight
const k9sTokyoNightSkin = `foreground: &foreground "#c0caf5"
background: &background "#24283b"
current_line: &current_line "#8c6c3e"
selection: &selection "#364a82"
comment: &comment "#565f89"
cyan: &cyan "#7dcfff"
green: &green "#9ece6a"
yellow: &yellow "#e0af68"
orange: &orange "#ff9e64"
magenta: &magenta "#bb9af7"
blue: &blue "#7aa2f7"
red: &red "#f7768e"
purple: &purple "#9d7cd8"
pink: &pink "#bb9af7"
white: &white "#a9b1d6"
black: &black "#1d202f"

k9s:
  body:
    fgColor: *foreground
    bgColor: default
    logoColor: *blue
  prompt:
    fgColor: *foreground
    bgColor: *background
    suggestColor: *orange
  info:
    fgColor: *magenta
    sectionColor: *foreground
  dialog:
    fgColor: *foreground
    bgColor: default
    buttonFgColor: *foreground
    buttonBgColor: *magenta
    buttonFocusFgColor: *background
    buttonFocusBgColor: *foreground
    labelFgColor: *comment
    fieldFgColor: *foreground
  frame:
    border:
      fgColor: *selection
      focusColor: *foreground
    menu:
      fgColor: *foreground
      keyColor: *magenta
      numKeyColor: *magenta
    crumbs:
      fgColor: *white
      bgColor: *cyan
      activeColor: *yellow
    status:
      newColor: *magenta
      modifyColor: *blue
      addColor: *green
      errorColor: *red
      highlightcolor: *orange
      killColor: *comment
      completedColor: *comment
    title:
      fgColor: *foreground
      bgColor: default
      highlightColor: *blue
      counterColor: *magenta
      filterColor: *magenta
  views:
    charts:
      bgColor: default
      defaultDialColors:
        - *blue
        - *red
      defaultChartColors:
        - *blue
        - *red
    table:
      fgColor: *foreground
      bgColor: default
      cursorFgColor: *white
      cursorBgColor: *background
      markColor: darkgoldenrod
      header:
        fgColor: *foreground
        bgColor: default
        sorterColor: *cyan
    xray:
      fgColor: *foreground
      bgColor: default
      cursorColor: *current_line
      graphicColor: *blue
      showIcons: false
    yaml:
      keyColor: *magenta
      colonColor: *blue
      valueColor: *foreground
    logs:
      fgColor: *foreground
      bgColor: default
      indicator:
        fgColor: *foreground
        bgColor: *selection
    help:
      fgColor: *foreground
      bgColor: default
      indicator:
        fgColor: *red
        bgColor: *selection
`

// argoApp is a minimal structure matching ArgoCD's Application JSON.
type argoApp struct {
	Metadata struct {
		Name            string           `json:"name"`
		Namespace       string           `json:"namespace"`
		OwnerReferences []ownerReference `json:"ownerReferences,omitempty"`
	} `json:"metadata"`
	Spec struct {
		Project string `json:"project"`
		Source  *struct {
			RepoURL        string `json:"repoURL"`
			Path           string `json:"path"`
			TargetRevision string `json:"targetRevision"`
		} `json:"source,omitempty"`
		Destination struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"destination"`
	} `json:"spec"`
	Status struct {
		Sync struct {
			Status   string `json:"status"`
			Revision string `json:"revision,omitempty"`
		} `json:"sync"`
		Health struct {
			Status  string `json:"status"`
			Message string `json:"message,omitempty"`
		} `json:"health"`
		OperationState struct {
			Phase      string `json:"phase,omitempty"`
			StartedAt  string `json:"startedAt,omitempty"`
			FinishedAt string `json:"finishedAt,omitempty"`
		} `json:"operationState,omitempty"`
		History   []deploymentHistory `json:"history,omitempty"`
		Resources []interface{}       `json:"resources,omitempty"`
	} `json:"status"`
}

type deploymentHistory struct {
	ID         int    `json:"id"`
	Revision   string `json:"revision"`
	DeployedAt string `json:"deployedAt"`
	Source     *struct {
		RepoURL        string `json:"repoURL"`
		Path           string `json:"path"`
		TargetRevision string `json:"targetRevision"`
	} `json:"source,omitempty"`
}

type ownerReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	UID        string `json:"uid"`
}

type resourceNode struct {
	Kind       string        `json:"kind"`
	Name       string        `json:"name"`
	Namespace  string        `json:"namespace"`
	Version    string        `json:"version"`
	Group      string        `json:"group"`
	UID        string        `json:"uid"`
	Health     *healthStatus `json:"health,omitempty"`
	Status     string        `json:"status"`
	ParentRefs []parentRef   `json:"parentRefs,omitempty"`
	Info       []infoItem    `json:"info,omitempty"`
	CreatedAt  string        `json:"createdAt,omitempty"`
}

type healthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type parentRef struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Group     string `json:"group"`
	Version   string `json:"version"`
	UID       string `json:"uid"`
}

type infoItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// buildDemoApps creates the curated set of realistic applications.
func buildDemoApps() []argoApp {
	now := time.Now().UTC()
	finished := now.Add(-23 * time.Minute).Format(time.RFC3339)
	started := now.Add(-24 * time.Minute).Format(time.RFC3339)
	olderFinished := now.Add(-2 * time.Hour).Format(time.RFC3339)
	olderStarted := now.Add(-2*time.Hour - 3*time.Minute).Format(time.RFC3339)

	apps := []argoApp{
		makeApp("payment-api", "ecommerce", "prod-us-east-1", "payments", "Synced", "Healthy", "", "abc1234", finished, started),
		makeApp("user-service", "ecommerce", "prod-us-east-1", "users", "Synced", "Healthy", "", "def5678", finished, started),
		makeApp("frontend-web", "ecommerce", "prod-us-east-1", "frontend", "OutOfSync", "Healthy", "", "aaa1111", olderFinished, olderStarted),
		makeApp("cart-service", "ecommerce", "staging-eu-west-1", "cart", "Synced", "Healthy", "", "bbb2222", finished, started),
		makeApp("notification-worker", "platform", "staging-eu-west-1", "notifications", "Synced", "Degraded", "CrashLoopBackOff: container restarting", "ccc3333", olderFinished, olderStarted),
		makeApp("config-server", "platform", "prod-us-east-1", "platform", "Synced", "Healthy", "", "ddd4444", finished, started),
		makeApp("redis-cache", "platform", "prod-us-east-1", "cache", "Synced", "Healthy", "", "eee5555", finished, started),
		makeApp("ingress-controller", "platform", "staging-eu-west-1", "ingress", "Synced", "Healthy", "", "fff6666", olderFinished, olderStarted),
	}

	// Mark ecommerce apps as belonging to an ApplicationSet
	appsetName := "ecommerce-apps"
	for i := range apps {
		if apps[i].Spec.Project == "ecommerce" {
			apps[i].Metadata.OwnerReferences = []ownerReference{{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "ApplicationSet",
				Name:       appsetName,
				UID:        "appset-ecommerce-uid",
			}}
		}
	}

	return apps
}

func makeApp(name, project, cluster, namespace, syncStatus, healthStat, healthMsg, revision, finishedAt, startedAt string) argoApp {
	var a argoApp
	a.Metadata.Name = name
	a.Metadata.Namespace = "argocd"
	a.Spec.Project = project
	a.Spec.Source = &struct {
		RepoURL        string `json:"repoURL"`
		Path           string `json:"path"`
		TargetRevision string `json:"targetRevision"`
	}{
		RepoURL:        fmt.Sprintf("https://github.com/acme-corp/%s.git", name),
		Path:           fmt.Sprintf("deploy/%s", name),
		TargetRevision: "HEAD",
	}
	a.Spec.Destination.Name = cluster
	a.Spec.Destination.Namespace = namespace
	a.Status.Sync.Status = syncStatus
	a.Status.Sync.Revision = revision
	a.Status.Health.Status = healthStat
	a.Status.Health.Message = healthMsg
	a.Status.OperationState.Phase = "Succeeded"
	a.Status.OperationState.FinishedAt = finishedAt
	a.Status.OperationState.StartedAt = startedAt
	return a
}

// addHistory adds deployment history entries to an app (for rollback demos).
func addHistory(app *argoApp) {
	now := time.Now().UTC()
	app.Status.History = []deploymentHistory{
		{ID: 5, Revision: app.Status.Sync.Revision, DeployedAt: now.Add(-23 * time.Minute).Format(time.RFC3339)},
		{ID: 4, Revision: "9f8e7d6", DeployedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)},
		{ID: 3, Revision: "5a4b3c2", DeployedAt: now.Add(-26 * time.Hour).Format(time.RFC3339)},
		{ID: 2, Revision: "1d2e3f4", DeployedAt: now.Add(-72 * time.Hour).Format(time.RFC3339)},
		{ID: 1, Revision: "a0b1c2d", DeployedAt: now.Add(-168 * time.Hour).Format(time.RFC3339)},
	}
	for i := range app.Status.History {
		app.Status.History[i].Source = app.Spec.Source
	}
}

// buildResourceTree creates a realistic resource tree for payment-api.
func buildResourceTree() map[string]interface{} {
	return map[string]interface{}{
		"nodes": []resourceNode{
			{
				Kind: "Service", Name: "payment-api", Namespace: "payments",
				Version: "v1", Group: "", UID: "svc-1",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				Info:   []infoItem{{Name: "Type", Value: "ClusterIP"}},
			},
			{
				Kind: "Deployment", Name: "payment-api", Namespace: "payments",
				Version: "v1", Group: "apps", UID: "dep-1",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
			},
			{
				Kind: "ReplicaSet", Name: "payment-api-7d4f8b6c95", Namespace: "payments",
				Version: "v1", Group: "apps", UID: "rs-1",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "Deployment", Name: "payment-api", Namespace: "payments",
					Group: "apps", Version: "v1", UID: "dep-1"}},
			},
			{
				Kind: "Pod", Name: "payment-api-7d4f8b6c95-x2k9m", Namespace: "payments",
				Version: "v1", Group: "", UID: "pod-1",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "ReplicaSet", Name: "payment-api-7d4f8b6c95", Namespace: "payments",
					Group: "apps", Version: "v1", UID: "rs-1"}},
				Info: []infoItem{{Name: "Status Reason", Value: "Running"}, {Name: "Containers", Value: "1/1"}},
			},
			{
				Kind: "Pod", Name: "payment-api-7d4f8b6c95-a8n3j", Namespace: "payments",
				Version: "v1", Group: "", UID: "pod-2",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "ReplicaSet", Name: "payment-api-7d4f8b6c95", Namespace: "payments",
					Group: "apps", Version: "v1", UID: "rs-1"}},
				Info: []infoItem{{Name: "Status Reason", Value: "Running"}, {Name: "Containers", Value: "1/1"}},
			},
			{
				Kind: "ConfigMap", Name: "payment-api-config", Namespace: "payments",
				Version: "v1", Group: "", UID: "cm-1",
				Status: "Synced",
			},
			{
				Kind: "HorizontalPodAutoscaler", Name: "payment-api", Namespace: "payments",
				Version: "v2", Group: "autoscaling", UID: "hpa-1",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
			},
			{
				Kind: "ServiceAccount", Name: "payment-api", Namespace: "payments",
				Version: "v1", Group: "", UID: "sa-1",
				Status: "Synced",
			},
		},
	}
}

// buildNotificationTree creates a resource tree with degraded resources.
func buildNotificationTree() map[string]interface{} {
	return map[string]interface{}{
		"nodes": []resourceNode{
			{
				Kind: "Deployment", Name: "notification-worker", Namespace: "notifications",
				Version: "v1", Group: "apps", UID: "nw-dep-1",
				Health: &healthStatus{Status: "Degraded", Message: "Deployment has minimum availability"},
				Status: "Synced",
			},
			{
				Kind: "ReplicaSet", Name: "notification-worker-5f6d7e8a9b", Namespace: "notifications",
				Version: "v1", Group: "apps", UID: "nw-rs-1",
				Health: &healthStatus{Status: "Degraded"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "Deployment", Name: "notification-worker", Namespace: "notifications",
					Group: "apps", Version: "v1", UID: "nw-dep-1"}},
			},
			{
				Kind: "Pod", Name: "notification-worker-5f6d7e8a9b-cr4sh", Namespace: "notifications",
				Version: "v1", Group: "", UID: "nw-pod-1",
				Health: &healthStatus{Status: "Degraded", Message: "CrashLoopBackOff"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "ReplicaSet", Name: "notification-worker-5f6d7e8a9b", Namespace: "notifications",
					Group: "apps", Version: "v1", UID: "nw-rs-1"}},
				Info: []infoItem{{Name: "Status Reason", Value: "CrashLoopBackOff"}, {Name: "Containers", Value: "0/1"}},
			},
			{
				Kind: "Service", Name: "notification-worker", Namespace: "notifications",
				Version: "v1", Group: "", UID: "nw-svc-1",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				Info:   []infoItem{{Name: "Type", Value: "ClusterIP"}},
			},
			{
				Kind: "ConfigMap", Name: "notification-worker-config", Namespace: "notifications",
				Version: "v1", Group: "", UID: "nw-cm-1",
				Status: "Synced",
			},
		},
	}
}

// buildSimpleResourceTree creates a minimal resource tree for apps without a custom one.
func buildSimpleResourceTree(appName, namespace string) map[string]interface{} {
	return map[string]interface{}{
		"nodes": []resourceNode{
			{
				Kind: "Deployment", Name: appName, Namespace: namespace,
				Version: "v1", Group: "apps", UID: appName + "-dep",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
			},
			{
				Kind: "ReplicaSet", Name: appName + "-6b8f9d4c77", Namespace: namespace,
				Version: "v1", Group: "apps", UID: appName + "-rs",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "Deployment", Name: appName, Namespace: namespace,
					Group: "apps", Version: "v1", UID: appName + "-dep"}},
			},
			{
				Kind: "Pod", Name: appName + "-6b8f9d4c77-zk4m2", Namespace: namespace,
				Version: "v1", Group: "", UID: appName + "-pod",
				Health: &healthStatus{Status: "Healthy"},
				Status: "Synced",
				ParentRefs: []parentRef{{Kind: "ReplicaSet", Name: appName + "-6b8f9d4c77", Namespace: namespace,
					Group: "apps", Version: "v1", UID: appName + "-rs"}},
				Info: []infoItem{{Name: "Status Reason", Value: "Running"}, {Name: "Containers", Value: "1/1"}},
			},
		},
	}
}

func startMockServer(apps []argoApp, done <-chan struct{}) *httptest.Server {
	appsJSON, _ := json.Marshal(apps)
	listResp := fmt.Sprintf(`{"metadata":{"resourceVersion":"5000"},"items":%s}`, string(appsJSON))

	// Build maps for per-app lookups
	appNS := make(map[string]string)
	appByName := make(map[string]*argoApp)
	for i, a := range apps {
		appNS[a.Metadata.Name] = a.Spec.Destination.Namespace
		appByName[a.Metadata.Name] = &apps[i]
	}

	// Add history to all apps for rollback support
	for i := range apps {
		addHistory(&apps[i])
	}

	// Mutex protects apps slice — sync handler updates status in-place,
	// SSE handler reads current state under lock.
	var mu sync.Mutex

	// Channel for sync notifications: sync handler sends app name,
	// SSE stream handler sends an updated event with Synced status.
	syncNotify := make(chan string, 8)

	// Pre-build resource tree JSON
	paymentTree, _ := json.Marshal(buildResourceTree())
	notificationTree, _ := json.Marshal(buildNotificationTree())

	mux := http.NewServeMux()

	// Auth check
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	// Version
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"2.14.0"}`))
	})

	// App list
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listResp))
		}
	})

	// SSE watch stream — builds payload dynamically from current app state
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, _ *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		// Send current state of all apps (reflects any in-place sync updates)
		mu.Lock()
		for _, a := range apps {
			evt := map[string]interface{}{
				"result": map[string]interface{}{
					"type":        "MODIFIED",
					"application": a,
				},
			}
			evtJSON, _ := json.Marshal(evt)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", evtJSON)
		}
		mu.Unlock()
		if fl != nil {
			fl.Flush()
		}
		// Listen for sync completions and send updated SSE events
		for {
			select {
			case name := <-syncNotify:
				mu.Lock()
				app, ok := appByName[name]
				if ok {
					// App was already updated in-place by sync handler;
					// send the current (Synced) state as an SSE event.
					evt := map[string]interface{}{
						"result": map[string]interface{}{
							"type":        "MODIFIED",
							"application": *app,
						},
					}
					evtJSON, _ := json.Marshal(evt)
					mu.Unlock()
					time.Sleep(150 * time.Millisecond)
					_, _ = fmt.Fprintf(w, "data: %s\n\n", evtJSON)
					if fl != nil {
						fl.Flush()
					}
				} else {
					mu.Unlock()
				}
			case <-done:
				return
			}
		}
	})

	// Known resource trees
	mux.HandleFunc("/api/v1/applications/payment-api/resource-tree", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(paymentTree)
	})
	mux.HandleFunc("/api/v1/applications/notification-worker/resource-tree", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(notificationTree)
	})

	// Catch-all for dynamic routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		const appsPrefix = "/api/v1/applications/"

		// Only handle paths under /api/v1/applications/
		if !strings.HasPrefix(path, appsPrefix) {
			http.NotFound(w, r)
			return
		}

		rest := path[len(appsPrefix):]

		// POST /api/v1/applications/{name}/sync
		if r.Method == http.MethodPost && strings.HasSuffix(rest, "/sync") {
			name := strings.TrimSuffix(rest, "/sync")
			time.Sleep(200 * time.Millisecond)
			// Update app status in-place so any new SSE connections see Synced
			mu.Lock()
			if app, ok := appByName[name]; ok {
				app.Status.Sync.Status = "Synced"
			}
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			// Notify active SSE streams to push the update
			select {
			case syncNotify <- name:
			default:
			}
			return
		}

		// POST /api/v1/applications/{name}/rollback
		if r.Method == http.MethodPost && strings.HasSuffix(rest, "/rollback") {
			time.Sleep(300 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}

		// DELETE /api/v1/applications/{name}
		if r.Method == http.MethodDelete && !strings.Contains(rest, "/") {
			time.Sleep(200 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Success": true}`))
			return
		}

		// GET /api/v1/applications/{name}/managed-resources (for diff)
		if strings.HasSuffix(rest, "/managed-resources") {
			name := strings.TrimSuffix(rest, "/managed-resources")
			w.Header().Set("Content-Type", "application/json")
			live := fmt.Sprintf(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"%s","namespace":"default"},"spec":{"replicas":2,"template":{"spec":{"containers":[{"name":"%s","image":"acme-corp/%s:v1.2.3"}]}}}}`, name, name, name)
			desired := fmt.Sprintf(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"%s","namespace":"default"},"spec":{"replicas":3,"template":{"spec":{"containers":[{"name":"%s","image":"acme-corp/%s:v1.3.0"}]}}}}`, name, name, name)
			liveEsc, _ := json.Marshal(live)
			desiredEsc, _ := json.Marshal(desired)
			_, _ = fmt.Fprintf(w, `{"items":[{"kind":"Deployment","namespace":"default","name":"%s","liveState":%s,"targetState":%s}]}`, name, liveEsc, desiredEsc)
			return
		}

		// GET /api/v1/applications/{name}/revisions/{rev}/metadata
		if strings.Contains(rest, "/revisions/") && strings.HasSuffix(rest, "/metadata") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"author":"Jane Smith","date":"` + time.Now().Add(-1*time.Hour).Format(time.RFC3339) + `","message":"chore: update deployment config","tags":[]}`))
			return
		}

		// GET /api/v1/applications/{name}/resource-tree
		if strings.HasSuffix(rest, "/resource-tree") {
			name := strings.TrimSuffix(rest, "/resource-tree")
			ns := appNS[name]
			if ns == "" {
				ns = "default"
			}
			tree, _ := json.Marshal(buildSimpleResourceTree(name, ns))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(tree)
			return
		}

		// GET /api/v1/applications/{name} (full app detail with history)
		if r.Method == http.MethodGet && !strings.Contains(rest, "/") {
			name := rest
			if app, ok := appByName[name]; ok {
				w.Header().Set("Content-Type", "application/json")
				data, _ := json.Marshal(app)
				_, _ = w.Write(data)
				return
			}
		}

		// SSE resource tree stream
		const streamPrefix = "/api/v1/stream/applications/"
		if strings.HasPrefix(path, streamPrefix) && strings.HasSuffix(path, "/resource-tree") {
			name := strings.TrimPrefix(path, streamPrefix)
			name = strings.TrimSuffix(name, "/resource-tree")
			ns := appNS[name]
			if ns == "" {
				ns = "default"
			}
			var tree interface{}
			switch name {
			case "payment-api":
				tree = buildResourceTree()
			case "notification-worker":
				tree = buildNotificationTree()
			default:
				tree = buildSimpleResourceTree(name, ns)
			}
			result := map[string]interface{}{"result": tree}
			data, _ := json.Marshal(result)

			fl, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			if fl != nil {
				fl.Flush()
			}
			<-done
			return
		}

		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}

func writeArgoConfig(path, serverURL string) error {
	var buf bytes.Buffer
	buf.WriteString("contexts:\n")
	buf.WriteString("  - name: default\n    server: " + serverURL + "\n    user: default-user\n")
	buf.WriteString("servers:\n")
	buf.WriteString("  - server: " + serverURL + "\n    insecure: true\n")
	buf.WriteString("users:\n")
	buf.WriteString("  - name: default-user\n    auth-token: demo-token\n")
	buf.WriteString("current-context: default\n")
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func writeArgonautConfig(path, theme, defaultView, version string) error {
	// Top-level keys must come before any [section] header in TOML
	var config string
	if defaultView != "" {
		config += fmt.Sprintf("default_view = %q\n", defaultView)
	}
	if version != "" {
		config += fmt.Sprintf("last_seen_version = %q\n", version)
	}
	config += fmt.Sprintf("\n[appearance]\ntheme = %q\n", theme)
	return os.WriteFile(path, []byte(config), 0o600)
}

func main() {
	scenario := flag.String("scenario", "", "Demo scenario (overview, sync, resources, commands, themes, k9s)")
	argonautBin := flag.String("bin", "", "Path to argonaut binary (auto-detected if empty)")
	theme := flag.String("theme", "tokyo-night", "Theme to use")
	flag.Parse()

	// Allow env var override so VHS tapes can just type "argonaut"
	if *scenario == "" {
		if env := os.Getenv("ARGONAUT_DEMO_SCENARIO"); env != "" {
			*scenario = env
		} else {
			*scenario = "overview"
		}
	}

	// Find argonaut binary
	bin := *argonautBin
	if bin == "" {
		self, _ := os.Executable()
		dir := filepath.Dir(self)
		candidates := []string{
			filepath.Join(dir, "argonaut-app"),
			filepath.Join(dir, "..", "..", "bin", "argonaut-app"),
			"argonaut-app",
		}
		for _, c := range candidates {
			if _, err := exec.LookPath(c); err == nil {
				bin = c
				break
			}
			if _, err := os.Stat(c); err == nil {
				bin = c
				break
			}
		}
		if bin == "" {
			fmt.Fprintln(os.Stderr, "Error: could not find argonaut-app binary. Use --bin to specify its path.")
			os.Exit(1)
		}
	}

	// Validate scenario
	validScenarios := map[string]bool{
		"overview": true, "sync": true, "resources": true,
		"commands": true, "themes": true, "k9s": true,
	}
	if !validScenarios[*scenario] {
		fmt.Fprintf(os.Stderr, "Unknown scenario: %s\nAvailable: overview, sync, resources, commands, themes, k9s\n", *scenario)
		os.Exit(1)
	}

	// Build demo data
	apps := buildDemoApps()

	// k9s scenario: make config-server OutOfSync so we have two adjacent
	// OutOfSync apps (config-server + frontend-web) for multi-select sync demo
	if *scenario == "k9s" {
		for i := range apps {
			if apps[i].Metadata.Name == "config-server" {
				apps[i].Status.Sync.Status = "OutOfSync"
			}
		}
	}

	// done is closed when the child process exits, unblocking SSE handlers.
	done := make(chan struct{})

	srv := startMockServer(apps, done)
	defer srv.Close()

	// Create isolated workspace
	workspace, err := os.MkdirTemp("", "argonaut-demo-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create workspace: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workspace)

	// Write ArgoCD CLI config
	argoDir := filepath.Join(workspace, ".config", "argocd")
	_ = os.MkdirAll(argoDir, 0o755)
	argoCfgPath := filepath.Join(argoDir, "config")
	if err := writeArgoConfig(argoCfgPath, srv.URL); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write argocd config: %v\n", err)
		os.Exit(1)
	}

	// Write Argonaut config
	argonautDir := filepath.Join(workspace, ".config", "argonaut")
	_ = os.MkdirAll(argonautDir, 0o755)
	argonautCfgPath := filepath.Join(argonautDir, "config.toml")
	defaultView := ""
	if *scenario == "k9s" {
		defaultView = "apps"
	}
	if err := writeArgonautConfig(argonautCfgPath, *theme, defaultView, appVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write argonaut config: %v\n", err)
		os.Exit(1)
	}

	// Suppress any "what's new" or config-existed logic by providing a dummy
	_ = io.Discard

	// If KUBECONFIG is set (e.g. k9s demo), copy it into the workspace so that
	// both argonaut and k9s find it at the default ~/.kube/config path.
	// This is needed because HOME is overridden to the workspace.
	if kubecfg := os.Getenv("KUBECONFIG"); kubecfg != "" {
		kubeDir := filepath.Join(workspace, ".kube")
		_ = os.MkdirAll(kubeDir, 0o755)
		if data, err := os.ReadFile(kubecfg); err == nil {
			_ = os.WriteFile(filepath.Join(kubeDir, "config"), data, 0o600)
		}
	}

	// Write k9s Tokyo Night skin so k9s matches the argonaut/VHS theme.
	if *scenario == "k9s" {
		k9sSkinDir := filepath.Join(workspace, ".config", "k9s", "skins")
		_ = os.MkdirAll(k9sSkinDir, 0o755)
		_ = os.WriteFile(filepath.Join(k9sSkinDir, "tokyo-night.yaml"), []byte(k9sTokyoNightSkin), 0o644)
	}

	// Run argonaut as a child process
	cmd := exec.Command(bin, "-argocd-config="+argoCfgPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	xdgConfig := filepath.Join(workspace, ".config")
	env := append(os.Environ(),
		"HOME="+workspace,
		"XDG_CONFIG_HOME="+xdgConfig,
		"ARGONAUT_CONFIG="+argonautCfgPath,
	)
	if *scenario == "k9s" {
		env = append(env, "K9S_SKIN=tokyo-night")
	}
	cmd.Env = env

	// Forward signals to child
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err = cmd.Run()
	close(done)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Failed to run argonaut: %v\n", err)
		os.Exit(1)
	}
}
