import { describe, expect, test } from "bun:test";
import { NavigationCommand } from "../../commands/navigation";
import { createMockContext, createMockState } from "../test-utils";

describe("NavigationCommand (:ns, :cls, :proj, :app)", () => {
  describe("NavigationCommand execution", () => {
    test("should switch to namespaces view when :ns command is executed", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);

      nsCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "namespaces" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });

    test("should switch to clusters view when :cls command is executed", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);

      clsCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "clusters" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });

    test("should switch to projects view when :proj command is executed", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const projCommand = new NavigationCommand("projects", "project", [
        "projects",
        "proj",
      ]);

      projCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "projects" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });

    test("should switch to apps view when :app command is executed", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
      });

      const appCommand = new NavigationCommand("apps", "app", ["apps"]);

      appCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "apps" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });
  });

  describe("NavigationCommand with arguments", () => {
    test("should set scope clusters when switching to clusters view with argument", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);

      clsCommand.execute(context, "production");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "clusters" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(["production"]),
      });
    });

    test("should set scope namespaces when switching to namespaces view with argument", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);

      nsCommand.execute(context, "default");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "namespaces" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(["default"]),
      });
    });

    test("should set scope projects when switching to projects view with argument", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const projCommand = new NavigationCommand("projects", "project", [
        "projects",
        "proj",
      ]);

      projCommand.execute(context, "team-a");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "projects" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(["team-a"]),
      });
    });

    test("should set selected apps when switching to apps view with argument", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const appCommand = new NavigationCommand("apps", "app", ["apps"]);

      appCommand.execute(context, "my-app");

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
        payload: { view: "apps" },
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(["my-app"]),
      });
    });
  });

  describe("NavigationCommand without arguments", () => {
    test("should clear selections when switching to view without argument", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          selections: {
            scopeClusters: new Set(["existing-cluster"]),
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

      clsCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
    });
  });

  describe("NavigationCommand canExecute", () => {
    test("should allow execution in normal mode", () => {
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);

      expect(nsCommand.canExecute(context)).toBe(true);
    });

    test("should allow execution in command mode", () => {
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);

      expect(nsCommand.canExecute(context)).toBe(true);
    });

    test("should not allow execution in search mode", () => {
      const context = createMockContext({
        state: createMockState({ mode: "search" }),
      });

      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);

      expect(nsCommand.canExecute(context)).toBe(false);
    });
  });

  describe("NavigationCommand aliases", () => {
    test("should have correct aliases for namespace command", () => {
      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);

      expect(nsCommand.aliases).toEqual(["namespaces", "ns"]);
    });

    test("should have correct aliases for cluster command", () => {
      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);

      expect(clsCommand.aliases).toEqual(["clusters", "cls"]);
    });

    test("should have correct aliases for project command", () => {
      const projCommand = new NavigationCommand("projects", "project", [
        "projects",
        "proj",
      ]);

      expect(projCommand.aliases).toEqual(["projects", "proj"]);
    });

    test("should have correct aliases for app command", () => {
      const appCommand = new NavigationCommand("apps", "app", ["apps"]);

      expect(appCommand.aliases).toEqual(["apps"]);
    });
  });

  describe("NavigationCommand descriptions", () => {
    test("should have correct description for each view", () => {
      const nsCommand = new NavigationCommand("namespaces", "namespace", [
        "namespaces",
        "ns",
      ]);
      const clsCommand = new NavigationCommand("clusters", "cluster", [
        "clusters",
        "cls",
      ]);
      const projCommand = new NavigationCommand("projects", "project", [
        "projects",
        "proj",
      ]);
      const appCommand = new NavigationCommand("apps", "app", ["apps"]);

      expect(nsCommand.description).toBe("Switch to namespaces view");
      expect(clsCommand.description).toBe("Switch to clusters view");
      expect(projCommand.description).toBe("Switch to projects view");
      expect(appCommand.description).toBe("Switch to apps view");
    });
  });
});
