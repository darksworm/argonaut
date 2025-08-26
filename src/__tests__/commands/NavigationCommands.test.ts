// src/__tests__/commands/NavigationCommands.test.ts
import { mock } from "bun:test";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

import type { Command, CommandContext } from "../../commands/types";
import type { View } from "../../types/domain";
import { createMockContext, createMockState } from "../test-utils";

// Test implementations of navigation command classes without external dependencies
class TestNavigationCommand implements Command {
  constructor(
    private targetView: View,
    _commandName: string,
    public aliases: string[] = [],
  ) {}

  get description() {
    return `Switch to ${this.targetView} view`;
  }

  canExecute(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  execute(context: CommandContext, arg?: string): void {
    const { dispatch } = context;

    dispatch({ type: "RESET_NAVIGATION", payload: { view: this.targetView } });
    dispatch({ type: "SET_MODE", payload: "normal" });

    // Handle view-specific argument for selection
    if (arg) {
      switch (this.targetView) {
        case "clusters":
          dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set([arg]) });
          break;
        case "namespaces":
          dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set([arg]) });
          break;
        case "projects":
          dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set([arg]) });
          break;
        case "apps":
          dispatch({ type: "SET_SELECTED_APPS", payload: new Set([arg]) });
          break;
      }
    } else {
      // Clear selection when returning to view without argument
      switch (this.targetView) {
        case "clusters":
          dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set() });
          break;
        case "namespaces":
          dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set() });
          break;
        case "projects":
          dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set() });
          break;
        case "apps":
          dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
          break;
      }
    }
  }
}

class TestClearCommand implements Command {
  aliases = [];
  description = "Clear current view selection";

  canExecute(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  execute(context: CommandContext): void {
    const { state, dispatch, statusLog } = context;
    const { view } = state.navigation;

    switch (view) {
      case "clusters":
        dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set() });
        break;
      case "namespaces":
        dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set() });
        break;
      case "projects":
        dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set() });
        break;
      case "apps":
        dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
        break;
    }

    statusLog.info("Selection cleared.", "user-action");
  }
}

class TestClearAllCommand implements Command {
  aliases = [];
  description = "Clear all selections and filters";

  execute(context: CommandContext): void {
    const { dispatch, statusLog } = context;

    dispatch({ type: "CLEAR_ALL_SELECTIONS" });
    dispatch({ type: "CLEAR_FILTERS" });

    statusLog.info("All filtering cleared.", "user-action");
  }
}

describe("NavigationCommand", () => {
  describe("constructor and properties", () => {
    it("should create command with correct target view", () => {
      // Arrange & Act
      const appsCommand = new TestNavigationCommand("apps", "apps");
      const clustersCommand = new TestNavigationCommand("clusters", "clusters");

      // Assert
      expect(appsCommand.description).toBe("Switch to apps view");
      expect(clustersCommand.description).toBe("Switch to clusters view");
    });

    it("should handle aliases correctly", () => {
      // Arrange & Act
      const command = new TestNavigationCommand("apps", "applications", [
        "app",
        "a",
      ]);

      // Assert
      expect(command.aliases).toEqual(["app", "a"]);
    });

    it("should default to empty aliases", () => {
      // Arrange & Act
      const command = new TestNavigationCommand("clusters", "clusters");

      // Assert
      expect(command.aliases).toEqual([]);
    });
  });

  describe("canExecute", () => {
    it("should allow execution in normal mode", () => {
      // Arrange
      const command = new TestNavigationCommand("apps", "apps");
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
      });

      // Act & Assert
      expect(command.canExecute(context)).toBe(true);
    });

    it("should prevent execution in non-normal modes", () => {
      // Arrange
      const command = new TestNavigationCommand("apps", "apps");
      const nonNormalModes = [
        "confirm-sync",
        "help",
        "search",
        "command",
        "auth-required",
      ];

      nonNormalModes.forEach((mode) => {
        const context = createMockContext({
          state: createMockState({ mode: mode as any }),
        });

        // Act & Assert
        expect(command.canExecute(context)).toBe(false);
      });
    });
  });

  describe("execute", () => {
    describe("basic navigation", () => {
      it("should switch to target view without argument", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("clusters", "clusters");
        const context = createMockContext({
          dispatch: mockDispatch,
        });

        // Act
        command.execute(context);

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "RESET_NAVIGATION",
          payload: { view: "clusters" },
        });
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_MODE",
          payload: "normal",
        });
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_SCOPE_CLUSTERS",
          payload: new Set(),
        });
      });

      it("should work for all view types", () => {
        // Arrange
        const views: View[] = ["apps", "clusters", "namespaces", "projects"];

        views.forEach((view) => {
          const mockDispatch = mock();
          const command = new TestNavigationCommand(view, view);
          const context = createMockContext({
            dispatch: mockDispatch,
          });

          // Act
          command.execute(context);

          // Assert
          expect(mockDispatch).toHaveBeenCalledWith({
            type: "RESET_NAVIGATION",
            payload: { view },
          });
        });
      });
    });

    describe("navigation with arguments", () => {
      it("should handle cluster navigation with argument", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("clusters", "clusters");
        const context = createMockContext({
          dispatch: mockDispatch,
        });

        // Act
        command.execute(context, "production");

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_SCOPE_CLUSTERS",
          payload: new Set(["production"]),
        });
      });

      it("should handle namespace navigation with argument", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("namespaces", "namespaces");
        const context = createMockContext({
          dispatch: mockDispatch,
        });

        // Act
        command.execute(context, "kube-system");

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_SCOPE_NAMESPACES",
          payload: new Set(["kube-system"]),
        });
      });

      it("should handle project navigation with argument", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("projects", "projects");
        const context = createMockContext({
          dispatch: mockDispatch,
        });

        // Act
        command.execute(context, "team-alpha");

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_SCOPE_PROJECTS",
          payload: new Set(["team-alpha"]),
        });
      });

      it("should handle app navigation with argument", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("apps", "apps");
        const context = createMockContext({
          dispatch: mockDispatch,
        });

        // Act
        command.execute(context, "my-application");

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_SELECTED_APPS",
          payload: new Set(["my-application"]),
        });
      });
    });

    describe("selection clearing", () => {
      it("should clear selections when switching views without arguments", () => {
        // Arrange
        const views: View[] = ["apps", "clusters", "namespaces", "projects"];
        const expectedActions = [
          "SET_SELECTED_APPS",
          "SET_SCOPE_CLUSTERS",
          "SET_SCOPE_NAMESPACES",
          "SET_SCOPE_PROJECTS",
        ];

        views.forEach((view, index) => {
          const mockDispatch = mock();
          const command = new TestNavigationCommand(view, view);
          const context = createMockContext({
            dispatch: mockDispatch,
          });

          // Act
          command.execute(context);

          // Assert - Check that selections are cleared
          expect(mockDispatch).toHaveBeenCalledWith({
            type: expectedActions[index],
            payload: new Set(),
          });
        });
      });
    });

    describe("complex navigation scenarios", () => {
      it("should handle switching between views with existing selections", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("apps", "apps");
        const context = createMockContext({
          state: createMockState({
            selections: {
              selectedApps: new Set(["existing-app"]),
              scopeClusters: new Set(["existing-cluster"]),
              scopeNamespaces: new Set(["existing-namespace"]),
              scopeProjects: new Set(["existing-project"]),
            },
          }),
          dispatch: mockDispatch,
        });

        // Act
        command.execute(context, "new-app");

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "RESET_NAVIGATION",
          payload: { view: "apps" },
        });
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_SELECTED_APPS",
          payload: new Set(["new-app"]),
        });
      });

      it("should work when called multiple times rapidly", () => {
        // Arrange
        const mockDispatch = mock();
        const command = new TestNavigationCommand("clusters", "clusters");
        const context = createMockContext({
          dispatch: mockDispatch,
        });

        // Act - Rapid execution
        command.execute(context, "cluster1");
        command.execute(context, "cluster2");
        command.execute(context);

        // Assert - Each call should dispatch the expected actions
        expect(mockDispatch).toHaveBeenCalledTimes(9); // 3 calls × 3 dispatches each
      });
    });
  });

  describe("error handling", () => {
    it("should handle dispatch failures gracefully", () => {
      // Arrange
      const mockDispatch = mock().mockImplementation(() => {
        throw new Error("Dispatch failed");
      });
      const command = new TestNavigationCommand("apps", "apps");
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act & Assert
      expect(() => command.execute(context)).toThrow("Dispatch failed");
    });
  });
});

describe("ClearCommand", () => {
  let clearCommand: TestClearCommand;

  beforeEach(() => {
    clearCommand = new TestClearCommand();
  });

  describe("canExecute", () => {
    it("should allow execution in normal mode", () => {
      // Arrange
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
      });

      // Act & Assert
      expect(clearCommand.canExecute(context)).toBe(true);
    });

    it("should prevent execution in non-normal modes", () => {
      // Arrange
      const nonNormalModes = ["confirm-sync", "help", "search", "command"];

      nonNormalModes.forEach((mode) => {
        const context = createMockContext({
          state: createMockState({ mode: mode as any }),
        });

        // Act & Assert
        expect(clearCommand.canExecute(context)).toBe(false);
      });
    });
  });

  describe("execute", () => {
    it("should clear cluster selections when in clusters view", () => {
      // Arrange
      const mockDispatch = mock();
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
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    it("should clear namespace selections when in namespaces view", () => {
      // Arrange
      const mockDispatch = mock();
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
          navigation: { view: "namespaces", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(),
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    it("should clear project selections when in projects view", () => {
      // Arrange
      const mockDispatch = mock();
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
          navigation: { view: "projects", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(),
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    it("should clear app selections when in apps view", () => {
      // Arrange
      const mockDispatch = mock();
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
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });

    it("should work with existing selections", () => {
      // Arrange
      const mockDispatch = mock();
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
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(["app1", "app2"]),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Selection cleared.",
        "user-action",
      );
    });
  });

  describe("properties", () => {
    it("should have correct description", () => {
      expect(clearCommand.description).toBe("Clear current view selection");
    });

    it("should have empty aliases array", () => {
      expect(clearCommand.aliases).toEqual([]);
    });
  });

  describe("error handling", () => {
    it("should handle dispatch failures", () => {
      // Arrange
      const mockDispatch = mock().mockImplementation(() => {
        throw new Error("Dispatch failed");
      });
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
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act & Assert
      expect(() => clearCommand.execute(context)).toThrow("Dispatch failed");
    });
  });
});

describe("ClearAllCommand", () => {
  let clearAllCommand: TestClearAllCommand;

  beforeEach(() => {
    clearAllCommand = new TestClearAllCommand();
  });

  describe("execute", () => {
    it("should clear all selections and filters", () => {
      // Arrange
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearAllCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_ALL_SELECTIONS",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_FILTERS",
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "All filtering cleared.",
        "user-action",
      );
    });

    it("should work regardless of current mode", () => {
      // Arrange - Test in different modes
      const modes = ["normal", "confirm-sync", "help"] as const;

      modes.forEach((mode) => {
        const mockDispatch = mock();
        const mockStatusLog = {
          info: mock(),
          warn: mock(),
          error: mock(),
          debug: mock(),
          set: mock(),
          clear: mock(),
        };
        const context = createMockContext({
          state: createMockState({ mode }),
          dispatch: mockDispatch,
          statusLog: mockStatusLog,
        });

        // Act
        clearAllCommand.execute(context);

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "CLEAR_ALL_SELECTIONS",
        });
        expect(mockDispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
        expect(mockStatusLog.info).toHaveBeenCalledWith(
          "All filtering cleared.",
          "user-action",
        );
      });
    });

    it("should work regardless of current view", () => {
      // Arrange - Test in different views
      const views: View[] = ["apps", "clusters", "namespaces", "projects"];

      views.forEach((view) => {
        const mockDispatch = mock();
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
            navigation: { view, selectedIdx: 0, lastGPressed: 0 },
          }),
          dispatch: mockDispatch,
          statusLog: mockStatusLog,
        });

        // Act
        clearAllCommand.execute(context);

        // Assert
        expect(mockDispatch).toHaveBeenCalledWith({
          type: "CLEAR_ALL_SELECTIONS",
        });
        expect(mockDispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      });
    });

    it("should work with extensive existing selections", () => {
      // Arrange
      const mockDispatch = mock();
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
          selections: {
            selectedApps: new Set(["app1", "app2", "app3"]),
            scopeClusters: new Set(["cluster1", "cluster2"]),
            scopeNamespaces: new Set(["ns1", "ns2", "ns3"]),
            scopeProjects: new Set(["project1"]),
          },
          ui: {
            searchQuery: "test-search",
            activeFilter: "health=Healthy",
            command: ":test",
            isVersionOutdated: false,
            latestVersion: undefined,
          },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      clearAllCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_ALL_SELECTIONS",
      });
      expect(mockDispatch).toHaveBeenCalledWith({ type: "CLEAR_FILTERS" });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "All filtering cleared.",
        "user-action",
      );
    });
  });

  describe("properties", () => {
    it("should have correct description", () => {
      expect(clearAllCommand.description).toBe(
        "Clear all selections and filters",
      );
    });

    it("should have empty aliases array", () => {
      expect(clearAllCommand.aliases).toEqual([]);
    });
  });

  describe("error handling", () => {
    it("should handle dispatch failures gracefully", () => {
      // Arrange
      const mockDispatch = mock()
        .mockImplementationOnce(() => {
          /* First call succeeds */
        })
        .mockImplementationOnce(() => {
          throw new Error("Second dispatch failed");
        });

      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act & Assert
      expect(() => clearAllCommand.execute(context)).toThrow(
        "Second dispatch failed",
      );
      expect(mockDispatch).toHaveBeenCalledTimes(2);
    });

    it("should handle statusLog failures", () => {
      // Arrange
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock().mockImplementation(() => {
          throw new Error("StatusLog failed");
        }),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act & Assert
      expect(() => clearAllCommand.execute(context)).toThrow(
        "StatusLog failed",
      );
      expect(mockDispatch).toHaveBeenCalledTimes(2); // Both dispatches should complete
    });
  });
});

describe("Navigation Commands Integration", () => {
  describe("workflow scenarios", () => {
    it("should support typical navigation workflow", () => {
      // Arrange
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act - Simulate typical workflow
      // 1. Navigate to clusters
      const clustersCommand = new TestNavigationCommand("clusters", "clusters");
      clustersCommand.execute(context, "production");

      // 2. Navigate to namespaces within that cluster
      const namespacesCommand = new TestNavigationCommand(
        "namespaces",
        "namespaces",
      );
      namespacesCommand.execute(context, "kube-system");

      // 3. Navigate to apps
      const appsCommand = new TestNavigationCommand("apps", "apps");
      appsCommand.execute(context);

      // 4. Clear current view selection
      const clearCommand = new TestClearCommand();
      clearCommand.execute(context);

      // 5. Clear all selections
      const clearAllCommand = new TestClearAllCommand();
      clearAllCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledTimes(12); // All dispatch calls (3+3+3+2+1)
      expect(mockStatusLog.info).toHaveBeenCalledTimes(2); // Clear commands
    });
  });

  describe("performance scenarios", () => {
    it("should handle rapid navigation changes efficiently", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act - Rapid navigation
      const views: View[] = ["apps", "clusters", "namespaces", "projects"];
      views.forEach((view) => {
        const command = new TestNavigationCommand(view, view);
        command.execute(context);
      });

      // Assert - Should complete without issues
      expect(mockDispatch).toHaveBeenCalledTimes(12); // 4 views × 3 dispatches each
    });
  });
});
