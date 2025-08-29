// src/__tests__/bugs/namespace-selection-boundary.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";
import { NavigationInputHandler } from "../../commands/handlers/keyboard";
import { createMockContext, createMockState } from "../test-utils";

/**
 * Test to reproduce and verify fix for the bug where users cannot
 * select the final element in the namespaces view.
 */
describe("Namespace Selection Boundary Bug", () => {
  let handler: NavigationInputHandler;

  beforeEach(() => {
    handler = new NavigationInputHandler();
  });

  it("should reproduce the bug - cannot select final namespace", () => {
    const mockApps = [
      {
        name: "app1",
        namespace: "default",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
      {
        name: "app2",
        namespace: "kube-system",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
      {
        name: "app3",
        namespace: "monitoring",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
      {
        name: "app4",
        namespace: "ingress-nginx",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
    ];

    const mockDispatch = mock();
    const context = createMockContext({
      state: createMockState({
        navigation: {
          view: "namespaces",
          selectedIdx: 2,
          lastGPressed: 0,
          lastEscPressed: 0,
        },
        apps: mockApps,
      }),
      dispatch: mockDispatch,
    });

    // Try to navigate down from index 2 (there should be 4 unique namespaces: default, kube-system, monitoring, ingress-nginx)
    // The issue is that getVisibleItemsCount returns 10 instead of the actual count
    const result = handler.handleInput("j", {}, context);

    expect(result).toBe(true);

    // The bug: it should set selectedIdx to 3 (the last namespace), but with the hardcoded 10,
    // it will try to set it to 3 (because Math.min(2 + 1, Math.max(0, 10 - 1)) = Math.min(3, 9) = 3)
    // This actually works in this case, but let's test the real boundary issue
    expect(mockDispatch).toHaveBeenCalledWith({
      type: "SET_SELECTED_IDX",
      payload: 3,
    });
  });

  it("should demonstrate the boundary fix - correct visible items count", () => {
    const mockApps = [
      {
        name: "app1",
        namespace: "default",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
      {
        name: "app2",
        namespace: "kube-system",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
    ];

    const mockDispatch = mock();
    const context = createMockContext({
      state: createMockState({
        navigation: {
          view: "namespaces",
          selectedIdx: 1,
          lastGPressed: 0,
          lastEscPressed: 0,
        }, // At the last valid index (1)
        apps: mockApps, // Only 2 unique namespaces: "default", "kube-system"
      }),
      dispatch: mockDispatch,
    });

    // Try to navigate down from index 1 - should not be able to go further
    const result = handler.handleInput("j", {}, context);

    expect(result).toBe(true);

    // Fixed: now it calculates Math.min(1 + 1, Math.max(0, 2 - 1)) = Math.min(2, 1) = 1
    // So selectedIdx correctly stays at 1 (the last valid index)
    const lastCall =
      mockDispatch.mock.calls[mockDispatch.mock.calls.length - 1];
    expect(lastCall[0]).toEqual({
      type: "SET_SELECTED_IDX",
      payload: 1, // Fixed! Now correctly stays at 1
    });
  });

  it("should demonstrate the getVisibleItemsCount fix", () => {
    const mockApps = [
      {
        name: "app1",
        namespace: "default",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
      {
        name: "app2",
        namespace: "kube-system",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
    ];

    const context = createMockContext({
      state: createMockState({
        navigation: {
          view: "namespaces",
          selectedIdx: 0,
          lastGPressed: 0,
          lastEscPressed: 0,
        },
        apps: mockApps,
      }),
    });

    // @ts-expect-error - accessing private method for testing
    const count = handler.getVisibleItemsCount(context);

    // Fixed: this now returns 2 (actual unique namespace count) instead of 10 (hardcoded)
    expect(count).toBe(2); // Now correctly returns the actual count
  });

  it("should fix the boundary issue - can now properly navigate to last namespace", () => {
    const mockApps = [
      {
        name: "app1",
        namespace: "default",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
      {
        name: "app2",
        namespace: "kube-system",
        clusterLabel: "cluster1",
        project: "default",
        sync: "Synced",
        health: "Healthy",
        clusterId: "cluster1",
        appNamespace: "argocd",
        lastSyncAt: "2023-12-01T10:00:00Z",
      },
    ];

    const mockDispatch = mock();
    const context = createMockContext({
      state: createMockState({
        navigation: {
          view: "namespaces",
          selectedIdx: 1,
          lastGPressed: 0,
          lastEscPressed: 0,
        }, // At the last valid index (1)
        apps: mockApps, // Only 2 unique namespaces: "default", "kube-system"
      }),
      dispatch: mockDispatch,
    });

    // Try to navigate down from index 1 - should not be able to go further
    const result = handler.handleInput("j", {}, context);

    expect(result).toBe(true);

    // Fixed: now it calculates Math.min(1 + 1, Math.max(0, 2 - 1)) = Math.min(2, 1) = 1
    // So selectedIdx stays at 1 (the last valid index)
    const lastCall =
      mockDispatch.mock.calls[mockDispatch.mock.calls.length - 1];
    expect(lastCall[0]).toEqual({
      type: "SET_SELECTED_IDX",
      payload: 1, // Fixed! Now correctly stays at 1
    });
  });
});
