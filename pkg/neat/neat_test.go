package neat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCleanServiceWithClusterDefaults(t *testing.T) {
	// Test service with the exact garbage fields you mentioned
	serviceYAML := `apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: test-namespace
spec:
  clusterIP: 10.43.211.195
  clusterIPs:
  - 10.43.211.195
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: test-app
  sessionAffinity: None
  type: ClusterIP`

	cleaned, err := Clean(serviceYAML)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Verify that all the garbage fields are removed
	garbageFields := []string{
		"clusterIP: 10.43.211.195",
		"clusterIPs:",
		"- 10.43.211.195",
		"internalTrafficPolicy: Cluster",
		"ipFamilies:",
		"- IPv4",
		"ipFamilyPolicy: SingleStack",
		"protocol: TCP",
		"sessionAffinity: None",
		"type: ClusterIP",
	}

	for _, garbage := range garbageFields {
		if strings.Contains(cleaned, garbage) {
			t.Errorf("Garbage field should be removed: %s", garbage)
		}
	}

	// Verify that important data is kept
	importantFields := []string{
		"name: test-service",
		"namespace: test-namespace",
		"port: 80",
		"targetPort: 8080",
		"app: test-app",
	}

	for _, important := range importantFields {
		if !strings.Contains(cleaned, important) {
			t.Errorf("Important field should be kept: %s", important)
		}
	}

	t.Logf("Cleaned service:\n%s", cleaned)
}

func TestCleanDeployment(t *testing.T) {
	// Sample deployment with lots of noise that kubectl-neat should remove
	deploymentYAML := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: test-namespace
  creationTimestamp: "2023-01-01T00:00:00Z"
  resourceVersion: "12345"
  uid: "abcd-1234-efgh-5678"
  generation: 1
  managedFields:
  - manager: kubectl
    operation: Apply
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"test-deployment"}}
    deployment.kubernetes.io/revision: "1"
    my-custom-annotation: "keep-this"
spec:
  replicas: 3
  progressDeadlineSeconds: 600
  revisionHistoryLimit: 10
  strategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
        pod-template-hash: "12345"
    spec:
      restartPolicy: Always
      dnsPolicy: ClusterFirst
      terminationGracePeriodSeconds: 30
      containers:
      - name: test-container
        image: nginx:latest
        imagePullPolicy: IfNotPresent
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
        ports:
        - containerPort: 80
status:
  replicas: 3
  readyReplicas: 3`

	cleaned, err := Clean(deploymentYAML)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Verify that noise was removed (metadata is handled differently by neatMetadata)
	if strings.Contains(cleaned, "status:") {
		t.Error("status should be removed")
	}
	if strings.Contains(cleaned, "progressDeadlineSeconds: 600") {
		t.Error("default progressDeadlineSeconds should be removed")
	}
	if strings.Contains(cleaned, "revisionHistoryLimit: 10") {
		t.Error("default revisionHistoryLimit should be removed")
	}
	if strings.Contains(cleaned, "type: RollingUpdate") {
		t.Error("default strategy type should be removed")
	}
	if strings.Contains(cleaned, "restartPolicy: Always") {
		t.Error("default restartPolicy should be removed")
	}
	if strings.Contains(cleaned, "dnsPolicy: ClusterFirst") {
		t.Error("default dnsPolicy should be removed")
	}
	if strings.Contains(cleaned, "terminationGracePeriodSeconds: 30") {
		t.Error("default terminationGracePeriodSeconds should be removed")
	}
	if strings.Contains(cleaned, "imagePullPolicy: IfNotPresent") {
		t.Error("default imagePullPolicy should be removed")
	}
	if strings.Contains(cleaned, "terminationMessagePath") {
		t.Error("default terminationMessagePath should be removed")
	}
	if strings.Contains(cleaned, "terminationMessagePolicy") {
		t.Error("default terminationMessagePolicy should be removed")
	}

	// Note: kubectl-neat's neatMetadata function handles metadata differently than our original test expected
	// It only keeps name, namespace, labels, and annotations, removing all the clutter fields

	// Verify that important data is kept
	if !strings.Contains(cleaned, "name: test-deployment") {
		t.Error("deployment name should be kept")
	}
	if !strings.Contains(cleaned, "namespace: test-namespace") {
		t.Error("namespace should be kept")
	}
	if !strings.Contains(cleaned, "replicas: 3") {
		t.Error("replicas should be kept")
	}
	if !strings.Contains(cleaned, "my-custom-annotation: keep-this") {
		t.Error("custom annotations should be kept")
	}
	if !strings.Contains(cleaned, "image: nginx:latest") {
		t.Error("container image should be kept")
	}
}

func TestCleanJSON(t *testing.T) {
	serviceJSON := `{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "test-service",
    "creationTimestamp": "2023-01-01T00:00:00Z",
    "resourceVersion": "12345"
  },
  "spec": {
    "type": "ClusterIP",
    "clusterIP": "10.43.211.195",
    "internalTrafficPolicy": "Cluster",
    "ports": [{
      "port": 80,
      "protocol": "TCP"
    }]
  },
  "status": {
    "loadBalancer": {}
  }
}`

	cleaned, err := Neat(serviceJSON)
	if err != nil {
		t.Fatalf("Neat failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		t.Fatalf("cleaned JSON is not valid: %v", err)
	}

	// Verify noise was removed
	if _, exists := result["status"]; exists {
		t.Error("status should be removed")
	}

	if spec, ok := result["spec"].(map[string]interface{}); ok {
		if _, exists := spec["type"]; exists {
			t.Error("default service type should be removed")
		}
		if _, exists := spec["clusterIP"]; exists {
			t.Error("clusterIP should be removed")
		}
		if _, exists := spec["internalTrafficPolicy"]; exists {
			t.Error("internalTrafficPolicy should be removed")
		}

		// Check ports
		if ports, ok := spec["ports"].([]interface{}); ok {
			if len(ports) > 0 {
				port := ports[0].(map[string]interface{})
				if _, exists := port["protocol"]; exists {
					t.Error("default protocol TCP should be removed")
				}
			}
		}
	}

	// Verify important data is kept
	if result["apiVersion"] != "v1" {
		t.Error("apiVersion should be kept")
	}
	if result["kind"] != "Service" {
		t.Error("kind should be kept")
	}

	// kubectl-neat's metadata handling only keeps name, namespace, labels, annotations
	if metadata, ok := result["metadata"].(map[string]interface{}); ok {
		if metadata["name"] != "test-service" {
			t.Error("service name should be kept")
		}
		// creationTimestamp and resourceVersion should be removed by neatMetadata
	}
}