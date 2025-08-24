/**
 * Example usage of the Kubernetes field stripper
 * 
 * This demonstrates how the stripper cleans up ArgoCD resource data
 * to produce meaningful diffs focused on actual changes.
 */

import { stripKubernetesFields, stripArgoCDDiff } from './kube-strip';

// Example ArgoCD resource data (like what you get from the API)
const exampleArgoCDResource = {
  "apiVersion": "v1",
  "kind": "Service", 
  "metadata": {
    "annotations": {
      "argocd.argoproj.io/tracking-id": "diagon-ollivanders:/Service:diagon-dev/ollivanders--guestbook-ui",
      "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Service\"}",
      "deployment.kubernetes.io/revision": "1"
    },
    "creationTimestamp": "2025-08-12T20:42:04Z",
    "labels": {
      "app.kubernetes.io/instance": "diagonollivanders"
    },
    "managedFields": [
      {
        "apiVersion": "v1",
        "fieldsType": "FieldsV1",
        "manager": "argocd-controller",
        "operation": "Update"
      }
    ],
    "name": "ollivanders--guestbook-ui",
    "namespace": "diagon-dev",
    "resourceVersion": "46492",
    "uid": "36cd5d20-16ce-4896-b9e8-bf4a772f686f"
  },
  "spec": {
    "clusterIP": "10.43.214.96",
    "clusterIPs": ["10.43.214.96"], 
    "internalTrafficPolicy": "Cluster",
    "ipFamilies": ["IPv4"],
    "ipFamilyPolicy": "SingleStack",
    "ports": [
      {
        "port": 80,
        "protocol": "TCP",
        "targetPort": 80
      }
    ],
    "selector": {
      "app": "guestbook-ui",
      "app.kubernetes.io/instance": "diagonollivanders"
    },
    "sessionAffinity": "None",
    "type": "ClusterIP"
  },
  "status": {
    "loadBalancer": {}
  }
};

// Example Deployment with more complex stripping scenarios
const exampleDeployment = {
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "annotations": {
      "argocd.argoproj.io/tracking-id": "diagon-ollivanders:apps/Deployment:diagon-dev/ollivanders--guestbook-ui",
      "deployment.kubernetes.io/revision": "1",
      "kubectl.kubernetes.io/last-applied-configuration": "{\"big-json-blob\":\"here\"}"
    },
    "creationTimestamp": "2025-08-12T20:42:04Z",
    "generation": 1,
    "managedFields": [/* large array of system data */],
    "name": "ollivanders--guestbook-ui",
    "namespace": "diagon-dev",
    "resourceVersion": "119206",
    "uid": "32eb7382-1573-4fb5-8106-6fa9ab7a3e5e"
  },
  "spec": {
    "progressDeadlineSeconds": 600, // This is a default value
    "replicas": 1,
    "revisionHistoryLimit": 3,
    "selector": {
      "matchLabels": {
        "app": "guestbook-ui",
        "app.kubernetes.io/instance": "diagonollivanders"
      }
    },
    "strategy": {
      "rollingUpdate": {
        "maxSurge": "25%",
        "maxUnavailable": "25%"
      },
      "type": "RollingUpdate"
    },
    "template": {
      "metadata": {
        "creationTimestamp": null,
        "labels": {
          "app": "guestbook-ui",
          "app.kubernetes.io/instance": "diagonollivanders"
        }
      },
      "spec": {
        "containers": [
          {
            "image": "gcr.io/google-samples/gb-frontend:v5",
            "imagePullPolicy": "IfNotPresent", // Default value
            "name": "guestbook-ui",
            "ports": [
              {
                "containerPort": 80,
                "protocol": "TCP"
              }
            ],
            "resources": {}, // Empty default
            "terminationMessagePath": "/dev/termination-log", // Default value
            "terminationMessagePolicy": "File", // Default value
            "volumeMounts": [
              {
                "name": "kube-api-access-abc123", // Default service account token - should be removed
                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
              }
            ]
          }
        ],
        "dnsPolicy": "ClusterFirst", // Default value
        "restartPolicy": "Always", // Default value  
        "schedulerName": "default-scheduler", // Default value
        "securityContext": {}, // Empty default
        "serviceAccountName": "default", // Default value
        "terminationGracePeriodSeconds": 30, // Default value
        "volumes": [
          {
            "name": "kube-api-access-abc123", // Default service account token - should be removed
            "projected": {
              "defaultMode": 420,
              "sources": [
                {
                  "serviceAccountToken": {
                    "expirationSeconds": 3607,
                    "path": "token"
                  }
                }
              ]
            }
          }
        ]
      }
    }
  },
  "status": {
    "availableReplicas": 1,
    "conditions": [/* runtime status */],
    "observedGeneration": 1,
    "readyReplicas": 1,
    "replicas": 1,
    "updatedReplicas": 1
  }
};

function demonstrateStripping() {
  console.log('=== BEFORE STRIPPING ===');
  console.log('Service size:', JSON.stringify(exampleArgoCDResource).length, 'characters');
  console.log('Deployment size:', JSON.stringify(exampleDeployment).length, 'characters');
  
  console.log('\n=== AFTER STRIPPING ===');
  const strippedService = stripKubernetesFields(exampleArgoCDResource);
  const strippedDeployment = stripKubernetesFields(exampleDeployment);
  
  console.log('Service size:', JSON.stringify(strippedService).length, 'characters');
  console.log('Deployment size:', JSON.stringify(strippedDeployment).length, 'characters');
  
  console.log('\n=== CLEANED SERVICE ===');
  console.log(JSON.stringify(strippedService, null, 2));
  
  console.log('\n=== CLEANED DEPLOYMENT (key fields) ==='); 
  console.log('Metadata keys:', Object.keys(strippedDeployment.metadata || {}));
  console.log('Spec keys:', Object.keys(strippedDeployment.spec || {}));
  console.log('Status removed:', !('status' in strippedDeployment));
  console.log('Default values removed from spec.template.spec:');
  const templateSpec = strippedDeployment?.spec?.template?.spec;
  if (templateSpec) {
    console.log('  dnsPolicy removed:', !('dnsPolicy' in templateSpec));
    console.log('  serviceAccountName removed:', !('serviceAccountName' in templateSpec)); 
    console.log('  volumes with default tokens removed:', !templateSpec.volumes || templateSpec.volumes.length === 0);
  }
}

// Example ArgoCD diff API response format
const exampleArgoCDDiffData = [
  {
    "kind": "Service",
    "namespace": "diagon-dev", 
    "name": "ollivanders--guestbook-ui",
    "targetState": JSON.stringify(exampleArgoCDResource),
    "liveState": JSON.stringify({
      ...exampleArgoCDResource,
      spec: {
        ...exampleArgoCDResource.spec,
        ports: [{ port: 8080, targetPort: 8080, protocol: "TCP" }] // Different port
      }
    }),
    "resourceVersion": "46492"
  }
];

function demonstrateArgoCDDiffStripping() {
  console.log('\n=== ARGOCD DIFF STRIPPING ===');
  console.log('Original diff data size:', JSON.stringify(exampleArgoCDDiffData).length, 'characters');
  
  const strippedDiff = stripArgoCDDiff(exampleArgoCDDiffData);
  console.log('Stripped diff data size:', JSON.stringify(strippedDiff).length, 'characters');
  
  // The stripped data will have cleaner targetState and liveState JSON
  const originalTarget = JSON.parse(exampleArgoCDDiffData[0].targetState);
  const strippedTarget = JSON.parse(strippedDiff[0].targetState);
  
  console.log('Original target keys:', Object.keys(originalTarget));
  console.log('Stripped target keys:', Object.keys(strippedTarget));
  console.log('Metadata keys reduced from', Object.keys(originalTarget.metadata).length, 'to', Object.keys(strippedTarget.metadata || {}).length);
}

// Uncomment to run examples:
// demonstrateStripping();
// demonstrateArgoCDDiffStripping();

export {
  demonstrateStripping,
  demonstrateArgoCDDiffStripping,
  exampleArgoCDResource,
  exampleDeployment
};