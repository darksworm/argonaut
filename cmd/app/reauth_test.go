package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/auth"
	"github.com/darksworm/argonaut/pkg/model"
)

// fakeReauthProvider is a test double for auth.ReauthProvider.
// It writes a token to configPath so handleReauthCompleteMsg can re-read it.
type fakeReauthProvider struct {
	err error
}

func (f *fakeReauthProvider) Reauth(_ context.Context, _ *model.Server, _ string, _ string) (string, error) {
	return "", f.err
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
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: "", SSO: true}
	m.reauthProvider = &fakeReauthProvider{}
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

func TestTriggerReauthSetsModePendingAndLaunchesBgCmd(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: "", SSO: true}
	m.reauthProvider = &fakeReauthProvider{}

	result, cmd := m.Update(model.TriggerReauthMsg{})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeReauthPending {
		t.Errorf("expected ModeReauthPending, got %s", newM.state.Mode)
	}
	if newM.reauthAttempts != 1 {
		t.Errorf("expected reauthAttempts=1, got %d", newM.reauthAttempts)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (background goroutine) from TriggerReauthMsg")
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

func TestValidateAuthentication_EmptyTokenEmitsTriggerReauth(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: ""}
	m.switchEpoch = 1

	cmd := m.validateAuthentication()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from validateAuthentication with empty token")
	}
	msg := cmd()
	if _, ok := msg.(model.TriggerReauthMsg); !ok {
		t.Errorf("expected TriggerReauthMsg from empty token, got %T: %v", msg, msg)
	}
}

func TestAuthErrorMsg_EmitsTriggerReauthWhenServerSet(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: "tok"}
	m.switchEpoch = 1
	m.ready = true

	result, cmd := m.Update(model.AuthErrorMsg{
		Error:       fmt.Errorf("unauthenticated"),
		SwitchEpoch: 1,
	})
	newM := result.(*Model)

	// Mode should NOT immediately switch to ModeAuthRequired
	if newM.state.Mode == model.ModeAuthRequired {
		t.Error("expected mode NOT to be ModeAuthRequired immediately (should emit TriggerReauthMsg instead)")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from AuthErrorMsg")
	}
	msg := cmd()
	if _, ok := msg.(model.TriggerReauthMsg); !ok {
		t.Errorf("expected TriggerReauthMsg, got %T: %v", msg, msg)
	}
}

func TestAuthErrorMsg_FallsBackToAuthRequiredWhenNoServer(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = nil
	m.switchEpoch = 1
	m.ready = true

	result, cmd := m.Update(model.AuthErrorMsg{
		Error:       fmt.Errorf("unauthenticated"),
		SwitchEpoch: 1,
	})
	newM := result.(*Model)

	_ = newM
	// Should emit a cmd that results in ModeAuthRequired (not TriggerReauthMsg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	// The result is SetModeMsg{Mode: ModeAuthRequired}
	setMode, ok := msg.(model.SetModeMsg)
	if !ok {
		t.Errorf("expected SetModeMsg, got %T: %v", msg, msg)
		return
	}
	if setMode.Mode != model.ModeAuthRequired {
		t.Errorf("expected ModeAuthRequired, got %v", setMode.Mode)
	}
}

func TestTriggerReauth_NonSSO_FallsBackToAuthRequired(t *testing.T) {
	m := NewModel(nil)
	// SSO defaults to false — username/password user, no refresh-token in their config
	m.state.Server = &model.Server{BaseURL: "https://argocd.example.com", Token: "tok"}

	result, cmd := m.Update(model.TriggerReauthMsg{})
	newM := result.(*Model)

	if newM.state.Mode != model.ModeAuthRequired {
		t.Errorf("expected ModeAuthRequired for non-SSO server, got %s", newM.state.Mode)
	}
	if cmd != nil {
		t.Error("expected nil cmd for non-SSO server")
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
