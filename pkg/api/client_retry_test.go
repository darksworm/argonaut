package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// newConnectionDroppingServer returns a server that abruptly closes the
// connection on every request, producing a retryable network error, plus a
// counter of how many requests actually arrived.
func newConnectionDroppingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
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
	return server, &hits
}

func TestMutatingRequests_NetworkError_AreNotRetried(t *testing.T) {
	methods := []struct {
		name string
		call func(*Client, context.Context) error
	}{
		{"POST", func(c *Client, ctx context.Context) error {
			_, err := c.Post(ctx, "/api/v1/applications/test/sync", nil)
			return err
		}},
		{"PUT", func(c *Client, ctx context.Context) error {
			_, err := c.Put(ctx, "/api/v1/applications/test", map[string]string{"k": "v"})
			return err
		}},
		{"DELETE", func(c *Client, ctx context.Context) error {
			_, err := c.Delete(ctx, "/api/v1/applications/test")
			return err
		}},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			server, hits := newConnectionDroppingServer(t)
			client := NewClient(&model.Server{BaseURL: server.URL, Token: "test-token"})

			err := m.call(client, context.Background())
			if err == nil {
				t.Fatal("expected an error from a dropped connection, got nil")
			}
			if got := hits.Load(); got != 1 {
				t.Errorf("%s reached the server %d times; a request that may have executed server-side must not be retried", m.name, got)
			}
		})
	}
}

func TestGet_NetworkError_IsRetried(t *testing.T) {
	server, hits := newConnectionDroppingServer(t)
	client := NewClient(&model.Server{BaseURL: server.URL, Token: "test-token"})

	_, err := client.Get(context.Background(), "/api/v1/applications")
	if err == nil {
		t.Fatal("expected an error from a dropped connection, got nil")
	}
	if got := hits.Load(); got < 2 {
		t.Errorf("GET reached the server %d times; idempotent reads should be retried", got)
	}
}
