// src/__tests__/commands/SystemCommands.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";

import type { Command, CommandContext } from "../../commands/types";
import { createMockContext, createMockState } from "../test-utils";

// Test implementations of system command classes without external dependencies
class TestQuitCommand implements Command {
  aliases = ["quit", "exit"];
  description = "Exit the application";

  execute(context: CommandContext): void {
    context.cleanupAndExit();
  }
}

class TestHelpCommand implements Command {
  aliases = ["?"];
  description = "Show help";

  execute(context: CommandContext): void {
    context.dispatch({ type: "SET_MODE", payload: "help" });
  }
}

class TestLoginCommand implements Command {
  aliases = [];
  description = "Show login instructions";

  execute(context: CommandContext): void {
    const { dispatch, statusLog } = context;

    statusLog.error(
      "please use argocd login to authenticate before running argonaut",
      "auth",
    );
    dispatch({ type: "SET_MODE", payload: "auth-required" });
  }
}

class TestRulerCommand implements Command {
  aliases = [];
  description = "Open ruler line mode";

  execute(context: CommandContext): void {
    context.dispatch({ type: "SET_MODE", payload: "rulerline" });
  }
}

describe("QuitCommand", () => {
  let quitCommand: TestQuitCommand;

  beforeEach(() => {
    quitCommand = new TestQuitCommand();
  });

  describe("execute", () => {
    it("should call cleanupAndExit", () => {
      // Arrange
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        cleanupAndExit: mockCleanupAndExit,
      });

      // Act
      quitCommand.execute(context);

      // Assert
      expect(mockCleanupAndExit).toHaveBeenCalledTimes(1);
    });

    it("should handle cleanup gracefully even if context is incomplete", () => {
      // Arrange
      const mockCleanupAndExit = mock();
      const context = createMockContext({
        cleanupAndExit: mockCleanupAndExit,
      });

      // Act & Assert - should not throw
      expect(() => quitCommand.execute(context)).not.toThrow();
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });
  });

  describe("properties", () => {
    it("should have correct aliases", () => {
      expect(quitCommand.aliases).toEqual(["quit", "exit"]);
    });

    it("should have correct description", () => {
      expect(quitCommand.description).toBe("Exit the application");
    });
  });

  describe("edge cases", () => {
    it("should work with minimal context", () => {
      // Arrange
      const mockCleanupAndExit = mock();
      const minimalContext = {
        ...createMockContext(),
        cleanupAndExit: mockCleanupAndExit,
      };

      // Act & Assert
      expect(() => quitCommand.execute(minimalContext)).not.toThrow();
      expect(mockCleanupAndExit).toHaveBeenCalled();
    });
  });
});

describe("HelpCommand", () => {
  let helpCommand: TestHelpCommand;

  beforeEach(() => {
    helpCommand = new TestHelpCommand();
  });

  describe("execute", () => {
    it("should set mode to help", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act
      helpCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
      expect(mockDispatch).toHaveBeenCalledTimes(1);
    });

    it("should work regardless of current mode", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ mode: "confirm-sync" }),
        dispatch: mockDispatch,
      });

      // Act
      helpCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
    });

    it("should work regardless of authentication state", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ server: null }),
        dispatch: mockDispatch,
      });

      // Act
      helpCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
    });
  });

  describe("properties", () => {
    it("should have correct aliases", () => {
      expect(helpCommand.aliases).toEqual(["?"]);
    });

    it("should have correct description", () => {
      expect(helpCommand.description).toBe("Show help");
    });
  });

  describe("integration scenarios", () => {
    it("should work when called from different views", () => {
      // Test from apps view
      const mockDispatch = mock();
      const appsContext = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      helpCommand.execute(appsContext);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });

      // Reset mock
      mockDispatch.mockReset();

      // Test from clusters view
      const clustersContext = createMockContext({
        state: createMockState({
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0 },
        }),
        dispatch: mockDispatch,
      });

      helpCommand.execute(clustersContext);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
    });
  });
});

describe("LoginCommand", () => {
  let loginCommand: TestLoginCommand;

  beforeEach(() => {
    loginCommand = new TestLoginCommand();
  });

  describe("execute", () => {
    it("should show error message and set auth-required mode", () => {
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
      loginCommand.execute(context);

      // Assert
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "please use argocd login to authenticate before running argonaut",
        "auth",
      );
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "auth-required",
      });
    });

    it("should work when called from any authentication state", () => {
      // Arrange - Test with authenticated state
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const authenticatedContext = createMockContext({
        state: createMockState({
          server: {
            config: { baseUrl: "https://test.com" },
            token: "test-token",
          },
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act
      loginCommand.execute(authenticatedContext);

      // Assert - Should still show login message
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "please use argocd login to authenticate before running argonaut",
        "auth",
      );
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "auth-required",
      });
    });

    it("should work when called with minimal context", () => {
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
      const minimalContext = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act & Assert
      expect(() => loginCommand.execute(minimalContext)).not.toThrow();
      expect(mockStatusLog.error).toHaveBeenCalled();
      expect(mockDispatch).toHaveBeenCalled();
    });
  });

  describe("properties", () => {
    it("should have empty aliases array", () => {
      expect(loginCommand.aliases).toEqual([]);
    });

    it("should have correct description", () => {
      expect(loginCommand.description).toBe("Show login instructions");
    });
  });

  describe("error handling", () => {
    it("should handle dispatch failure gracefully", () => {
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
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act & Assert
      expect(() => loginCommand.execute(context)).toThrow("Dispatch failed");
      expect(mockStatusLog.error).toHaveBeenCalled();
    });

    it("should handle statusLog failure gracefully", () => {
      // Arrange
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock().mockImplementation(() => {
          throw new Error("StatusLog failed");
        }),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      // Act & Assert
      expect(() => loginCommand.execute(context)).toThrow("StatusLog failed");
    });
  });
});

describe("RulerCommand", () => {
  let rulerCommand: TestRulerCommand;

  beforeEach(() => {
    rulerCommand = new TestRulerCommand();
  });

  describe("execute", () => {
    it("should set mode to rulerline", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act
      rulerCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rulerline",
      });
      expect(mockDispatch).toHaveBeenCalledTimes(1);
    });

    it("should work regardless of current mode", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ mode: "confirm-sync" }),
        dispatch: mockDispatch,
      });

      // Act
      rulerCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rulerline",
      });
    });

    it("should work regardless of authentication state", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({ server: null }),
        dispatch: mockDispatch,
      });

      // Act
      rulerCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rulerline",
      });
    });

    it("should work from any view", () => {
      // Test from different views
      const views = ["apps", "clusters", "namespaces", "projects"] as const;

      views.forEach((view) => {
        const mockDispatch = mock();
        const context = createMockContext({
          state: createMockState({
            navigation: { view, selectedIdx: 0, lastGPressed: 0 },
          }),
          dispatch: mockDispatch,
        });

        rulerCommand.execute(context);

        expect(mockDispatch).toHaveBeenCalledWith({
          type: "SET_MODE",
          payload: "rulerline",
        });
      });
    });
  });

  describe("properties", () => {
    it("should have empty aliases array", () => {
      expect(rulerCommand.aliases).toEqual([]);
    });

    it("should have correct description", () => {
      expect(rulerCommand.description).toBe("Open ruler line mode");
    });
  });

  describe("edge cases", () => {
    it("should work with minimal context", () => {
      // Arrange
      const mockDispatch = mock();
      const minimalContext = {
        ...createMockContext(),
        dispatch: mockDispatch,
      };

      // Act & Assert
      expect(() => rulerCommand.execute(minimalContext)).not.toThrow();
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rulerline",
      });
    });
  });

  describe("error handling", () => {
    it("should handle dispatch failure", () => {
      // Arrange
      const mockDispatch = mock().mockImplementation(() => {
        throw new Error("Dispatch failed");
      });
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act & Assert
      expect(() => rulerCommand.execute(context)).toThrow("Dispatch failed");
    });
  });
});

describe("System Commands Integration", () => {
  describe("command interaction patterns", () => {
    it("should allow transitioning between system modes", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act - Simulate user workflow
      const helpCommand = new TestHelpCommand();
      const rulerCommand = new TestRulerCommand();

      // User opens help
      helpCommand.execute(context);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });

      // User switches to ruler mode
      rulerCommand.execute(context);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rulerline",
      });
    });

    it("should handle rapid command execution", () => {
      // Arrange
      const mockDispatch = mock();
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act - Rapid command execution
      const helpCommand = new TestHelpCommand();
      const rulerCommand = new TestRulerCommand();
      const loginCommand = new TestLoginCommand();

      helpCommand.execute(context);
      rulerCommand.execute(context);
      loginCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledTimes(3);
      expect(mockDispatch).toHaveBeenNthCalledWith(1, {
        type: "SET_MODE",
        payload: "help",
      });
      expect(mockDispatch).toHaveBeenNthCalledWith(2, {
        type: "SET_MODE",
        payload: "rulerline",
      });
      expect(mockDispatch).toHaveBeenNthCalledWith(3, {
        type: "SET_MODE",
        payload: "auth-required",
      });
    });
  });

  describe("error resilience", () => {
    it("should continue working after individual command failures", () => {
      // Arrange
      const mockDispatch = mock()
        .mockImplementationOnce(() => {
          throw new Error("First call failed");
        })
        .mockImplementationOnce(() => {
          /* Second call succeeds */
        });

      const context = createMockContext({
        dispatch: mockDispatch,
      });

      // Act & Assert
      const helpCommand = new TestHelpCommand();
      const rulerCommand = new TestRulerCommand();

      // First command fails
      expect(() => helpCommand.execute(context)).toThrow("First call failed");

      // Second command should still work
      expect(() => rulerCommand.execute(context)).not.toThrow();
      expect(mockDispatch).toHaveBeenCalledTimes(2);
    });
  });
});
