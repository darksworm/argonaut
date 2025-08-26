// src/__tests__/commands/CommandIntegration.test.ts
import { mock } from "bun:test";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

import type { Command, CommandContext } from "../../commands/types";
import { createMockContext } from "../test-utils";

// Mock Command Registry for integration testing
class MockCommandRegistry {
  private commands = new Map<string, Command>();
  private aliases = new Map<string, string>();

  register(name: string, command: Command): void {
    this.commands.set(name, command);

    // Register aliases
    command.aliases?.forEach((alias) => {
      this.aliases.set(alias, name);
    });
  }

  async execute(
    commandLine: string,
    context: CommandContext,
  ): Promise<boolean> {
    const [commandName, ...args] = commandLine.trim().split(/\s+/);

    if (!commandName) return false;

    // Check exact command name first, then aliases
    let command = this.commands.get(commandName);
    if (!command) {
      const actualCommandName = this.aliases.get(commandName);
      if (actualCommandName) {
        command = this.commands.get(actualCommandName);
      }
    }

    if (!command) {
      context.statusLog.error(`Unknown command: ${commandName}`, "command");
      return false;
    }

    // Check if command can execute
    if (command.canExecute && !command.canExecute(context)) {
      context.statusLog.warn(
        `Command ${commandName} cannot be executed in current state`,
        "command",
      );
      return false;
    }

    try {
      await command.execute(context, ...args);
      return true;
    } catch (error: any) {
      context.statusLog.error(`Command failed: ${error.message}`, "command");
      return false;
    }
  }

  getCommands(): string[] {
    return Array.from(this.commands.keys());
  }

  getAliases(): string[] {
    return Array.from(this.aliases.keys());
  }
}

// Test command implementations for integration testing
class MockSyncCommand implements Command {
  aliases = [];
  description = "Sync application(s)";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  execute(context: CommandContext, arg?: string): void {
    const { dispatch } = context;
    if (arg) {
      dispatch({ type: "SET_CONFIRM_TARGET", payload: arg });
      dispatch({ type: "SET_MODE", payload: "confirm-sync" });
    }
  }
}

class MockQuitCommand implements Command {
  aliases = ["quit", "exit"];
  description = "Exit the application";

  execute(context: CommandContext): void {
    context.cleanupAndExit();
  }
}

class MockHelpCommand implements Command {
  aliases = ["?"];
  description = "Show help";

  execute(context: CommandContext): void {
    context.dispatch({ type: "SET_MODE", payload: "help" });
  }
}

class MockNavigationCommand implements Command {
  aliases: string[];
  description: string;

  constructor(
    private targetView: string,
    aliases: string[] = [],
  ) {
    this.aliases = aliases;
    this.description = `Switch to ${targetView} view`;
  }

  canExecute(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  execute(context: CommandContext): void {
    context.dispatch({
      type: "RESET_NAVIGATION",
      payload: { view: this.targetView as any },
    });
  }
}

class MockAsyncCommand implements Command {
  aliases = [];
  description = "Async test command";

  async execute(context: CommandContext, delay?: string): Promise<void> {
    const delayMs = parseInt(delay || "10", 10);
    context.statusLog.info(
      `Starting async operation with ${delayMs}ms delay`,
      "async",
    );

    await new Promise((resolve) => setTimeout(resolve, delayMs));

    context.statusLog.info("Async operation completed", "async");
    context.dispatch({ type: "SET_MODE", payload: "normal" });
  }
}

class MockFailingCommand implements Command {
  aliases = [];
  description = "Command that always fails";

  execute(_context: CommandContext): void {
    throw new Error("This command always fails");
  }
}

describe("Command Integration Tests", () => {
  let registry: MockCommandRegistry;
  let context: CommandContext;
  let mockDispatch: mock.Mock;
  let mockStatusLog: any;
  let mockCleanupAndExit: mock.Mock;

  beforeEach(() => {
    registry = new MockCommandRegistry();
    mockDispatch = mock();
    mockStatusLog = {
      info: mock(),
      warn: mock(),
      error: mock(),
      debug: mock(),
      set: mock(),
      clear: mock(),
    };
    mockCleanupAndExit = mock();

    context = createMockContext({
      dispatch: mockDispatch,
      statusLog: mockStatusLog,
      cleanupAndExit: mockCleanupAndExit,
    });

    // Register test commands
    registry.register("sync", new MockSyncCommand());
    registry.register("quit", new MockQuitCommand());
    registry.register("help", new MockHelpCommand());
    registry.register("apps", new MockNavigationCommand("apps", ["app", "a"]));
    registry.register("async", new MockAsyncCommand());
    registry.register("fail", new MockFailingCommand());
  });

  describe("basic command execution", () => {
    it("should execute simple command successfully", async () => {
      // Act
      const result = await registry.execute("help", context);

      // Assert
      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
    });

    it("should execute command with arguments", async () => {
      // Act
      const result = await registry.execute("sync my-app", context);

      // Assert
      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "my-app",
      });
    });

    it("should handle unknown commands gracefully", async () => {
      // Act
      const result = await registry.execute("unknown-command", context);

      // Assert
      expect(result).toBe(false);
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Unknown command: unknown-command",
        "command",
      );
    });

    it("should handle empty command lines", async () => {
      // Act
      const result = await registry.execute("", context);

      // Assert
      expect(result).toBe(false);
    });

    it("should handle whitespace-only command lines", async () => {
      // Act
      const result = await registry.execute("   ", context);

      // Assert
      expect(result).toBe(false);
    });
  });

  describe("command aliases", () => {
    it("should execute commands via aliases", async () => {
      // Act
      const result = await registry.execute("?", context);

      // Assert
      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
    });

    it("should handle multiple aliases for same command", async () => {
      // Act & Assert
      const aliases = ["quit", "exit"];
      for (const alias of aliases) {
        mockCleanupAndExit.mockClear();
        const result = await registry.execute(alias, context);

        expect(result).toBe(true);
        expect(mockCleanupAndExit).toHaveBeenCalledTimes(1);
      }
    });

    it("should prioritize exact command names over aliases", async () => {
      // Register command with same name as existing alias
      registry.register("exit", new MockHelpCommand());

      // Act
      const result = await registry.execute("exit", context);

      // Assert - Should execute the direct command, not the alias
      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });
      expect(mockCleanupAndExit).not.toHaveBeenCalled();
    });
  });

  describe("command execution conditions", () => {
    it("should respect canExecute conditions", async () => {
      // Arrange - Set state where sync cannot execute
      context.state.server = null;

      // Act
      const result = await registry.execute("sync test-app", context);

      // Assert
      expect(result).toBe(false);
      expect(mockStatusLog.warn).toHaveBeenCalledWith(
        "Command sync cannot be executed in current state",
        "command",
      );
    });

    it("should execute when conditions are met", async () => {
      // Arrange - Ensure sync can execute
      context.state.server = {
        config: { baseUrl: "https://test.com" },
        token: "test",
      };

      // Act
      const result = await registry.execute("sync test-app", context);

      // Assert
      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_CONFIRM_TARGET",
        payload: "test-app",
      });
    });

    it("should handle commands without canExecute method", async () => {
      // Act
      const result = await registry.execute("help", context);

      // Assert - Should execute successfully
      expect(result).toBe(true);
    });
  });

  describe("error handling and recovery", () => {
    it("should handle command execution errors", async () => {
      // Act
      const result = await registry.execute("fail", context);

      // Assert
      expect(result).toBe(false);
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Command failed: This command always fails",
        "command",
      );
    });

    it("should continue working after command failures", async () => {
      // Act
      const failResult = await registry.execute("fail", context);
      const successResult = await registry.execute("help", context);

      // Assert
      expect(failResult).toBe(false);
      expect(successResult).toBe(true);
    });

    it("should handle async command errors", async () => {
      // Arrange - Create async command that fails
      class FailingAsyncCommand implements Command {
        aliases = [];
        description = "Failing async command";

        async execute(): Promise<void> {
          await new Promise((resolve) => setTimeout(resolve, 10));
          throw new Error("Async operation failed");
        }
      }
      registry.register("async-fail", new FailingAsyncCommand());

      // Act
      const result = await registry.execute("async-fail", context);

      // Assert
      expect(result).toBe(false);
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Command failed: Async operation failed",
        "command",
      );
    });
  });

  describe("async command execution", () => {
    it("should handle async commands properly", async () => {
      // Act
      const result = await registry.execute("async 50", context);

      // Assert
      expect(result).toBe(true);
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Starting async operation with 50ms delay",
        "async",
      );
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Async operation completed",
        "async",
      );
    });

    it("should handle concurrent async commands", async () => {
      // Act
      const promises = [
        registry.execute("async 20", context),
        registry.execute("async 30", context),
        registry.execute("async 10", context),
      ];

      const results = await Promise.all(promises);

      // Assert
      expect(results).toEqual([true, true, true]);
      expect(mockStatusLog.info).toHaveBeenCalledTimes(6); // 3 start + 3 complete
    });
  });

  describe("command workflow integration", () => {
    it("should support typical user workflow", async () => {
      // Arrange
      context.state.server = {
        config: { baseUrl: "https://test.com" },
        token: "test",
      };

      // Act - Simulate typical workflow
      const results = await Promise.all([
        registry.execute("apps", context), // Navigate to apps
        registry.execute("sync test-app", context), // Sync an app
        registry.execute("help", context), // Show help
        registry.execute("quit", context), // Quit
      ]);

      // Assert
      expect(results).toEqual([true, true, true, true]);
      expect(mockCleanupAndExit).toHaveBeenCalledTimes(1);
    });

    it("should handle rapid command sequences", async () => {
      // Arrange
      const commands = ["help", "apps", "help", "apps", "help"];

      // Act
      const results = [];
      for (const cmd of commands) {
        results.push(await registry.execute(cmd, context));
      }

      // Assert
      expect(results).toEqual([true, true, true, true, true]);
      expect(mockDispatch).toHaveBeenCalledTimes(5); // 5 commands, help=1 dispatch each
    });

    it("should handle mixed success and failure scenarios", async () => {
      // Arrange
      const commands = [
        { cmd: "help", expected: true },
        { cmd: "fail", expected: false },
        { cmd: "unknown", expected: false },
        { cmd: "help", expected: true },
        { cmd: "quit", expected: true },
      ];

      // Act & Assert
      for (const { cmd, expected } of commands) {
        const result = await registry.execute(cmd, context);
        expect(result).toBe(expected);
      }
    });
  });

  describe("command state consistency", () => {
    it("should maintain state consistency across commands", async () => {
      // Arrange
      context.state.mode = "normal";

      // Act - Execute sequence that changes mode
      await registry.execute("help", context);
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "help",
      });

      // Simulate state change
      context.state.mode = "help";

      // Act - Try navigation (which requires normal mode)
      const navResult = await registry.execute("apps", context);

      // Assert
      expect(navResult).toBe(false);
      expect(mockStatusLog.warn).toHaveBeenCalledWith(
        "Command apps cannot be executed in current state",
        "command",
      );
    });

    it("should handle state modifications during command execution", async () => {
      // Arrange - Command that modifies state during execution
      class StateModifyingCommand implements Command {
        aliases = [];
        description = "Modifies state during execution";

        execute(context: CommandContext): void {
          context.dispatch({ type: "SET_MODE", payload: "normal" });
          context.dispatch({ type: "SET_MODE", payload: "help" });
        }
      }
      registry.register("modify-state", new StateModifyingCommand());

      // Act
      const result = await registry.execute("modify-state", context);

      // Assert
      expect(result).toBe(true);
      expect(mockDispatch).toHaveBeenCalledTimes(2);
      expect(mockDispatch).toHaveBeenNthCalledWith(1, {
        type: "SET_MODE",
        payload: "normal",
      });
      expect(mockDispatch).toHaveBeenNthCalledWith(2, {
        type: "SET_MODE",
        payload: "help",
      });
    });
  });

  describe("performance and scalability", () => {
    it("should handle large number of registered commands efficiently", () => {
      // Arrange - Register many commands
      const startTime = performance.now();

      for (let i = 0; i < 1000; i++) {
        registry.register(`cmd${i}`, new MockHelpCommand());
      }

      const registrationTime = performance.now() - startTime;

      // Assert
      expect(registrationTime).toBeLessThan(100); // Should be fast
      expect(registry.getCommands()).toHaveLength(1006); // Original 6 + 1000 new
    });

    it("should execute commands efficiently even with many registered", async () => {
      // Arrange - Register many commands
      for (let i = 0; i < 1000; i++) {
        registry.register(`cmd${i}`, new MockHelpCommand());
      }

      const startTime = performance.now();

      // Act
      const result = await registry.execute("help", context);

      const executionTime = performance.now() - startTime;

      // Assert
      expect(result).toBe(true);
      expect(executionTime).toBeLessThan(10); // Should be very fast
    });

    it("should handle command arguments parsing efficiently", async () => {
      // Arrange
      const longArguments = Array.from(
        { length: 100 },
        (_, i) => `arg${i}`,
      ).join(" ");
      const commandLine = `sync ${longArguments}`;

      const startTime = performance.now();

      // Act
      const result = await registry.execute(commandLine, context);

      const executionTime = performance.now() - startTime;

      // Assert
      expect(result).toBe(true);
      expect(executionTime).toBeLessThan(10);
    });
  });

  describe("command lifecycle", () => {
    it("should track command execution metrics", async () => {
      // Arrange
      const executionCounts = new Map<string, number>();

      const instrumentedRegistry = new (class extends MockCommandRegistry {
        async execute(
          commandLine: string,
          context: CommandContext,
        ): Promise<boolean> {
          const commandName = commandLine.trim().split(/\s+/)[0];
          executionCounts.set(
            commandName,
            (executionCounts.get(commandName) || 0) + 1,
          );
          return super.execute(commandLine, context);
        }
      })();

      // Register commands
      instrumentedRegistry.register("help", new MockHelpCommand());
      instrumentedRegistry.register("quit", new MockQuitCommand());

      // Act
      await instrumentedRegistry.execute("help", context);
      await instrumentedRegistry.execute("help", context);
      await instrumentedRegistry.execute("quit", context);

      // Assert
      expect(executionCounts.get("help")).toBe(2);
      expect(executionCounts.get("quit")).toBe(1);
    });
  });
});
