import { describe, expect, test } from "bun:test";
import { DiffCommand, SyncCommand } from "../../commands/application";
import {
  ClearAllCommand,
  ClearCommand,
  NavigationCommand,
  UpCommand,
} from "../../commands/navigation";
import {
  createMockApps,
  createMockContext,
  createMockState,
} from "../test-utils";

describe("Command Execution with Different App States", () => {
  describe("Commands with different view contexts", () => {
    test("sync command should work correctly from clusters view with selected apps", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(["production"]),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["important-app"]),
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should use the selected app even when not in apps view
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "important-app",
      });
    });

    test("sync command should use cursor app when in apps view", () => {
      const mockApps = createMockApps();
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: mockApps,
          navigation: {
            view: "apps",
            selectedIdx: 1,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(), // No explicitly selected apps
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should use the app at cursor position (index 1 = app2)
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app2",
      });
    });

    test("navigation commands should clear appropriate selections based on current state", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          selections: {
            scopeClusters: new Set(["prod", "staging"]),
            scopeNamespaces: new Set(["default", "kube-system"]),
            scopeProjects: new Set(["team-a", "team-b"]),
            selectedApps: new Set(["app1", "app2"]),
          },
        }),
      });

      // Switching to clusters should clear cluster selections
      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);
      clsCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
    });
  });

  describe("Commands with complex selection states", () => {
    test("clear command should clear correct selection type based on current view", () => {
      // Test clearing in each view type
      const views = [
        { view: "clusters", expectedAction: "SET_SCOPE_CLUSTERS" },
        { view: "namespaces", expectedAction: "SET_SCOPE_NAMESPACES" },
        { view: "projects", expectedAction: "SET_SCOPE_PROJECTS" },
        { view: "apps", expectedAction: "SET_SELECTED_APPS" },
      ] as const;

      for (const { view, expectedAction } of views) {
        const context = createMockContext({
          state: createMockState({
            mode: "command",
            navigation: {
              view,
              selectedIdx: 0,
              lastGPressed: 0,
              lastEscPressed: 0,
            },
            selections: {
              scopeClusters: new Set(["cluster1"]),
              scopeNamespaces: new Set(["ns1"]),
              scopeProjects: new Set(["proj1"]),
              selectedApps: new Set(["app1"]),
            },
          }),
        });

        const clearCommand = new ClearCommand();
        clearCommand.execute(context);

        expect(context.dispatch).toHaveBeenCalledWith({
          type: expectedAction,
          payload: new Set(),
        });
      }
    });

    test("up command should navigate correctly through hierarchy", () => {
      const hierarchyTests = [
        {
          fromView: "apps",
          expectedActions: [
            { type: "SET_SELECTED_APPS", payload: new Set() },
            { type: "SET_SCOPE_PROJECTS", payload: new Set() },
            { type: "SET_VIEW", payload: "projects" },
          ],
        },
        {
          fromView: "projects",
          expectedActions: [
            { type: "SET_SCOPE_NAMESPACES", payload: new Set() },
            { type: "SET_VIEW", payload: "namespaces" },
          ],
        },
        {
          fromView: "namespaces",
          expectedActions: [
            { type: "SET_SCOPE_CLUSTERS", payload: new Set() },
            { type: "SET_VIEW", payload: "clusters" },
          ],
        },
        {
          fromView: "clusters",
          expectedActions: [
            { type: "SET_SCOPE_CLUSTERS", payload: new Set() },
            // No view change expected
          ],
        },
      ] as const;

      for (const { fromView, expectedActions } of hierarchyTests) {
        const context = createMockContext({
          state: createMockState({
            mode: "command",
            navigation: {
              view: fromView,
              selectedIdx: 2,
              lastGPressed: 0,
              lastEscPressed: 0,
            },
          }),
        });

        const upCommand = new UpCommand();
        upCommand.execute(context);

        // Should always reset selectedIdx and clear filters
        expect(context.dispatch).toHaveBeenCalledWith({
          type: "SET_SELECTED_IDX",
          payload: 0,
        });
        expect(context.dispatch).toHaveBeenCalledWith({
          type: "CLEAR_FILTERS",
        });

        // Check specific actions for this view
        for (const expectedAction of expectedActions) {
          expect(context.dispatch).toHaveBeenCalledWith(expectedAction);
        }
      }
    });
  });

  describe("Commands with different application states", () => {
    test("diff command should handle empty apps list gracefully", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: [], // Empty apps array
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context);

      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to diff.",
        "user-action",
      );
    });

    test("sync command should handle multiple selected apps correctly", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["app1", "app2", "app3", "app4"]), // Multiple apps
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should trigger multi-sync confirmation
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "__MULTI__",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_WATCH",
        payload: false, // Should be disabled for multi-sync
      });
    });

    test("commands should handle server authentication states correctly", () => {
      const authenticatedContext = createMockContext({
        state: createMockState({
          server: {
            config: { baseUrl: "https://test.com" },
            token: "valid-token",
          },
        }),
      });

      const unauthenticatedContext = createMockContext({
        state: createMockState({
          server: null,
        }),
      });

      const syncCommand = new SyncCommand();
      const diffCommand = new DiffCommand();

      // Should be able to execute when authenticated
      expect(syncCommand.canExecute(authenticatedContext)).toBe(true);
      expect(diffCommand.canExecute(authenticatedContext)).toBe(true);

      // Should not be able to execute when not authenticated
      expect(syncCommand.canExecute(unauthenticatedContext)).toBe(false);
      expect(diffCommand.canExecute(unauthenticatedContext)).toBe(false);
    });
  });

  describe("Commands with filtering states", () => {
    test("navigation commands should work correctly with existing filters", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          ui: {
            searchQuery: "search-term",
            activeFilter: "filter-term",
            command: "",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
          selections: {
            scopeClusters: new Set(["existing-cluster"]),
            scopeNamespaces: new Set(["existing-namespace"]),
            scopeProjects: new Set(["existing-project"]),
            selectedApps: new Set(["existing-app"]),
          },
        }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);
      nsCommand.execute(context, "new-namespace");

      // Should switch view and set new namespace selection
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "namespaces" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(["new-namespace"]),
      });
    });

    test("clearAll command should clear all selections and filters regardless of current state", () => {
      const context = createMockContext({
        state: createMockState({
          ui: {
            searchQuery: "active-search",
            activeFilter: "active-filter",
            command: "some-command",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
          selections: {
            scopeClusters: new Set(["cluster1", "cluster2"]),
            scopeNamespaces: new Set(["ns1", "ns2"]),
            scopeProjects: new Set(["proj1", "proj2"]),
            selectedApps: new Set(["app1", "app2", "app3"]),
          },
        }),
      });

      const clearAllCommand = new ClearAllCommand();
      clearAllCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "CLEAR_ALL_SELECTIONS",
      });
      expect(context.dispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      expect(context.statusLog.info).toHaveBeenCalledWith(
        "All filtering cleared.",
        "user-action",
      );
    });
  });

  describe("Commands with different mode states", () => {
    test("navigation commands should work in both normal and command modes", () => {
      const modes = ["normal", "command"] as const;

      for (const mode of modes) {
        const context = createMockContext({
          state: createMockState({ mode }),
        });

        const nsCommand = new NavigationCommand("namespaces", "namespace", [
          "namespaces",
          "ns",
        ]);

        expect(nsCommand.canExecute(context)).toBe(true);

        nsCommand.execute(context);

        // Should always switch to normal mode after execution
        expect(context.dispatch).toHaveBeenCalledWith({
          type: "SET_MODE",
          payload: "normal",
        });
      }
    });

    test("navigation commands should not execute in restricted modes", () => {
      const restrictedModes = [
        "search",
        "confirm-sync",
        "rollback",
        "resources",
      ] as const;

      for (const mode of restrictedModes) {
        const context = createMockContext({
          state: createMockState({ mode }),
        });

        const nsCommand = new NavigationCommand("namespaces", "namespace", [
          "namespaces",
          "ns",
        ]);

        expect(nsCommand.canExecute(context)).toBe(false);
      }
    });
  });
});
