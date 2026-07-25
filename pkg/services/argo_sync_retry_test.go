package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestSyncApplication_NetworkError_IsNotRetried(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack failed: %v", err)
			return
		}
		conn.Close()
	}))
	t.Cleanup(server.Close)

	srv := &model.Server{BaseURL: server.URL, Token: "test-token"}
	svc := NewArgoApiService(srv)

	err := svc.SyncApplication(context.Background(), srv, "my-app", nil, false)
	if err == nil {
		t.Fatal("expected an error from a dropped connection, got nil")
	}
	if got := hits.Load(); got != 1 {
		t.Errorf("sync reached the server %d times; a sync that may have executed server-side must not be retried", got)
	}
}
