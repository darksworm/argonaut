// src/__tests__/handlers/NavigationInputHandler.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";
import { NavigationInputHandler } from "../../commands/handlers/keyboard";
import { createMockContext, createMockState } from "../test-utils";

describe("NavigationInputHandler", () => {
  let handler: NavigationInputHandler;

  beforeEach(() => {
    handler = new NavigationInputHandler();
  });

  describe("canHandle", () => {
    it("should handle input only in normal mode", () => {
      const normalModeContext = createMockContext({
        state: createMockState({ mode: "normal" }),
      });

      expect(handler.canHandle(normalModeContext)).toBe(true);
    });

    it("should not handle input in loading mode", () => {
      const loadingModeContext = createMockContext({
        state: createMockState({ mode: "loading" }),
      });

      expect(handler.canHandle(loadingModeContext)).toBe(false);
    });

    it("should not handle input in command mode", () => {
      const commandModeContext = createMockContext({
        state: createMockState({ mode: "command" }),
      });

      expect(handler.canHandle(commandModeContext)).toBe(false);
    });

    it("should not handle input in search mode", () => {
      const searchModeContext = createMockContext({
        state: createMockState({ mode: "search" }),
      });

      expect(handler.canHandle(searchModeContext)).toBe(false);
    });
  });

  describe("navigation keys", () => {
    it("should handle j key (down) navigation", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("j", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: expect.any(Number),
      });
    });

    it("should handle k key (up) navigation", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 2, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("k", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 1,
      });
    });

    it("should handle down arrow navigation", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { downArrow: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: expect.any(Number),
      });
    });

    it("should handle up arrow navigation", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 2, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { upArrow: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 1,
      });
    });

    it("should respect lower bounds (not go below 0)", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      handler.handleInput("k", {}, context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0, // Should not go below 0
      });
    });

    it("should handle gg (go to top) with timing", () => {
      const mockDispatch = mock();
      const now = Date.now();
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "apps",
            selectedIdx: 5,
            lastGPressed: now - 100,
            lastEscPressed: 0,
          }, // Recent g press
        }),
        dispatch: mockDispatch,
      });

      // Mock Date.now to return consistent value
      const originalDateNow = Date.now;
      Date.now = mock().mockReturnValue(now);

      const result = handler.handleInput("g", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0, // Go to top
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_LAST_G_PRESSED",
        payload: now,
      });

      // Restore Date.now
      Date.now = originalDateNow;
    });

    it("should handle single g without double-g effect", () => {
      const mockDispatch = mock();
      const now = Date.now();
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "apps",
            selectedIdx: 5,
            lastGPressed: now - 1000,
          }, // Old g press
        }),
        dispatch: mockDispatch,
      });

      // Mock Date.now to return consistent value
      const originalDateNow = Date.now;
      Date.now = mock().mockReturnValue(now);

      const result = handler.handleInput("g", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).not.toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0, // Should not go to top
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_LAST_G_PRESSED",
        payload: now,
      });

      // Restore Date.now
      Date.now = originalDateNow;
    });

    it("should handle G (go to bottom)", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("G", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: expect.any(Number), // Should be max index
      });
    });
  });

  describe("action keys", () => {
    it("should handle Enter (drill down)", () => {
      const mockNavigationActions = {
        drillDown: mock(),
        toggleSelection: mock(),
      };
      const context = createMockContext({
        navigationActions: mockNavigationActions,
      });

      const result = handler.handleInput("", { return: true }, context);

      expect(result).toBe(true);
      expect(mockNavigationActions.drillDown).toHaveBeenCalled();
    });

    it("should handle Space (toggle selection) in apps view", () => {
      const mockNavigationActions = {
        drillDown: mock(),
        toggleSelection: mock(),
      };
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
        navigationActions: mockNavigationActions,
      });

      const result = handler.handleInput(" ", {}, context);

      expect(result).toBe(true);
      expect(mockNavigationActions.toggleSelection).toHaveBeenCalled();
    });

    it("should not handle Space in non-apps views", () => {
      const mockNavigationActions = {
        drillDown: mock(),
        toggleSelection: mock(),
      };
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
        navigationActions: mockNavigationActions,
      });

      const result = handler.handleInput(" ", {}, context);

      expect(result).toBe(false);
      expect(mockNavigationActions.toggleSelection).not.toHaveBeenCalled();
    });

    it("should handle 'd' key (diff) in apps view with single app", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(), // No multiple apps selected
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("d", {}, context);

      expect(result).toBe(true);
      expect(mockExecuteCommand).toHaveBeenCalledWith("diff");
    });

    it("should not handle 'd' key in non-apps views", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("d", {}, context);

      expect(result).toBe(false);
      expect(mockExecuteCommand).not.toHaveBeenCalled();
    });

    it("should not handle 'd' key when multiple apps are selected", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(["app1", "app2"]), // Multiple apps selected
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("d", {}, context);

      expect(result).toBe(false);
      expect(mockExecuteCommand).not.toHaveBeenCalled();
    });

    it("should handle 'd' key when exactly one app is selected", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(["single-app"]), // Exactly one app selected
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("d", {}, context);

      expect(result).toBe(true);
      expect(mockExecuteCommand).toHaveBeenCalledWith("diff");
    });

    it("should not handle 'D' key (case sensitive)", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(),
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("D", {}, context);

      expect(result).toBe(false);
      expect(mockExecuteCommand).not.toHaveBeenCalled();
    });

    it("should handle 's' key (sync) in apps view", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(["app1", "app2"]), // Multiple apps allowed
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("s", {}, context);

      expect(result).toBe(true);
      expect(mockExecuteCommand).toHaveBeenCalledWith("sync");
    });

    it("should not handle 's' key in non-apps views", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "clusters",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("s", {}, context);

      expect(result).toBe(false);
      expect(mockExecuteCommand).not.toHaveBeenCalled();
    });

    it("should not handle 'S' key (case sensitive)", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: {
            view: "apps",
            selectedIdx: 0,
            lastGPressed: 0,
            lastEscPressed: 0,
          },
        }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("S", {}, context);

      expect(result).toBe(false);
      expect(mockExecuteCommand).not.toHaveBeenCalled();
    });

    it("should handle Escape (clear current view selections) - clusters", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(),
      });
    });

    it("should handle Escape (clear current view selections) - apps", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(["test-app", "test-app-2"]), // Has selections
          },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
      });

      expect(mockDispatch).not.toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "projects",
      });
    });

    it("should handle Escape (navigate up) when no selections exist", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
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
            selectedApps: new Set(["app"]),
          },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      // Should navigate up with all UpCommand actions: set index, clear filters, clear apps, and set view
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0,
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_FILTERS",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(),
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "projects",
      });
    });
  });

  describe("priority", () => {
    it("should have high priority (10)", () => {
      expect(handler.priority).toBe(10);
    });
  });

  describe("unhandled input", () => {
    it("should return false for unhandled keys", () => {
      const context = createMockContext();

      const result = handler.handleInput("x", {}, context);

      expect(result).toBe(false);
    });

    it("should return false for unhandled special keys", () => {
      const context = createMockContext();

      const result = handler.handleInput("", { tab: true }, context);

      expect(result).toBe(false);
    });
  });
});
