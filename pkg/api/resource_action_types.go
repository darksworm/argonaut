package api

// ResourceActionRequest represents a request to run a resource action via ArgoCD
// This type is required by the existing RunResourceAction method in applications.go
type ResourceActionRequest struct {
	AppName      string  // Name of the ArgoCD application
	AppNamespace *string // Optional namespace of the ArgoCD application (for multi-tenant)
	ResourceName string  // Name of the resource to act on
	Namespace    string  // Namespace of the resource
	Kind         string  // Kind of the resource
	Group        string  // API group of the resource
	Version      string  // API version of the resource
	Action       string  // Action to perform
}