package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestClient_GrpcWebRootPath_URLConstruction(t *testing.T) {
	tests := []struct {
		name             string
		grpcWebRootPath  string
		requestPath      string
		expectedFullPath string
	}{
		{
			name:             "Without grpc-web-root-path",
			grpcWebRootPath:  "",
			requestPath:      "/api/v1/applications",
			expectedFullPath: "/api/v1/applications",
		},
		{
			name:             "With grpc-web-root-path",
			grpcWebRootPath:  "argocd",
			requestPath:      "/api/v1/applications",
			expectedFullPath: "/argocd/api/v1/applications",
		},
		{
			name:             "With grpc-web-root-path with leading slash",
			grpcWebRootPath:  "/argocd",
			requestPath:      "/api/v1/applications",
			expectedFullPath: "/argocd/api/v1/applications",
		},
		{
			name:             "With grpc-web-root-path with trailing slash",
			grpcWebRootPath:  "argocd/",
			requestPath:      "/api/v1/applications",
			expectedFullPath: "/argocd/api/v1/applications",
		},
		{
			name:             "With grpc-web-root-path with both slashes",
			grpcWebRootPath:  "/argocd/",
			requestPath:      "/api/v1/applications",
			expectedFullPath: "/argocd/api/v1/applications",
		},
		{
			name:             "With nested root path",
			grpcWebRootPath:  "k8s/argocd",
			requestPath:      "/api/v1/projects",
			expectedFullPath: "/k8s/argocd/api/v1/projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track the actual request path received by the server
			var receivedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{}"))
			}))
			defer server.Close()

			// Create client with the test configuration
			client := NewClient(&model.Server{
				BaseURL:         server.URL,
				Token:           "test-token",
				GrpcWebRootPath: tt.grpcWebRootPath,
			})

			// Make a GET request to test URL construction
			_, err := client.Get(context.Background(), tt.requestPath)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the server received the expected path
			if receivedPath != tt.expectedFullPath {
				t.Errorf("Expected path %s, got %s", tt.expectedFullPath, receivedPath)
			}
		})
	}
}

func TestClient_GrpcWebRootPath_AllMethods(t *testing.T) {
	// Test that all HTTP methods (GET, POST, PUT, DELETE) use the root path correctly
	expectedPath := "/argocd/api/v1/test"
	grpcWebRootPath := "argocd"
	requestPath := "/api/v1/test"

	methods := []struct {
		name     string
		testFunc func(*Client, context.Context) error
	}{
		{
			name: "GET",
			testFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Get(ctx, requestPath)
				return err
			},
		},
		{
			name: "POST",
			testFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Post(ctx, requestPath, map[string]string{"test": "data"})
				return err
			},
		},
		{
			name: "PUT",
			testFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Put(ctx, requestPath, map[string]string{"test": "data"})
				return err
			},
		},
		{
			name: "DELETE",
			testFunc: func(c *Client, ctx context.Context) error {
				_, err := c.Delete(ctx, requestPath)
				return err
			},
		},
	}

	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			var receivedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{}"))
			}))
			defer server.Close()

			client := NewClient(&model.Server{
				BaseURL:         server.URL,
				Token:           "test-token",
				GrpcWebRootPath: grpcWebRootPath,
			})

			err := method.testFunc(client, context.Background())
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if receivedPath != expectedPath {
				t.Errorf("%s: Expected path %s, got %s", method.name, expectedPath, receivedPath)
			}
		})
	}
}

func TestClient_GrpcWebRootPath_StreamMethod(t *testing.T) {
	expectedPath := "/argocd/api/v1/stream"
	grpcWebRootPath := "argocd"
	requestPath := "/api/v1/stream"

	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: test\n\n"))
	}))
	defer server.Close()

	client := NewClient(&model.Server{
		BaseURL:         server.URL,
		Token:           "test-token",
		GrpcWebRootPath: grpcWebRootPath,
	})

	stream, err := client.Stream(context.Background(), requestPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer stream.Close()

	if receivedPath != expectedPath {
		t.Errorf("Stream: Expected path %s, got %s", expectedPath, receivedPath)
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name            string
		baseURL         string
		grpcWebRootPath string
		path            string
		expectedURL     string
	}{
		{
			name:            "No root path",
			baseURL:         "https://example.com",
			grpcWebRootPath: "",
			path:            "/api/v1/apps",
			expectedURL:     "https://example.com/api/v1/apps",
		},
		{
			name:            "Simple root path",
			baseURL:         "https://example.com",
			grpcWebRootPath: "argocd",
			path:            "/api/v1/apps",
			expectedURL:     "https://example.com/argocd/api/v1/apps",
		},
		{
			name:            "Root path with leading slash",
			baseURL:         "https://example.com",
			grpcWebRootPath: "/argocd",
			path:            "/api/v1/apps",
			expectedURL:     "https://example.com/argocd/api/v1/apps",
		},
		{
			name:            "Root path with trailing slash",
			baseURL:         "https://example.com",
			grpcWebRootPath: "argocd/",
			path:            "/api/v1/apps",
			expectedURL:     "https://example.com/argocd/api/v1/apps",
		},
		{
			name:            "Root path with both slashes",
			baseURL:         "https://example.com",
			grpcWebRootPath: "/argocd/",
			path:            "/api/v1/apps",
			expectedURL:     "https://example.com/argocd/api/v1/apps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				baseURL:         tt.baseURL,
				grpcWebRootPath: tt.grpcWebRootPath,
			}

			result := client.buildURL(tt.path)
			if result != tt.expectedURL {
				t.Errorf("buildURL() = %v, want %v", result, tt.expectedURL)
			}
		})
	}
}