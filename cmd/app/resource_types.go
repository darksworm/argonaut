package main

// ResourceIdentifier contains the parameters to identify a Kubernetes resource
type ResourceIdentifier struct {
	AppName   string // ArgoCD application name
	Group     string // Kubernetes API group (e.g., "apps", "")
	Kind      string // Kubernetes resource kind (e.g., "Deployment")
	Namespace string // Resource namespace
	Name      string // Resource name
}

// TLSConfig contains TLS certificate configuration
type TLSConfig struct {
	CACertFile     string // Path to CA certificate file
	CACertDir      string // Path to CA certificate directory
	ClientCertFile string // Path to client certificate file
	ClientKeyFile  string // Path to client key file
}

// DeleteOptions contains options for deleting resources
type DeleteOptions struct {
	Cascade           bool   // Whether to cascade delete dependent resources
	PropagationPolicy string // Deletion propagation policy
	Force             bool   // Force deletion (for resource deletion)
}

// AppDeleteParams contains parameters for deleting an application
type AppDeleteParams struct {
	AppName   string  // Application name
	Namespace *string // Application namespace (optional)
	Options   DeleteOptions
}
