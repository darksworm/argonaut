import { describe, expect, test } from "bun:test";
import {
  ClearAllCommand,
  ClearCommand,
  UpCommand,
} from "../../commands/navigation";
import { createMockContext, createMockState } from "../test-utils";

describe("Clear Commands (:clear, :all, :up)", () => {
  describe("ClearCommand (:clear)", () => {
    test("should clear cluster selections in clusters view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(["production", "staging"]),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
        }),
      });

      const clearCommand = new ClearCommand();
      clearCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    test("should clear namespace selections in namespaces view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "namespaces",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(["default", "kube-system"]),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
        }),
      });

      const clearCommand = new ClearCommand();
      clearCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(),
      });
      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    test("should clear project selections in projects view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "projects",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(["team-a", "team-b"]),
            selectedApps: new Set(),
          },
        }),
      });

      const clearCommand = new ClearCommand();
      clearCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(),
      });
      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    test("should clear app selections in apps view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(["app1", "app2"]),
          },
        }),
      });

      const clearCommand = new ClearCommand();
      clearCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
      });
      expect(context.statusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    test("should allow execution in normal and command modes", () => {
      const clearCommand = new ClearCommand();

      const normalContext = createMockContext({
        state: createMockState({ mode: "normal" }),
      });
      const commandContext = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      expect(clearCommand.canExecute(normalContext)).toBe(true);
      expect(clearCommand.canExecute(commandContext)).toBe(true);
    });

    test("should not allow execution in other modes", () => {
      const clearCommand = new ClearCommand();

      const searchContext = createMockContext({
        state: createMockState({ mode: "search" }),
      });

      expect(clearCommand.canExecute(searchContext)).toBe(false);
    });
  });

  describe("ClearAllCommand (:all)", () => {
    test("should clear all selections and filters", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          selections: {
            scopeClusters: new Set(["production"]),
            scopeNamespaces: new Set(["default"]),
            scopeProjects: new Set(["team-a"]),
            selectedApps: new Set(["app1"]),
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

    test("should have correct properties", () => {
      const clearAllCommand = new ClearAllCommand();

      expect(clearAllCommand.aliases).toEqual([]);
      expect(clearAllCommand.description).toBe(
        "Clear all selections and filters",
      );
    });
  });

  describe("UpCommand (:up)", () => {
    test("should navigate from apps to projects view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "apps",
            selectedIdx: 5,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(["production"]),
            scopeNamespaces: new Set(["default"]),
            scopeProjects: new Set(["team-a"]),
            selectedApps: new Set(["app1"]),
          },
        }),
      });

      const upCommand = new UpCommand();
      upCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0,
      });
      expect(context.dispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(),
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "projects",
      });
    });

    test("should navigate from projects to namespaces view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "projects",
            selectedIdx: 3,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(["production"]),
            scopeNamespaces: new Set(["default"]),
            scopeProjects: new Set(["team-a"]),
            selectedApps: new Set(),
          },
        }),
      });

      const upCommand = new UpCommand();
      upCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0,
      });
      expect(context.dispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(),
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "namespaces",
      });
    });

    test("should navigate from namespaces to clusters view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "namespaces",
            selectedIdx: 2,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(["production"]),
            scopeNamespaces: new Set(["default"]),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
        }),
      });

      const upCommand = new UpCommand();
      upCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0,
      });
      expect(context.dispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "clusters",
      });
    });

    test("should clear cluster selections when already in clusters view", () => {
      const context = createMockContext({
        state: createMockState({
          mode: "command",
          navigation: {
            view: "clusters",
            selectedIdx: 1,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
          selections: {
            scopeClusters: new Set(["production", "staging"]),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
            selectedApps: new Set(),
          },
        }),
      });

      const upCommand = new UpCommand();
      upCommand.execute(context);

      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0,
      });
      expect(context.dispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      expect(context.dispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
      // Should not change view when already at clusters
      expect(context.dispatch).not.toHaveBeenCalledWith(
        expect.objectContaining({ type: "SET_VIEW" }),
      );
    });

    test("should allow execution in normal and command modes", () => {
      const upCommand = new UpCommand();

      const normalContext = createMockContext({
        state: createMockState({ mode: "normal" }),
      });
      const commandContext = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      expect(upCommand.canExecute(normalContext)).toBe(true);
      expect(upCommand.canExecute(commandContext)).toBe(true);
    });

    test("should not allow execution in other modes", () => {
      const upCommand = new UpCommand();

      const searchContext = createMockContext({
        state: createMockState({ mode: "search" }),
      });

      expect(upCommand.canExecute(searchContext)).toBe(false);
    });

    test("should have correct properties", () => {
      const upCommand = new UpCommand();

      expect(upCommand.aliases).toEqual(["up"]);
      expect(upCommand.description).toBe(
        "Go up one level in navigation hierarchy",
      );
    });
  });
});
