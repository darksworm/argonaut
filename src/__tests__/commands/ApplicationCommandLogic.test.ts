import { describe, expect, mock, test } from "bun:test";
import {
  DiffCommand,
  ResourcesCommand,
  RollbackCommand,
  SyncCommand,
} from "../../commands/application";
import {
  createMockApps,
  createMockContext,
  createMockState,
} from "../test-utils";

// Mock the external dependencies
mock.module("../../components/DiffView", () => ({
  runAppDiffSession: mock(() => Promise.resolve()),
}));

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
});
