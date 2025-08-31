// src/__tests__/handlers/InputHandlers.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";
import {
  GlobalInputHandler,
  ModeInputHandler,
  SearchInputHandler,
} from "../../commands/handlers/keyboard";
import { createMockContext, createMockState } from "../test-utils";

describe("ModeInputHandler", () => {
  let handler: ModeInputHandler;

  beforeEach(() => {
    handler = new ModeInputHandler();
  });

  describe("canHandle", () => {
    it("should handle input only in normal mode", () => {
      const normalModeContext = createMockContext({
        state: createMockState({ mode: "normal" }),
      });

      expect(handler.canHandle(normalModeContext)).toBe(true);
    });

    it("should not handle input in other modes", () => {
      const loadingModeContext = createMockContext({
        state: createMockState({ mode: "loading" }),
      });

      expect(handler.canHandle(loadingModeContext)).toBe(false);
    });
  });

  describe("mode switching", () => {
    it("should handle ? (help mode)", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("?", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
    });

    it("should handle / (search mode)", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("/", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "search",
      });
    });

    it("should handle : (command mode)", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput(":", {}, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "command",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_COMMAND",
        payload: "",
      });
    });
  });

  describe("priority", () => {
    it("should have priority 20", () => {
      expect(handler.priority).toBe(20);
    });
  });

  describe("unhandled input", () => {
    it("should return false for unhandled keys", () => {
      const context = createMockContext();

      const result = handler.handleInput("x", {}, context);

      expect(result).toBe(false);
    });
  });
});

describe("SearchInputHandler", () => {
  let handler: SearchInputHandler;

  beforeEach(() => {
    handler = new SearchInputHandler();
  });

  describe("canHandle", () => {
    it("should handle input only in search mode", () => {
      const searchModeContext = createMockContext({
        state: createMockState({ mode: "search" }),
      });

      expect(handler.canHandle(searchModeContext)).toBe(true);
    });

    it("should not handle input in normal mode", () => {
      const normalModeContext = createMockContext({
        state: createMockState({ mode: "normal" }),
      });

      expect(handler.canHandle(normalModeContext)).toBe(false);
    });
  });

  describe("search mode navigation", () => {
    it("should handle Escape (exit search mode)", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ mode: "search" }),
        dispatch: mockDispatch,
      });

      const result = handler.handleInput("", { escape: true }, context);

      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SEARCH_QUERY",
        payload: "",
      });
    });

    it("should handle down arrow navigation", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          mode: "search",
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
          mode: "search",
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
  });

  describe("priority", () => {
    it("should have highest priority (30)", () => {
      expect(handler.priority).toBe(30);
    });
  });

  describe("unhandled input", () => {
    it("should return false for unhandled keys (let TextInput handle)", () => {
      const context = createMockContext({
        state: createMockState({ mode: "search" }),
      });

      const result = handler.handleInput("a", {}, context);

      expect(result).toBe(false); // Let TextInput handle typing
    });
  });
});

describe("GlobalInputHandler", () => {
  let handler: GlobalInputHandler;

  beforeEach(() => {
    handler = new GlobalInputHandler();
  });

  describe("canHandle", () => {
    it("should always handle input", () => {
      const context = createMockContext();

      expect(handler.canHandle(context)).toBe(true);
    });
  });

  describe("global actions", () => {
    it("should handle Ctrl+C (exit)", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("c", { ctrl: true }, context);

      expect(result).toBe(true);
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });

    it("should handle escape sequence (exit)", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("\u0003", {}, context);

      expect(result).toBe(true);
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });

    it("should handle q (quit) in normal mode", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("q", {}, context);

      expect(result).toBe(true);
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });

    it("should handle Q (quit) in normal mode (case insensitive)", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("Q", {}, context);

      expect(result).toBe(true);
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });

    it("should handle q (quit) in auth-required mode", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        state: createMockState({ mode: "auth-required" }),
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("q", {}, context);

      expect(result).toBe(true);
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });

    it("should handle q (quit) in loading mode", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        state: createMockState({ mode: "loading" }),
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("q", {}, context);

      expect(result).toBe(true);
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });

    it("should not handle q in command mode", () => {
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        state: createMockState({ mode: "command" }),
        cleanupAndExit: mockCleanupAndExit,
      });

      const result = handler.handleInput("q", {}, context);

      expect(result).toBe(false);
      expect(mockCleanupAndExit).not.toHaveBeenCalled();
    });

    it("should handle l (logs) in auth-required mode", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({ mode: "auth-required" }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("l", {}, context);

      expect(result).toBe(true);
      expect(mockExecuteCommand).toHaveBeenCalledWith("logs");
    });

    it("should handle L (logs) in auth-required mode (case insensitive)", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({ mode: "auth-required" }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("L", {}, context);

      expect(result).toBe(true);
      expect(mockExecuteCommand).toHaveBeenCalledWith("logs");
    });

    it("should not handle l in normal mode", () => {
      const mockExecuteCommand = mock();
      const context = createMockContext({
        state: createMockState({ mode: "normal" }),
        executeCommand: mockExecuteCommand,
      });

      const result = handler.handleInput("l", {}, context);

      expect(result).toBe(false);
      expect(mockExecuteCommand).not.toHaveBeenCalled();
    });
  });

  describe("priority", () => {
    it("should have lowest priority (0)", () => {
      expect(handler.priority).toBe(0);
    });
  });

  describe("unhandled input", () => {
    it("should return false for unhandled keys", () => {
      const context = createMockContext();

      const result = handler.handleInput("x", {}, context);

      expect(result).toBe(false);
    });
  });
});
