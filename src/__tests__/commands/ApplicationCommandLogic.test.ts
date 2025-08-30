import { describe, expect, mock, test } from "bun:test";
import {
  createMockApps,
  createMockContext,
  createMockState,
} from "../test-utils";

// Mock the external dependencies BEFORE importing the commands
mock.module("../../components/DiffView", () => ({
  runAppDiffSession: mock(() => Promise.resolve()),
}));

// Import commands AFTER mocks are set up
import {
  DiffCommand,
  ResourcesCommand,
  RollbackCommand,
  SyncCommand,
} from "../../commands/application";

describe("Application Commands (:diff, :sync, :rollback, :resources)", () => {
  describe("DiffCommand (:diff)", () => {
    test("should show diff for specified app argument", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context, "test-app");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Preparing diff for test-app…",
        "diff",
      );
    });

    test("should show diff for currently selected app in apps view", async () => {
      const mockApps = createMockApps();
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: mockApps,
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

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Preparing diff for app1…",
        "diff",
      );
    });

    test("should show diff for single selected app", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          }, // Not in apps view
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["selected-app"]),
          },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context);

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Preparing diff for selected-app…",
        "diff",
      );
    });

    test("should warn when no app is selected", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: [],
          navigation: {
            view: "clusters",
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

    test("should error when not authenticated", async () => {
      const context = createMockContext({
        state: createMockState({ server: null }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context);

      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );
    });

    test("should only allow execution when server is available", () => {
      const diffCommand = new DiffCommand();

      const withServer = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
        }),
      });
      const withoutServer = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(diffCommand.canExecute(withServer)).toBe(true);
      expect(diffCommand.canExecute(withoutServer)).toBe(false);
    });

    test("should have correct properties", () => {
      const diffCommand = new DiffCommand();

      expect(diffCommand.aliases).toEqual([]);
      expect(diffCommand.description).toBe("View diff for application");
    });
  });

  describe("SyncCommand (:sync)", () => {
    test("should sync specified app argument", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context, "test-app");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "test-app",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_PRUNE",
        payload: false,
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_WATCH",
        payload: true,
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "confirm-sync",
      });
    });

    test("should handle multi-app sync confirmation", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["app1", "app2", "app3"]),
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "__MULTI__",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_PRUNE",
        payload: false,
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_WATCH",
        payload: false,
      }); // disabled for multi
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "confirm-sync",
      });
    });

    test("should sync currently selected app in apps view", () => {
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
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app2",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "confirm-sync",
      });
    });

    test("should sync single selected app", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["single-app"]),
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "single-app",
      });
    });

    test("should warn when no app is selected to sync", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: [],
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to sync.",
        "user-action",
      );
    });

    test("should only allow execution when server is available", () => {
      const syncCommand = new SyncCommand();

      const withServer = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
        }),
      });
      const withoutServer = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(syncCommand.canExecute(withServer)).toBe(true);
      expect(syncCommand.canExecute(withoutServer)).toBe(false);
    });

    test("should have correct properties", () => {
      const syncCommand = new SyncCommand();

      expect(syncCommand.aliases).toEqual([]);
      expect(syncCommand.description).toBe("Sync application(s)");
    });
  });

  describe("RollbackCommand (:rollback)", () => {
    test("should rollback specified app argument", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context, "test-app");

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Opening rollback for test-app…",
        "rollback",
      );
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME",
        payload: "test-app",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rollback",
      });
    });

    test("should rollback currently selected app in apps view", async () => {
      const mockApps = createMockApps();
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: mockApps,
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context);

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Opening rollback for app1…",
        "rollback",
      );
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME",
        payload: "app1",
      });
    });

    test("should rollback single selected app", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["selected-app"]),
          },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context);

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Opening rollback for selected-app…",
        "rollback",
      );
    });

    test("should warn when no app is selected to rollback", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: [],
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context);

      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to rollback.",
        "user-action",
      );
    });

    test("should error when not authenticated", async () => {
      const context = createMockContext({
        state: createMockState({ server: null }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context);

      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );
    });

    test("should only allow execution when server is available", () => {
      const rollbackCommand = new RollbackCommand();

      const withServer = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
        }),
      });
      const withoutServer = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(rollbackCommand.canExecute(withServer)).toBe(true);
      expect(rollbackCommand.canExecute(withoutServer)).toBe(false);
    });

    test("should have correct properties", () => {
      const rollbackCommand = new RollbackCommand();

      expect(rollbackCommand.aliases).toEqual([]);
      expect(rollbackCommand.description).toBe("Rollback application");
    });
  });

  describe("ResourcesCommand (:resources)", () => {
    test("should show resources for specified app argument", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context, "test-app");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "test-app",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "resources",
      });
    });

    test("should show resources for currently selected app in apps view", () => {
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
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "app2",
      });
    });

    test("should show resources for single selected app", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["selected-app"]),
          },
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "selected-app",
      });
    });

    test("should warn when no app is selected to view resources", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: [],
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context);

      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to open resources view.",
        "user-action",
      );
    });

    test("should error when not authenticated", () => {
      const context = createMockContext({
        state: createMockState({ server: null }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context);

      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );
    });

    test("should only allow execution when server is available", () => {
      const resourcesCommand = new ResourcesCommand();

      const withServer = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
        }),
      });
      const withoutServer = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(resourcesCommand.canExecute(withServer)).toBe(true);
      expect(resourcesCommand.canExecute(withoutServer)).toBe(false);
    });

    test("should have correct properties", () => {
      const resourcesCommand = new ResourcesCommand();

      expect(resourcesCommand.aliases).toEqual(["resource", "res"]);
      expect(resourcesCommand.description).toBe(
        "View resources for application",
      );
    });
  });

  describe("Comprehensive edge cases for mutation testing", () => {
    test("DiffCommand should handle server authentication correctly", () => {
      const mockApps = createMockApps();

      // Test with no server
      const noServerContext = createMockContext({
        state: createMockState({
          server: null,
          apps: mockApps,
        }),
      });

      const diffCommand = new DiffCommand();
      diffCommand.execute(noServerContext, "test-app");

      expect(noServerContext.statusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );

      // Test with server
      const withServerContext = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: mockApps,
        }),
      });

      const diffCommand2 = new DiffCommand();
      diffCommand2.execute(withServerContext, "test-app");

      expect(withServerContext.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });

    test("All commands should handle server authentication states", () => {
      const commands = [
        new DiffCommand(),
        new SyncCommand(),
        new RollbackCommand(),
        new ResourcesCommand(),
      ];

      for (const command of commands) {
        // With server (canExecute only checks server !== null, not token)
        const authenticatedContext = createMockContext({
          state: createMockState({
            server: { config: { baseUrl: "test" }, token: "token" },
          }),
        });
        expect(command.canExecute(authenticatedContext)).toBe(true);

        // Without server
        const noServerContext = createMockContext({
          state: createMockState({ server: null }),
        });
        expect(command.canExecute(noServerContext)).toBe(false);

        // With server but no token (still passes canExecute)
        const noTokenContext = createMockContext({
          state: createMockState({
            server: { config: { baseUrl: "test" }, token: null },
          }),
        });
        expect(command.canExecute(noTokenContext)).toBe(true);
      }
    });

    test("SyncCommand should handle empty vs non-empty selected apps", () => {
      const mockApps = createMockApps();

      // Test with no selected apps and not in apps view
      const emptySelectionContext = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
          apps: mockApps,
          navigation: {
            view: "projects",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            selectedApps: new Set(),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
      });

      const syncCommandEmpty = new SyncCommand();
      syncCommandEmpty.execute(emptySelectionContext);

      expect(emptySelectionContext.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to sync.",
        "user-action",
      );

      // Test with multiple selected apps
      const multiSelectionContext = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
          apps: mockApps,
          selections: {
            selectedApps: new Set(["app1", "app2"]),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
      });

      const syncCommandMulti = new SyncCommand();
      syncCommandMulti.execute(multiSelectionContext);

      expect(multiSelectionContext.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "__MULTI__",
      });
    });

    test("Commands should validate boolean conditions correctly", () => {
      const mockApps = createMockApps();

      // Test selectedApps.size > 1 condition exactly
      const exactlyTwoAppsContext = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
          apps: mockApps,
          selections: {
            selectedApps: new Set(["app1", "app2"]),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(exactlyTwoAppsContext);

      expect(exactlyTwoAppsContext.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_WATCH",
        payload: false, // disabled for multi
      });

      // Test exactly one selected app
      const exactlyOneAppContext = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
          apps: mockApps,
          selections: {
            selectedApps: new Set(["app1"]),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
      });

      const syncCommand2 = new SyncCommand();
      syncCommand2.execute(exactlyOneAppContext);

      expect(exactlyOneAppContext.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_SYNC_WATCH",
        payload: true, // enabled for single
      });
    });

    test("Commands should handle array bounds correctly", () => {
      const mockApps = createMockApps();

      // Test selectedIdx boundary conditions for DiffCommand
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
          apps: mockApps,
          navigation: {
            view: "apps",
            selectedIdx: 999,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const diffCommand = new DiffCommand();
      diffCommand.execute(context);

      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to diff.",
        "user-action",
      );
    });

    test("Commands should handle Set operations correctly", () => {
      const mockApps = createMockApps();

      // Test Set.values() iteration - RollbackCommand with single selected app
      const singleSelectedApp = new Set(["app1"]);
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "test" }, token: "token" },
          apps: mockApps,
          selections: {
            selectedApps: singleSelectedApp,
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      rollbackCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME",
        payload: "app1",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rollback",
      });
    });

    test("Commands should handle property access patterns", () => {
      // Test different command property configurations (based on actual aliases)
      const commands = [
        { cmd: new DiffCommand(), aliases: [] },
        { cmd: new SyncCommand(), aliases: [] },
        { cmd: new RollbackCommand(), aliases: [] },
        { cmd: new ResourcesCommand(), aliases: ["resource", "res"] },
      ];

      for (const { cmd, aliases } of commands) {
        expect(cmd.aliases).toEqual(aliases);
        expect(typeof cmd.description).toBe("string");
        expect(cmd.description.length).toBeGreaterThan(0);
      }
    });
  });

  describe("App filtering behavior with scoped selections", () => {
    test("should execute diff on correct app when cluster filter is applied", async () => {
      const apps = [
        {
          name: "prod-app",
          sync: "Synced",
          health: "Healthy",
          clusterId: "production",
          clusterLabel: "production",
          namespace: "default",
          appNamespace: "argocd",
          project: "default",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "dev-app",
          sync: "OutOfSync",
          health: "Progressing",
          clusterId: "development", 
          clusterLabel: "development",
          namespace: "default",
          appNamespace: "argocd",
          project: "default",
          lastSyncAt: "2023-12-01T09:30:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(["production"]), // Only production apps visible
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { searchQuery: "", activeFilter: "", command: ":", isVersionOutdated: false, latestVersion: undefined },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context);

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Preparing diff for prod-app…", // Should select filtered app, not dev-app
        "diff",
      );
    });

    test("should execute sync on correct app when namespace filter is applied", () => {
      const apps = [
        {
          name: "system-app",
          sync: "Synced",
          health: "Healthy",
          clusterId: "cluster",
          clusterLabel: "cluster", 
          namespace: "system",
          appNamespace: "argocd",
          project: "default",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "user-app",
          sync: "OutOfSync",
          health: "Progressing",
          clusterId: "cluster",
          clusterLabel: "cluster",
          namespace: "user-apps",
          appNamespace: "argocd", 
          project: "default",
          lastSyncAt: "2023-12-01T09:30:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(["user-apps"]), // Only user-apps namespace visible
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { searchQuery: "", activeFilter: "", command: ":", isVersionOutdated: false, latestVersion: undefined },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "user-app", // Should select filtered app, not system-app
      });
    });

    test("should execute rollback on correct app when project filter is applied", async () => {
      const apps = [
        {
          name: "team-a-app",
          sync: "Synced",
          health: "Healthy",
          clusterId: "cluster",
          clusterLabel: "cluster",
          namespace: "default",
          appNamespace: "argocd",
          project: "team-a",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "team-b-app", 
          sync: "OutOfSync",
          health: "Progressing",
          clusterId: "cluster",
          clusterLabel: "cluster",
          namespace: "default",
          appNamespace: "argocd",
          project: "team-b",
          lastSyncAt: "2023-12-01T09:30:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(["team-b"]), // Only team-b project visible
            selectedApps: new Set(),
          },
          ui: { searchQuery: "", activeFilter: "", command: ":", isVersionOutdated: false, latestVersion: undefined },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME",
        payload: "team-b-app", // Should select filtered app, not team-a-app
      });
    });

    test("should handle combined cluster and namespace filtering", () => {
      const apps = [
        {
          name: "prod-system-app",
          sync: "Synced", 
          health: "Healthy",
          clusterId: "production",
          clusterLabel: "production",
          namespace: "system",
          appNamespace: "argocd",
          project: "infrastructure",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "prod-user-app",
          sync: "OutOfSync",
          health: "Progressing", 
          clusterId: "production",
          clusterLabel: "production",
          namespace: "user-apps",
          appNamespace: "argocd",
          project: "team-a",
          lastSyncAt: "2023-12-01T09:30:00Z",
        },
        {
          name: "dev-user-app",
          sync: "Synced",
          health: "Healthy",
          clusterId: "development",
          clusterLabel: "development", 
          namespace: "user-apps",
          appNamespace: "argocd",
          project: "team-a",
          lastSyncAt: "2023-12-01T08:00:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(["production"]), // Only production cluster
            scopeNamespaces: new Set(["user-apps"]), // Only user-apps namespace
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { searchQuery: "", activeFilter: "", command: ":", isVersionOutdated: false, latestVersion: undefined },
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "prod-user-app", // Should select the app that matches both filters
      });
    });

    test("should handle text filtering during search mode", async () => {
      const apps = [
        {
          name: "frontend-service",
          sync: "Synced",
          health: "Healthy", 
          clusterId: "production",
          clusterLabel: "production",
          namespace: "web",
          appNamespace: "argocd",
          project: "frontend",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "backend-api",
          sync: "OutOfSync",
          health: "Progressing",
          clusterId: "production",
          clusterLabel: "production",
          namespace: "api",
          appNamespace: "argocd",
          project: "backend",
          lastSyncAt: "2023-12-01T09:30:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          mode: "search", // In search mode, uses searchQuery not activeFilter
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { 
            searchQuery: "backend", // Search for backend
            activeFilter: "frontend", // This should be ignored in search mode
            command: ":", 
            isVersionOutdated: false, 
            latestVersion: undefined 
          },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context);

      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Preparing diff for backend-api…", // Should match searchQuery "backend", not activeFilter "frontend"
        "diff",
      );
    });

    test("should handle text filtering in normal mode", () => {
      const apps = [
        {
          name: "web-frontend",
          sync: "Healthy", 
          health: "Healthy",
          clusterId: "production",
          clusterLabel: "production",
          namespace: "web",
          appNamespace: "argocd",
          project: "frontend",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "data-backend",
          sync: "OutOfSync",
          health: "Progressing",
          clusterId: "production", 
          clusterLabel: "production",
          namespace: "data",
          appNamespace: "argocd",
          project: "backend",
          lastSyncAt: "2023-12-01T09:30:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          mode: "normal", // In normal mode, uses activeFilter not searchQuery
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { 
            searchQuery: "backend", // This should be ignored in normal mode
            activeFilter: "web", // Filter for web
            command: ":", 
            isVersionOutdated: false, 
            latestVersion: undefined 
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "web-frontend", // Should match activeFilter "web", not searchQuery "backend"
      });
    });

    test("should handle case-insensitive filtering", async () => {
      const apps = [
        {
          name: "Production-API",
          sync: "Synced",
          health: "Healthy",
          clusterId: "production",
          clusterLabel: "production", 
          namespace: "API-Services",
          appNamespace: "argocd",
          project: "Backend-Team",
          lastSyncAt: "2023-12-01T10:00:00Z",
        }
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(), 
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { 
            searchQuery: "",
            activeFilter: "production-api", // Lowercase filter should match uppercase app name
            command: ":", 
            isVersionOutdated: false, 
            latestVersion: undefined 
          },
        }),
      });

      const rollbackCommand = new RollbackCommand();
      await rollbackCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME", 
        payload: "Production-API", // Should match despite case difference
      });
    });

    test("should work correctly when scoped to non-apps view", () => {
      const apps = createMockApps();

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 }, // In clusters view
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["app1"]), // Has a selected app
          },
          ui: { searchQuery: "", activeFilter: "", command: ":", isVersionOutdated: false, latestVersion: undefined },
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context);

      // Should use selectedApps since not in apps view
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "app1",
      });
    });

    test("should handle empty results from filtering", () => {
      const apps = createMockApps();

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps,
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0, lastEscPressed: 0 },
          selections: {
            scopeClusters: new Set(["nonexistent-cluster"]), // Filter that matches no apps
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
          ui: { searchQuery: "", activeFilter: "", command: ":", isVersionOutdated: false, latestVersion: undefined },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to sync.",
        "user-action",
      );
    });
  });
});
