package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestExtractTarGz_RejectsUnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	unsupportedPayload := []byte("unsupported payload")
	unsupportedType := byte('V')
	if err := tw.WriteHeader(&tar.Header{
		Name:     "bad-entry",
		Mode:     0600,
		Size:     int64(len(unsupportedPayload)),
		Typeflag: unsupportedType,
	}); err != nil {
		t.Fatalf("write unsupported header: %v", err)
	}
	if _, err := tw.Write(unsupportedPayload); err != nil {
		t.Fatalf("write unsupported payload: %v", err)
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:     "should-not-extract",
		Mode:     0644,
		Size:     4,
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("write regular header: %v", err)
	}
	if _, err := tw.Write([]byte("next")); err != nil {
		t.Fatalf("write regular payload: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	dest := t.TempDir()
	u := &UpdateServiceImpl{}
	err := u.extractTarGz(bytes.NewReader(buf.Bytes()), dest)
	if err == nil {
		t.Fatal("expected error for unsupported tar entry type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported tar entry type") {
		t.Fatalf("expected unsupported type error, got: %v", err)
	}
}
