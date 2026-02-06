package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
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
