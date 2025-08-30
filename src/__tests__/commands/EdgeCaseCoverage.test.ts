import { describe, expect, mock, test } from "bun:test";
import { DiffCommand, SyncCommand } from "../../commands/application";
import {
  createMockApps,
  createMockContext,
  createMockState,
} from "../test-utils";

// Mock the external dependencies
mock.module("../../components/DiffView", () => ({
  runAppDiffSession: mock(() => Promise.resolve()),
}));

describe("Edge Case Coverage for Mutation Testing", () => {
  describe("SyncCommand edge cases", () => {
    test("should handle filtering with empty app properties", () => {
      const appsWithEmptyProps = [
        {
          name: "app1",
          sync: "",
          health: "",
          clusterId: "cluster1",
          clusterLabel: "",
          namespace: "",
          appNamespace: "argocd",
          project: "",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
        {
          name: "app2",
          sync: "Synced",
          health: "Healthy",
          clusterId: "cluster2",
          clusterLabel: "production",
          namespace: "default",
          appNamespace: "argocd",
          project: "team-a",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: appsWithEmptyProps,
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          ui: {
            searchQuery: "",
            activeFilter: "healthy",
            command: ":",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Filter "healthy" will match app2 which has "Healthy" health, so app2 becomes selectedIdx 0
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app2",
      });
    });

    test("should handle filtering with search mode vs active filter", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
          mode: "search",
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          ui: {
            searchQuery: "app1",
            activeFilter: "different-filter",
            command: ":",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should prioritize search mode over active filter
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app1",
      });
    });

    test("should handle apps with null/undefined properties", () => {
      const appsWithNullProps = [
        {
          name: "app1",
          sync: null as any,
          health: undefined as any,
          clusterId: "cluster1",
          clusterLabel: null as any,
          namespace: undefined as any,
          appNamespace: "argocd",
          project: null as any,
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
      ];

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: appsWithNullProps,
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app1",
      });
    });

    test("should handle case-sensitive filtering", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          ui: {
            searchQuery: "",
            activeFilter: "APP1", // uppercase filter
            command: ":",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should match case-insensitively
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app1",
      });
    });

    test("should handle non-apps view filtering", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          ui: {
            searchQuery: "",
            activeFilter: "in-cluster",
            command: ":",
            isVersionOutdated: false,
            latestVersion: undefined,
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
  });

  describe("DiffCommand edge cases", () => {
    test("should handle diff session with empty argument", async () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context, "");

      // Empty string is falsy, so should warn about no app selected
      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to diff.",
        "user-action",
      );
    });

    test("should handle diff session error with no message", async () => {
      const error = {}; // Error with no message property
      mock.module("../../components/DiffView", () => ({
        runAppDiffSession: mock(() => Promise.reject(error)),
      }));

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context, "test-app");

      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Diff failed: [object Object]",
        "diff",
      );
    });

    test("should handle diff session error with null message", async () => {
      const error = { message: null };
      mock.module("../../components/DiffView", () => ({
        runAppDiffSession: mock(() => Promise.reject(error)),
      }));

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context, "test-app");

      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Diff failed: [object Object]",
        "diff",
      );
    });

    test("should handle process.stdin errors gracefully", async () => {
      const originalStdin = process.stdin;
      const mockStdin = {
        setRawMode: mock(() => {
          throw new Error("setRawMode failed");
        }),
        resume: mock(() => {
          throw new Error("resume failed");
        }),
      };
      (process as any).stdin = mockStdin;

      mock.module("../../components/DiffView", () => ({
        runAppDiffSession: mock(() => Promise.reject(new Error("test error"))),
      }));

      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const diffCommand = new DiffCommand();
      await diffCommand.execute(context, "test-app");

      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Diff failed: test error",
        "diff",
      );

      // Restore original stdin
      process.stdin = originalStdin;
    });
  });

  describe("Filtering edge cases", () => {
    test("should handle empty filter gracefully", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          ui: {
            searchQuery: "",
            activeFilter: "", // empty filter
            command: ":",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should work with empty filter (no filtering applied)
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app1",
      });
    });

    test("should handle boolean false filter correctly", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
          apps: createMockApps(),
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          ui: {
            searchQuery: "",
            activeFilter: "false", // string "false" should still filter
            command: ":",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context);

      // Should not match any apps with filter "false"
      expect(context.statusLog.warn).toHaveBeenCalledWith(
        "No app selected to sync.",
        "user-action",
      );
    });
  });
});
