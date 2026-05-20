package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

type tarEntry struct {
	name     string
	body     string
	typeflag byte
}

func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0o644,
			Size:     int64(len(e.body)),
			Typeflag: e.typeflag,
		}
		if e.typeflag == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write tar body: %v", err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	svc := &UpdateServiceImpl{}
	destDir := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{name: "../evil.txt", body: "owned"},
	})

	err := svc.extractTarGz(bytes.NewReader(archive), destDir)
	if err == nil {
		t.Fatal("expected path traversal archive to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(destDir, "..", "evil.txt")); statErr == nil {
		t.Fatal("unexpected file created outside extraction directory")
	}
}

func TestExtractTarGzRejectsSymlink(t *testing.T) {
	svc := &UpdateServiceImpl{}
	destDir := t.TempDir()
	archive := buildTarGz(t, []tarEntry{
		{name: "link", typeflag: tar.TypeSymlink},
	})

	err := svc.extractTarGz(bytes.NewReader(archive), destDir)
	if err == nil {
		t.Fatal("expected symlink archive entry to be rejected")
	}
}

func TestExtractTarGzRejectsUnknownTypeWithPayload(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	payload := []byte("unsupported payload")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "unknown",
		Mode:     0o600,
		Size:     int64(len(payload)),
		Typeflag: byte('V'),
	}); err != nil {
		t.Fatalf("write unknown header: %v", err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatalf("write unknown payload: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	svc := &UpdateServiceImpl{}
	err := svc.extractTarGz(bytes.NewReader(buf.Bytes()), t.TempDir())
	if err == nil {
		t.Fatal("expected unknown tar type to be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported tar entry type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseChecksumForAsset(t *testing.T) {
	content := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  argonaut-v1.0.0-linux-amd64.tar.gz\n"
	got := parseChecksumForAsset(content, "argonaut-v1.0.0-linux-amd64.tar.gz")
	want := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("abc")
	// sha256("abc")
	const good = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if err := verifySHA256(data, good); err != nil {
		t.Fatalf("expected checksum to verify: %v", err)
	}
	if err := verifySHA256(data, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestDownloadAndReplaceRejectsMissingChecksum(t *testing.T) {
	svc := &UpdateServiceImpl{}
	err := svc.DownloadAndReplace(nil)
	if err == nil {
		t.Fatal("expected nil updateInfo to fail")
	}

	err = svc.DownloadAndReplace(&model.UpdateInfo{
		DownloadURL: "https://example.com/argonaut.tar.gz",
	})
	if err == nil {
		t.Fatal("expected missing checksum to fail")
	}
}

// --- isVersionNewer ---------------------------------------------------------

func TestIsVersionNewer(t *testing.T) {
	tests := []struct {
		newV, currentV string
		want           bool
	}{
		// dev → always upgrade
		{"v1.0.0", "dev", true},
		{"1.0.0", "dev", true},

		// same → no
		{"v1.0.0", "v1.0.0", false},
		{"1.0.0", "v1.0.0", false},
		{"v1.0.0", "1.0.0", false},

		// strict greater
		{"v1.0.1", "v1.0.0", true},
		{"v1.1.0", "v1.0.9", true},
		{"v2.0.0", "v1.99.99", true},
		{"v1.0.0", "v0.99.99", true},

		// strict less
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.9", "v1.1.0", false},
		{"v1.99.99", "v2.0.0", false},

		// length mismatch (padded with zeros)
		{"v1.0", "v1.0.0", false},
		{"v1.0.0.1", "v1.0.0", true},
		{"v1.0", "v1.0.1", false},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tc.newV, tc.currentV), func(t *testing.T) {
			if got := isVersionNewer(tc.newV, tc.currentV); got != tc.want {
				t.Errorf("isVersionNewer(%q, %q) = %v, want %v", tc.newV, tc.currentV, got, tc.want)
			}
		})
	}
}

// --- findDownloadURL --------------------------------------------------------

func TestFindDownloadURL_PrefersExactPlatformMatch(t *testing.T) {
	svc := &UpdateServiceImpl{}
	release := &GitHubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		}{
			{Name: "argonaut-1.0.0-windows-amd64.zip", BrowserDownloadURL: "https://example.com/win.zip"},
			{Name: fmt.Sprintf("argonaut-1.0.0-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH), BrowserDownloadURL: "https://example.com/native.tar.gz"},
			{Name: "argonaut-1.0.0-darwin-arm64.tar.gz", BrowserDownloadURL: "https://example.com/mac.tar.gz"},
		},
	}
	got := svc.findDownloadURL(release)
	if got != "https://example.com/native.tar.gz" {
		t.Errorf("expected native asset, got %q", got)
	}
}

func TestFindDownloadURL_NoMatchReturnsEmpty(t *testing.T) {
	svc := &UpdateServiceImpl{}
	release := &GitHubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		}{
			{Name: "argonaut-1.0.0-plan9-mips.tar.gz", BrowserDownloadURL: "https://example.com/plan9.tar.gz"},
		},
	}
	if got := svc.findDownloadURL(release); got != "" {
		t.Errorf("expected empty URL when no platform matches, got %q", got)
	}
}

func TestFindDownloadURL_AmdAlias(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("alias test only meaningful on amd64")
	}
	svc := &UpdateServiceImpl{}
	release := &GitHubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		}{
			{Name: fmt.Sprintf("argonaut-1.0.0-%s-x86_64.tar.gz", runtime.GOOS), BrowserDownloadURL: "https://example.com/aliased.tar.gz"},
		},
	}
	got := svc.findDownloadURL(release)
	if got != "https://example.com/aliased.tar.gz" {
		t.Errorf("expected x86_64 alias to match amd64, got %q", got)
	}
}

// --- findChecksumAssetURL ---------------------------------------------------

func TestFindChecksumAssetURL(t *testing.T) {
	tests := []struct {
		name   string
		assets []string
		want   string
	}{
		{"sha256sums file", []string{"argonaut.tar.gz", "sha256sums.txt"}, "sha256sums.txt"},
		{"checksums.txt file", []string{"argonaut.tar.gz", "checksums.txt"}, "checksums.txt"},
		{"no checksum asset", []string{"argonaut.tar.gz", "release-notes.md"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &GitHubRelease{}
			for _, n := range tc.assets {
				r.Assets = append(r.Assets, struct {
					Name               string `json:"name"`
					BrowserDownloadURL string `json:"browser_download_url"`
					Size               int64  `json:"size"`
				}{Name: n, BrowserDownloadURL: n})
			}
			got := findChecksumAssetURL(r)
			if got != tc.want {
				t.Errorf("findChecksumAssetURL = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- CheckForUpdates (HTTP-mocked) ------------------------------------------

// newUpdateServiceWithGitHubBase returns an UpdateService wired to call the
// given base URL for both the release-info request and the checksum-asset
// request. It does this by replacing the default http.Client transport with
// one that rewrites the request URL on the fly, so the production code path
// (`https://api.github.com/...`) is exercised unchanged.
func newUpdateServiceWithGitHubBase(t *testing.T, baseURL string) UpdateService {
	t.Helper()
	httpClient := &http.Client{
		Transport: &rewriteTransport{base: baseURL, inner: http.DefaultTransport},
	}
	return NewUpdateService(UpdateServiceConfig{
		HTTPClient:       httpClient,
		GitHubRepo:       "darksworm/argonaut",
		CheckIntervalMin: 60,
	})
}

type rewriteTransport struct {
	base  string
	inner http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite api.github.com → mock host; keep path+query intact.
	if req.URL.Host == "api.github.com" {
		newURL := rt.base + req.URL.Path
		if req.URL.RawQuery != "" {
			newURL += "?" + req.URL.RawQuery
		}
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header.Clone()
		return rt.inner.RoundTrip(newReq)
	}
	return rt.inner.RoundTrip(req)
}

func TestCheckForUpdates_NewerRelease_ReportsAvailable(t *testing.T) {
	releaseJSON := fmt.Sprintf(`{
		"tag_name": "v2.0.0",
		"name": "v2.0.0",
		"published_at": "2026-01-01T00:00:00Z",
		"assets": [
			{"name": "argonaut-2.0.0-%s-%s.tar.gz", "browser_download_url": "%%s/dl/argonaut-2.0.0-%s-%s.tar.gz", "size": 100},
			{"name": "checksums.txt", "browser_download_url": "%%s/dl/checksums.txt", "size": 64}
		]
	}`, runtime.GOOS, runtime.GOARCH, runtime.GOOS, runtime.GOARCH)

	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/darksworm/argonaut/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, releaseJSON, srv.URL, srv.URL)
	})
	mux.HandleFunc("/dl/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  argonaut-2.0.0-%s-%s.tar.gz\n", runtime.GOOS, runtime.GOARCH)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	svc := newUpdateServiceWithGitHubBase(t, srv.URL)
	info, err := svc.CheckForUpdates("v1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if !info.Available {
		t.Errorf("expected Available=true; v2.0.0 > v1.0.0")
	}
	if info.CurrentVersion != "v1.0.0" {
		t.Errorf("CurrentVersion = %q, want v1.0.0", info.CurrentVersion)
	}
	if info.LatestVersion != "v2.0.0" {
		t.Errorf("LatestVersion = %q, want v2.0.0", info.LatestVersion)
	}
	if info.DownloadURL == "" {
		t.Errorf("expected DownloadURL to be populated for available update")
	}
	if info.ChecksumSHA256 != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Errorf("ChecksumSHA256 = %q, want the hex from the mock", info.ChecksumSHA256)
	}
}

func TestCheckForUpdates_SameVersion_ReportsUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/darksworm/argonaut/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"tag_name": "v1.0.0",
			"name": "v1.0.0",
			"published_at": "2026-01-01T00:00:00Z",
			"assets": [
				{"name": "argonaut-1.0.0-%s-%s.tar.gz", "browser_download_url": "%s/dl/argonaut.tar.gz", "size": 100}
			]
		}`, runtime.GOOS, runtime.GOARCH, srv.URL)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	svc := newUpdateServiceWithGitHubBase(t, srv.URL)
	info, err := svc.CheckForUpdates("v1.0.0")
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if info.Available {
		t.Errorf("expected Available=false when current == latest")
	}
	if info.DownloadURL != "" {
		t.Errorf("expected no DownloadURL when no update is available, got %q", info.DownloadURL)
	}
}

func TestCheckForUpdates_DevVersion_ReportsAvailable(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/darksworm/argonaut/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"tag_name": "v0.0.1",
			"name": "v0.0.1",
			"published_at": "2026-01-01T00:00:00Z",
			"assets": [
				{"name": "argonaut-0.0.1-%s-%s.tar.gz", "browser_download_url": "%s/dl/x.tar.gz", "size": 1}
			]
		}`, runtime.GOOS, runtime.GOARCH, srv.URL)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	svc := newUpdateServiceWithGitHubBase(t, srv.URL)
	info, err := svc.CheckForUpdates("dev")
	if err != nil {
		t.Fatalf("CheckForUpdates: %v", err)
	}
	if !info.Available {
		t.Errorf("expected Available=true for dev currentVersion (any release should look newer)")
	}
}

func TestCheckForUpdates_GitHubError_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/darksworm/argonaut/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newUpdateServiceWithGitHubBase(t, srv.URL)
	if _, err := svc.CheckForUpdates("v1.0.0"); err == nil {
		t.Error("expected error when GitHub returns non-200")
	}
}

func TestCheckForUpdates_BadJSON_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/darksworm/argonaut/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newUpdateServiceWithGitHubBase(t, srv.URL)
	if _, err := svc.CheckForUpdates("v1.0.0"); err == nil {
		t.Error("expected error when GitHub returns malformed JSON")
	}
}
