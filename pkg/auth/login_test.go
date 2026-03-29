package auth

import (
	"testing"
)

func containsArg(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

func argAfter(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestStripProtocol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://foo.com", "foo.com"},
		{"http://foo.com:8080", "foo.com:8080"},
		{"foo.com", "foo.com"},
		{"", ""},
	}
	for _, tc := range tests {
		got := StripProtocol(tc.input)
		if got != tc.want {
			t.Errorf("StripProtocol(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBuildLoginCmd_SSO(t *testing.T) {
	p := ArgocdCLIAuthProvider{}
	cmd := p.LoginCmd(LoginParams{
		ServerURL:   "argocd.example.com",
		ContextName: "default",
	})
	args := cmd.Args
	if args[0] != "argocd" {
		t.Errorf("expected argocd binary, got %s", args[0])
	}
	if !containsArg(args, "--sso") {
		t.Error("expected --sso flag")
	}
	if !containsArg(args, "argocd.example.com") {
		t.Error("expected server URL in args")
	}
}

func TestBuildLoginCmd_Insecure(t *testing.T) {
	p := ArgocdCLIAuthProvider{}
	cmd := p.LoginCmd(LoginParams{
		ServerURL: "argocd.example.com",
		Insecure:  true,
	})
	if !containsArg(cmd.Args, "--insecure") {
		t.Error("expected --insecure flag when Insecure=true")
	}
	// Must NOT appear when false
	cmd2 := p.LoginCmd(LoginParams{ServerURL: "argocd.example.com", Insecure: false})
	if containsArg(cmd2.Args, "--insecure") {
		t.Error("unexpected --insecure flag when Insecure=false")
	}
}

func TestBuildLoginCmd_GrpcWeb(t *testing.T) {
	p := ArgocdCLIAuthProvider{}
	cmd := p.LoginCmd(LoginParams{
		ServerURL: "argocd.example.com",
		GrpcWeb:   true,
	})
	if !containsArg(cmd.Args, "--grpc-web") {
		t.Error("expected --grpc-web flag when GrpcWeb=true")
	}
	cmd2 := p.LoginCmd(LoginParams{ServerURL: "argocd.example.com", GrpcWeb: false})
	if containsArg(cmd2.Args, "--grpc-web") {
		t.Error("unexpected --grpc-web when GrpcWeb=false")
	}
}

func TestBuildLoginCmd_GrpcWebRootPath(t *testing.T) {
	p := ArgocdCLIAuthProvider{}
	cmd := p.LoginCmd(LoginParams{
		ServerURL:       "argocd.example.com",
		GrpcWebRootPath: "/argo",
	})
	if !containsArg(cmd.Args, "--grpc-web-root-path") {
		t.Error("expected --grpc-web-root-path flag")
	}
	if argAfter(cmd.Args, "--grpc-web-root-path") != "/argo" {
		t.Errorf("expected /argo, got %q", argAfter(cmd.Args, "--grpc-web-root-path"))
	}
}

func TestBuildLoginCmd_CustomConfig(t *testing.T) {
	p := ArgocdCLIAuthProvider{}
	cmd := p.LoginCmd(LoginParams{
		ServerURL:  "argocd.example.com",
		ConfigPath: "/tmp/argocd/config",
	})
	if !containsArg(cmd.Args, "--config") {
		t.Error("expected --config flag when ConfigPath set")
	}
	if argAfter(cmd.Args, "--config") != "/tmp/argocd/config" {
		t.Errorf("expected /tmp/argocd/config, got %q", argAfter(cmd.Args, "--config"))
	}
	// When empty, no --config
	cmd2 := p.LoginCmd(LoginParams{ServerURL: "argocd.example.com", ConfigPath: ""})
	if containsArg(cmd2.Args, "--config") {
		t.Error("unexpected --config when ConfigPath empty")
	}
}

func TestBuildLoginCmd_ContextName(t *testing.T) {
	p := ArgocdCLIAuthProvider{}
	cmd := p.LoginCmd(LoginParams{
		ServerURL:   "argocd.example.com",
		ContextName: "my-ctx",
	})
	if !containsArg(cmd.Args, "--name") {
		t.Error("expected --name flag when ContextName set")
	}
	if argAfter(cmd.Args, "--name") != "my-ctx" {
		t.Errorf("expected my-ctx, got %q", argAfter(cmd.Args, "--name"))
	}
}
