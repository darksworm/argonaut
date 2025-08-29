// src/__tests__/bugs/sync-command-wrong-app.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";
import { SyncCommand } from "../../commands/application";
import { createMockContext, createMockState } from "../test-utils";

/**
 * Test to reproduce and verify fix for the bug where the sync command
 * targets the wrong app when executed from non-apps views.
 */
describe("Sync Command Wrong App Bug", () => {
  let syncCommand: SyncCommand;

  beforeEach(() => {
    syncCommand = new SyncCommand();
  });

  it("should fix the getVisibleItems to return correct data for namespaces view", () => {
    const mockApps = [
      { name: "app-in-default", namespace: "default", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-in-kube-system", namespace: "kube-system", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-in-monitoring", namespace: "monitoring", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
    ];

    const context = createMockContext({
      state: createMockState({
        navigation: { view: "namespaces", selectedIdx: 1, lastGPressed: 0, lastEscPressed: 0 }, 
        apps: mockApps,
        server: { config: { baseUrl: "https://test.com" }, token: "test-token" },
      }),
    });

    // @ts-expect-error - accessing private method for testing
    const visibleItems = syncCommand.getVisibleItems(context);
    
    // Fixed: in namespaces view, getVisibleItems now returns the actual namespaces
    expect(visibleItems).toEqual(["default", "kube-system", "monitoring"]); 
    expect(visibleItems.length).toBe(3); 
    expect(visibleItems[1]).toBe("kube-system"); // selectedIdx=1 should refer to kube-system namespace
  });

  it("should handle apps view correctly and return app objects", () => {
    const mockApps = [
      { name: "app-one", namespace: "default", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-two", namespace: "kube-system", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
    ];

    const context = createMockContext({
      state: createMockState({
        navigation: { view: "apps", selectedIdx: 1, lastGPressed: 0, lastEscPressed: 0 }, 
        apps: mockApps,
        server: { config: { baseUrl: "https://test.com" }, token: "test-token" },
      }),
    });

    // @ts-expect-error - accessing private method for testing
    const visibleItems = syncCommand.getVisibleItems(context);
    
    // In apps view, should return the actual app objects
    expect(visibleItems).toEqual(mockApps);
    expect(visibleItems[1]).toEqual(mockApps[1]); // selectedIdx=1 should give us app-two object
    expect(visibleItems[1].name).toBe("app-two");
  });

  it("should warn when no app can be determined from non-apps view", () => {
    const mockApps = [
      { name: "app-one", namespace: "default", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
    ];

    const mockStatusLog = {
      info: mock(),
      warn: mock(),
      error: mock(),
      debug: mock(),
      set: mock(),
      clear: mock(),
    };

    const context = createMockContext({
      state: createMockState({
        navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 }, // In clusters view
        apps: mockApps,
        server: { config: { baseUrl: "https://test.com" }, token: "test-token" },
        selections: {
          scopeClusters: new Set(),
          scopeNamespaces: new Set(),
          scopeProjects: new Set(),
          selectedApps: new Set(), // No apps selected
        },
      }),
      statusLog: mockStatusLog,
    });

    syncCommand.execute(context);
    
    // Should warn that no app is selected to sync
    expect(mockStatusLog.warn).toHaveBeenCalledWith(
      "No app selected to sync.",
      "user-action"
    );
  });

  it("should use selectedApps when available in non-apps view", () => {
    const mockApps = [
      { name: "app-one", namespace: "default", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-two", namespace: "kube-system", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
    ];

    const mockDispatch = mock();

    const context = createMockContext({
      state: createMockState({
        navigation: { view: "namespaces", selectedIdx: 1, lastGPressed: 0, lastEscPressed: 0 }, 
        apps: mockApps,
        server: { config: { baseUrl: "https://test.com" }, token: "test-token" },
        selections: {
          scopeClusters: new Set(),
          scopeNamespaces: new Set(),
          scopeProjects: new Set(),
          selectedApps: new Set(["app-two"]), // app-two is selected
        },
      }),
      dispatch: mockDispatch,
    });

    syncCommand.execute(context);
    
    // Should use the selected app and set up sync confirmation
    expect(mockDispatch).toHaveBeenCalledWith({
      type: "SET_CONFIRM_TARGET",
      payload: "app-two" // Should use the selected app, not the one at selectedIdx
    });
    expect(mockDispatch).toHaveBeenCalledWith({
      type: "SET_MODE",
      payload: "confirm-sync"
    });
  });
});