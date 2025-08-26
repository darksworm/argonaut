// src/__tests__/handlers/NavigationInputHandler.test.ts
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
      const mockDispatch = jest.fn();
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
      const mockDispatch = jest.fn();
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
      const mockDispatch = jest.fn();
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
      const mockDispatch = jest.fn();
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
      const mockDispatch = jest.fn();
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
      const mockDispatch = jest.fn();
      const now = Date.now();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 5, lastGPressed: now - 100 }, // Recent g press
        }),
        dispatch: mockDispatch,
      });

      // Mock Date.now to return consistent value
      const originalDateNow = Date.now;
      Date.now = jest.fn().mockReturnValue(now);

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
      const mockDispatch = jest.fn();
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
      Date.now = jest.fn().mockReturnValue(now);

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
      const mockDispatch = jest.fn();
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
        drillDown: jest.fn(),
        toggleSelection: jest.fn(),
      };
      const context = createMockContext({
        navigationActions: mockNavigationActions,
      });

      const result = handler.handleInput("", { return: true }, context);

      expect(result).toBe(true);
      expect(mockNavigationActions.drillDown).toHaveBeenCalled();
    });

    it("should handle Space (toggle selection)", () => {
      const mockNavigationActions = {
        drillDown: jest.fn(),
        toggleSelection: jest.fn(),
      };
      const context = createMockContext({
        navigationActions: mockNavigationActions,
      });

      const result = handler.handleInput(" ", {}, context);

      expect(result).toBe(true);
      expect(mockNavigationActions.toggleSelection).toHaveBeenCalled();
    });

    it("should handle Escape (clear current view selections) - clusters", () => {
      const mockDispatch = jest.fn();
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

    it("should handle Escape (clear current view selections) - namespaces", () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "namespaces", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(),
      });
    });

    it("should handle Escape (clear current view selections) - projects", () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "projects", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(),
      });
    });

    it("should handle Escape (clear current view selections) - apps", () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(),
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
