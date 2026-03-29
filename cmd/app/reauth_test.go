package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/auth"
	"github.com/darksworm/argonaut/pkg/model"
)

// fakeAuthProvider is a test double for auth.JWTAuthProvider.
type fakeAuthProvider struct {
	cmd *exec.Cmd
}

func (f *fakeAuthProvider) LoginCmd(_ auth.LoginParams) *exec.Cmd {
	return f.cmd
}

// writeTestArgoConfig writes a minimal ArgoCD CLI config with the given server URL and token.
// Returns the path of the written file.
func writeTestArgoConfig(t *testing.T, serverURL, token string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	// Strip protocol — argocd config uses bare hostname as the server key
	bare := auth.StripProtocol(serverURL)
	content := fmt.Sprintf(`contexts:
  - name: default
    server: %s
    user: default-user
servers:
  - server: %s
    insecure: true
users:
  - name: default-user
    auth-token: %s
current-context: default
`, bare, bare, token)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("writeTestArgoConfig: %v", err)
	}
	return p
}

// --- Tests ---

func TestTriggerReauthAlreadyPending_NoOp(t *testing.T) {
	m := NewModel(nil)
	m.state.Mode = model.ModeReauthPending

	result, cmd := m.Update(model.TriggerReauthMsg{})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeReauthPending {
		t.Errorf("expected mode unchanged (ModeReauthPending), got %s", newM.state.Mode)
	}
	if cmd != nil {
		t.Error("expected nil cmd for already-pending no-op")
	}
}

func TestTriggerReauthInfiniteLoopGuard(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: ""}
	m.jwtAuthProvider = &fakeAuthProvider{cmd: exec.Command("true")}
	m.reauthAttempts = 2 // after increment (→3), this exceeds the > 2 limit, triggering fallback

	result, _ := m.Update(model.TriggerReauthMsg{})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeAuthRequired {
		t.Errorf("expected ModeAuthRequired after loop guard, got %s", newM.state.Mode)
	}
	if newM.reauthAttempts != 0 {
		t.Errorf("expected reauthAttempts reset to 0 after loop guard, got %d", newM.reauthAttempts)
	}
}

func TestTriggerReauthSetsModePendingAndLaunchesExecProcess(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: ""}
	m.jwtAuthProvider = &fakeAuthProvider{cmd: exec.Command("true")}

	result, cmd := m.Update(model.TriggerReauthMsg{})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeReauthPending {
		t.Errorf("expected ModeReauthPending, got %s", newM.state.Mode)
	}
	if newM.reauthAttempts != 1 {
		t.Errorf("expected reauthAttempts=1, got %d", newM.reauthAttempts)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (ExecProcess) from TriggerReauthMsg")
	}
}

func TestReauthCompleteSuccess_UpdatesTokenAndResetsAttempts(t *testing.T) {
	const freshToken = "fresh-reauth-token-99"
	cfgPath := writeTestArgoConfig(t, "https://argocd.example.com", freshToken)

	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: ""}
	m.state.Mode = model.ModeReauthPending
	m.reauthAttempts = 1
	m.switchEpoch = 1
	m.argoConfigPath = cfgPath
	m.currentContextName = "default"

	result, cmd := m.Update(model.ReauthCompleteMsg{Err: nil, SwitchEpoch: 1})
	newM := result.(*Model)

	if newM.reauthAttempts != 0 {
		t.Errorf("expected reauthAttempts reset to 0, got %d", newM.reauthAttempts)
	}
	if newM.state.Server == nil || newM.state.Server.Token != freshToken {
		t.Errorf("expected token %q, got %q", freshToken, newM.state.Server.Token)
	}
	if newM.switchEpoch != 2 {
		t.Errorf("expected switchEpoch incremented to 2 after successful reauth, got %d", newM.switchEpoch)
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd (validateAuthentication) after successful reauth")
	}
}

func TestReauthCompleteFailure_FallsBackToAuthRequired(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: ""}
	m.state.Mode = model.ModeReauthPending
	m.switchEpoch = 1

	result, _ := m.Update(model.ReauthCompleteMsg{
		Err:         errors.New("argocd login: exit status 1"),
		SwitchEpoch: 1,
	})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeAuthRequired {
		t.Errorf("expected ModeAuthRequired on reauth failure, got %s", newM.state.Mode)
	}
}

func TestReauthCompleteStaleEpoch_Discarded(t *testing.T) {
	m := NewModel(nil)
	m.state.Mode = model.ModeReauthPending
	m.switchEpoch = 5

	result, cmd := m.Update(model.ReauthCompleteMsg{Err: nil, SwitchEpoch: 3})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeReauthPending {
		t.Errorf("expected mode unchanged, got %s", newM.state.Mode)
	}
	if cmd != nil {
		t.Error("expected nil cmd for stale ReauthCompleteMsg")
	}
}

func TestReauthViewMessage(t *testing.T) {
	m := NewModel(nil)
	m.state.Terminal.Rows = 24
	m.state.Terminal.Cols = 80
	m.ready = true
	m.state.Mode = model.ModeReauthPending

	view := m.View().Content
	if !strings.Contains(view, "Re-authenticating via SSO") {
		t.Errorf("expected 'Re-authenticating via SSO' in ModeReauthPending view, got:\n%s", view)
	}
}
