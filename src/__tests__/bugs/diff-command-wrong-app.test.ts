// src/__tests__/bugs/diff-command-wrong-app.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";
import { DiffCommand } from "../../commands/application";
import { createMockContext, createMockState } from "../test-utils";

/**
 * Test to reproduce and verify fix for the bug where the diff command
 * shows the wrong app when executed from non-apps views.
 */
describe("Diff Command Wrong App Bug", () => {
  let diffCommand: DiffCommand;

  beforeEach(() => {
    diffCommand = new DiffCommand();
  });

  it("should fix the getVisibleItems to return correct data for namespaces view", () => {
    const mockApps = [
      { name: "app-in-default", namespace: "default", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-in-kube-system", namespace: "kube-system", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-in-monitoring", namespace: "monitoring", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
    ];

    const context = createMockContext({
      state: createMockState({
        navigation: { view: "namespaces", selectedIdx: 1, lastGPressed: 0, lastEscPressed: 0 }, // Selected index 1 
        apps: mockApps,
        server: { config: { baseUrl: "https://test.com" }, token: "test-token" },
      }),
    });

    // @ts-expect-error - accessing private method for testing
    const visibleItems = diffCommand.getVisibleItems(context);
    
    // Fixed: in namespaces view, getVisibleItems now returns the actual namespaces
    expect(visibleItems).toEqual(["default", "kube-system", "monitoring"]); // Fixed! Now returns namespaces
    expect(visibleItems.length).toBe(3); // Correct count of unique namespaces
    expect(visibleItems[1]).toBe("kube-system"); // selectedIdx=1 should refer to kube-system namespace
  });

  it("should warn when no app can be determined from non-apps view", async () => {
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

    await diffCommand.execute(context);
    
    // Should warn that no app is selected to diff
    expect(mockStatusLog.warn).toHaveBeenCalledWith(
      "No app selected to diff.",
      "user-action"
    );
  });

  it("should handle apps view correctly and return app objects", () => {
    const mockApps = [
      { name: "app-one", namespace: "default", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
      { name: "app-two", namespace: "kube-system", clusterLabel: "cluster1", project: "default", sync: "Synced", health: "Healthy", clusterId: "cluster1", appNamespace: "argocd", lastSyncAt: "2023-12-01T10:00:00Z" },
    ];

    const context = createMockContext({
      state: createMockState({
        navigation: { view: "apps", selectedIdx: 1, lastGPressed: 0, lastEscPressed: 0 }, // In apps view
        apps: mockApps,
        server: { config: { baseUrl: "https://test.com" }, token: "test-token" },
      }),
    });

    // @ts-expect-error - accessing private method for testing
    const visibleItems = diffCommand.getVisibleItems(context);
    
    // In apps view, should return the actual app objects
    expect(visibleItems).toEqual(mockApps);
    expect(visibleItems[1]).toEqual(mockApps[1]); // selectedIdx=1 should give us app-two object
    expect(visibleItems[1].name).toBe("app-two");
  });
});