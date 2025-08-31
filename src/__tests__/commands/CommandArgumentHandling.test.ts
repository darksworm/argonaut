import { describe, expect, test } from "bun:test";
import { ResourcesCommand, SyncCommand } from "../../commands/application";
import { NavigationCommand } from "../../commands/navigation";
import {
  createMockApps,
  createMockContext,
  createMockState,
} from "../test-utils";

describe("Command Argument Handling", () => {
  describe("Navigation commands with arguments", () => {
    test(":ns with namespace argument should filter to specific namespace", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
        }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);
      nsCommand.execute(context, "kube-system");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(["kube-system"]),
      });
    });

    test(":cls with cluster argument should filter to specific cluster", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);
      clsCommand.execute(context, "production");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(["production"]),
      });
    });

    test(":proj with project argument should filter to specific project", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const projCommand = new NavigationCommand("projects", "project", [
        "projects",
        "proj",
      ]);
      projCommand.execute(context, "team-frontend");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(["team-frontend"]),
      });
    });

    test(":app with app argument should navigate and focus on that app", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          apps: createMockApps(),
        }),
      });

      const appCommand = new NavigationCommand("apps", "app", ["apps"]);
      appCommand.execute(context, "app2");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "apps" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 1,
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(["app2"]),
      });
    });
  });

  describe("Application commands with arguments", () => {
    test(":sync with app argument should prepare sync for specific app", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const syncCommand = new SyncCommand();
      syncCommand.execute(context, "specific-app");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "specific-app",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "confirm-sync",
      });
    });

    test(":resources with app argument should show resources for specific app", () => {
      const context = createMockContext({
        state: createMockState({
          server: { config: { baseUrl: "https://test.com" }, token: "token" },
        }),
      });

      const resourcesCommand = new ResourcesCommand();
      resourcesCommand.execute(context, "specific-app");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "specific-app",
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "resources",
      });
    });
  });

  describe("Argument edge cases", () => {
    test("navigation commands with empty string argument should clear selections (empty string is falsy)", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          selections: {
            scopeClusters: new Set(["existing"]),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
        }),
      });

      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);
      clsCommand.execute(context, "");

      // Empty string is falsy in JavaScript, so it clears selections instead of setting them
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
    });

    test("navigation commands with whitespace-only argument should be treated as argument", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);
      nsCommand.execute(context, "   ");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(["   "]),
      });
    });

    test("application commands with undefined argument should fall back to context-based selection", () => {
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
      syncCommand.execute(context, undefined);

      // Should use the app at selectedIdx (1) which is "app2"
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "app2",
      });
    });

    test("commands should handle special characters in arguments", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const appCommand = new NavigationCommand("apps", "app", ["apps"]);
      appCommand.execute(context, "my-app.with-special_chars123");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(["my-app.with-special_chars123"]),
      });
    });
  });
});
