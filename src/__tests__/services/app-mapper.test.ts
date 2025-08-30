import { describe, expect, test } from "bun:test";
import { appToItem } from "../../services/app-mapper";
import type { ArgoApplication } from "../../types/argo";

describe("app-mapper", () => {
  describe("appToItem", () => {
    test("should map complete ArgoApplication to AppItem", () => {
      const argoApp: ArgoApplication = {
        metadata: {
          name: "test-app",
          namespace: "argocd",
        },
        spec: {
          project: "default",
          destination: {
            server: "https://kubernetes.default.svc",
            namespace: "default",
            name: "in-cluster",
          },
        },
        status: {
          sync: {
            status: "Synced",
          },
          health: {
            status: "Healthy",
          },
          history: [
            {
              deployedAt: "2023-12-01T10:00:00Z",
            },
          ],
          operationState: {
            finishedAt: "2023-12-01T09:30:00Z",
          },
          reconciledAt: "2023-12-01T09:00:00Z",
        },
      };

      const result = appToItem(argoApp);

      expect(result).toEqual({
        name: "test-app",
        sync: "Synced",
        health: "Healthy",
        lastSyncAt: "2023-12-01T10:00:00Z",
        project: "default",
        clusterId: "in-cluster",
        clusterLabel: "in-cluster",
        namespace: "default",
        appNamespace: "argocd",
      });
    });

    test("should handle missing metadata fields", () => {
      const argoApp: ArgoApplication = {
        metadata: {},
        spec: {},
        status: {},
      };

      const result = appToItem(argoApp);

      expect(result).toEqual({
        name: "",
        sync: "Unknown",
        health: "Unknown",
        lastSyncAt: undefined,
        project: undefined,
        clusterId: undefined,
        clusterLabel: "unknown",
        namespace: undefined,
        appNamespace: undefined,
      });
    });

    test("should prioritize deployedAt over finishedAt for lastSyncAt", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {},
        status: {
          history: [{ deployedAt: "2023-12-01T10:00:00Z" }],
          operationState: { finishedAt: "2023-12-01T09:30:00Z" },
          reconciledAt: "2023-12-01T09:00:00Z",
        },
      };

      const result = appToItem(argoApp);
      expect(result.lastSyncAt).toBe("2023-12-01T10:00:00Z");
    });

    test("should prioritize finishedAt over reconciledAt when deployedAt is missing", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {},
        status: {
          operationState: { finishedAt: "2023-12-01T09:30:00Z" },
          reconciledAt: "2023-12-01T09:00:00Z",
        },
      };

      const result = appToItem(argoApp);
      expect(result.lastSyncAt).toBe("2023-12-01T09:30:00Z");
    });

    test("should fallback to reconciledAt when other timestamps are missing", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {},
        status: {
          reconciledAt: "2023-12-01T09:00:00Z",
        },
      };

      const result = appToItem(argoApp);
      expect(result.lastSyncAt).toBe("2023-12-01T09:00:00Z");
    });

    test("should prefer cluster name over server URL for cluster identification", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {
          destination: {
            name: "production-cluster",
            server: "https://prod.k8s.example.com",
          },
        },
        status: {},
      };

      const result = appToItem(argoApp);
      expect(result.clusterId).toBe("production-cluster");
      expect(result.clusterLabel).toBe("production-cluster");
    });

    test("should extract host from server URL when cluster name is missing", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {
          destination: {
            server: "https://prod.k8s.example.com",
          },
        },
        status: {},
      };

      const result = appToItem(argoApp);
      expect(result.clusterId).toBe("prod.k8s.example.com");
      expect(result.clusterLabel).toBe("prod.k8s.example.com");
    });

    test("should handle server URL without host extraction fallback", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {
          destination: {
            server: "invalid-url",
          },
        },
        status: {},
      };

      const result = appToItem(argoApp);
      expect(result.clusterId).toBe("invalid-url");
      expect(result.clusterLabel).toBe("invalid-url");
    });

    test("should handle empty history array", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "test-app" },
        spec: {},
        status: {
          history: [],
          operationState: { finishedAt: "2023-12-01T09:30:00Z" },
        },
      };

      const result = appToItem(argoApp);
      expect(result.lastSyncAt).toBe("2023-12-01T09:30:00Z");
    });

    test("should handle null/undefined nested objects", () => {
      const argoApp: ArgoApplication = {
        metadata: null as any,
        spec: null as any,
        status: null as any,
      };

      const result = appToItem(argoApp);
      expect(result).toEqual({
        name: "",
        sync: "Unknown",
        health: "Unknown",
        lastSyncAt: undefined,
        project: undefined,
        clusterId: undefined,
        clusterLabel: "unknown",
        namespace: undefined,
        appNamespace: undefined,
      });
    });

    test("should handle mixed valid and invalid data", () => {
      const argoApp: ArgoApplication = {
        metadata: {
          name: "valid-app",
          namespace: "valid-namespace",
        },
        spec: {
          destination: {}, // empty destination
        },
        status: {
          sync: { status: "OutOfSync" },
          health: null as any, // null health
        },
      };

      const result = appToItem(argoApp);
      expect(result.name).toBe("valid-app");
      expect(result.appNamespace).toBe("valid-namespace");
      expect(result.sync).toBe("OutOfSync");
      expect(result.health).toBe("Unknown");
      expect(result.clusterLabel).toBe("unknown");
    });

    test("should handle complex nested status objects", () => {
      const argoApp: ArgoApplication = {
        metadata: { name: "complex-app" },
        spec: {
          destination: {
            server: "https://kubernetes.default.svc",
          },
        },
        status: {
          sync: {
            status: "OutOfSync",
            revision: "abc123",
          },
          health: {
            status: "Degraded",
            message: "Pod crashed",
          },
          history: [
            { deployedAt: "2023-12-01T10:00:00Z", revision: "abc123" },
            { deployedAt: "2023-11-30T10:00:00Z", revision: "def456" },
          ],
        },
      };

      const result = appToItem(argoApp);
      expect(result.sync).toBe("OutOfSync");
      expect(result.health).toBe("Degraded");
      expect(result.lastSyncAt).toBe("2023-12-01T10:00:00Z");
    });

    test("should handle special server URLs", () => {
      const testCases = [
        {
          server: "https://kubernetes.default.svc",
          expectedId: "kubernetes.default.svc",
        },
        {
          server: "https://127.0.0.1:6443",
          expectedId: "127.0.0.1:6443",
        },
        {
          server: "http://localhost:8080",
          expectedId: "localhost:8080",
        },
      ];

      testCases.forEach(({ server, expectedId }) => {
        const argoApp: ArgoApplication = {
          metadata: { name: "test-app" },
          spec: { destination: { server } },
          status: {},
        };

        const result = appToItem(argoApp);
        expect(result.clusterId).toBe(expectedId);
        expect(result.clusterLabel).toBe(expectedId);
      });
    });

    test("should handle undefined ArgoApplication gracefully", () => {
      const result = appToItem(undefined as any);

      expect(result).toEqual({
        name: "",
        sync: "Unknown",
        health: "Unknown",
        lastSyncAt: undefined,
        project: undefined,
        clusterId: undefined,
        clusterLabel: "unknown",
        namespace: undefined,
        appNamespace: undefined,
      });
    });
  });
});
