package main

// K9sResourceParams contains the parameters for opening a resource in k9s
type K9sResourceParams struct {
	Kind      string // Kubernetes resource kind (e.g., "Deployment")
	Namespace string // Resource namespace
	Context   string // Kubernetes context to use
	Name      string // Resource name (used as filter in k9s)
}
