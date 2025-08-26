// src/__tests__/commands/CommandRegistry.test.ts
import { CommandRegistry } from "../../commands/registry";
import type { InputHandler } from "../../commands/types";
import { createMockCommand, createMockContext } from "../test-utils";

describe("CommandRegistry", () => {
  let registry: CommandRegistry;

  beforeEach(() => {
    registry = new CommandRegistry();
  });

  describe("registerCommand", () => {
    it("should register commands with aliases", () => {
      const mockCommand = createMockCommand({
        aliases: ["s", "synchronize"],
      });

      registry.registerCommand("sync", mockCommand);

      // Test that all variations work
      const context = createMockContext();
      registry.executeCommand("sync", context);
      registry.executeCommand("s", context);
      registry.executeCommand("synchronize", context);

      expect(mockCommand.execute).toHaveBeenCalledTimes(3);
    });

    it("should handle commands without aliases", () => {
      const mockCommand = createMockCommand();

      registry.registerCommand("test", mockCommand);

      const context = createMockContext();
      registry.executeCommand("test", context);

      expect(mockCommand.execute).toHaveBeenCalledTimes(1);
    });

    it("should handle case-insensitive command names", () => {
      const mockCommand = createMockCommand();

      registry.registerCommand("Sync", mockCommand);

      const context = createMockContext();
      registry.executeCommand("sync", context);
      registry.executeCommand("SYNC", context);
      registry.executeCommand("Sync", context);

      expect(mockCommand.execute).toHaveBeenCalledTimes(3);
    });
  });

  describe("executeCommand", () => {
    it("should execute commands with proper context", async () => {
      const mockCommand = createMockCommand();
      registry.registerCommand("test", mockCommand);

      const context = createMockContext();
      const result = await registry.executeCommand(
        "test",
        context,
        "arg1",
        "arg2",
      );

      expect(result).toBe(true);
      expect(mockCommand.execute).toHaveBeenCalledWith(context, "arg1", "arg2");
    });

    it("should handle command not found", async () => {
      const context = createMockContext();
      const result = await registry.executeCommand("nonexistent", context);

      expect(result).toBe(false);
    });

    it("should respect canExecute checks", async () => {
      const mockCommand = createMockCommand({
        canExecute: jest.fn().mockReturnValue(false),
      });
      registry.registerCommand("test", mockCommand);

      const context = createMockContext();
      const result = await registry.executeCommand("test", context);

      expect(result).toBe(false);
      expect(mockCommand.canExecute).toHaveBeenCalledWith(context);
      expect(mockCommand.execute).not.toHaveBeenCalled();
    });

    it("should handle command execution errors", async () => {
      const error = new Error("Command failed");
      const mockCommand = createMockCommand({
        execute: jest.fn().mockRejectedValue(error),
      });
      registry.registerCommand("failing", mockCommand);

      const context = createMockContext();
      const result = await registry.executeCommand("failing", context);

      expect(result).toBe(false);
      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Command 'failing' failed: Command failed",
        "command",
      );
    });

    it("should handle non-Error exceptions", async () => {
      const mockCommand = createMockCommand({
        execute: jest.fn().mockRejectedValue("String error"),
      });
      registry.registerCommand("failing", mockCommand);

      const context = createMockContext();
      const result = await registry.executeCommand("failing", context);

      expect(result).toBe(false);
      expect(context.statusLog.error).toHaveBeenCalledWith(
        "Command 'failing' failed: String error",
        "command",
      );
    });
  });

  describe("registerInputHandler", () => {
    it("should register input handlers", () => {
      const mockHandler: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true),
        canHandle: jest.fn().mockReturnValue(true),
        priority: 5,
      };

      registry.registerInputHandler(mockHandler);

      const context = createMockContext();
      const result = registry.handleInput("test", {}, context);

      expect(result).toBe(true);
      expect(mockHandler.handleInput).toHaveBeenCalledWith("test", {}, context);
    });

    it("should prioritize input handlers correctly", () => {
      const lowPriorityHandler: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true),
        canHandle: jest.fn().mockReturnValue(true),
        priority: 1,
      };

      const highPriorityHandler: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true),
        canHandle: jest.fn().mockReturnValue(true),
        priority: 10,
      };

      // Register low priority first
      registry.registerInputHandler(lowPriorityHandler);
      registry.registerInputHandler(highPriorityHandler);

      const context = createMockContext();
      registry.handleInput("test", {}, context);

      // High priority should be called first
      expect(highPriorityHandler.handleInput).toHaveBeenCalled();
      expect(lowPriorityHandler.handleInput).not.toHaveBeenCalled();
    });

    it("should fall through to lower priority handlers", () => {
      const firstHandler: InputHandler = {
        handleInput: jest.fn().mockReturnValue(false), // doesn't handle
        canHandle: jest.fn().mockReturnValue(true),
        priority: 10,
      };

      const secondHandler: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true), // handles it
        canHandle: jest.fn().mockReturnValue(true),
        priority: 5,
      };

      registry.registerInputHandler(firstHandler);
      registry.registerInputHandler(secondHandler);

      const context = createMockContext();
      const result = registry.handleInput("test", {}, context);

      expect(result).toBe(true);
      expect(firstHandler.handleInput).toHaveBeenCalled();
      expect(secondHandler.handleInput).toHaveBeenCalled();
    });

    it("should respect canHandle checks", () => {
      const mockHandler: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true),
        canHandle: jest.fn().mockReturnValue(false), // can't handle
        priority: 5,
      };

      registry.registerInputHandler(mockHandler);

      const context = createMockContext();
      const result = registry.handleInput("test", {}, context);

      expect(result).toBe(false);
      expect(mockHandler.canHandle).toHaveBeenCalledWith(context);
      expect(mockHandler.handleInput).not.toHaveBeenCalled();
    });

    it("should handle handlers without priority (default to 0)", () => {
      const handlerWithoutPriority: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true),
        canHandle: jest.fn().mockReturnValue(true),
        // no priority specified
      };

      const handlerWithPriority: InputHandler = {
        handleInput: jest.fn().mockReturnValue(true),
        canHandle: jest.fn().mockReturnValue(true),
        priority: 1,
      };

      registry.registerInputHandler(handlerWithoutPriority);
      registry.registerInputHandler(handlerWithPriority);

      const context = createMockContext();
      registry.handleInput("test", {}, context);

      // Handler with priority 1 should be called first
      expect(handlerWithPriority.handleInput).toHaveBeenCalled();
      expect(handlerWithoutPriority.handleInput).not.toHaveBeenCalled();
    });
  });

  describe("getCommand", () => {
    it("should retrieve registered commands", () => {
      const mockCommand = createMockCommand();
      registry.registerCommand("test", mockCommand);

      const retrievedCommand = registry.getCommand("test");
      expect(retrievedCommand).toBe(mockCommand);
    });

    it("should handle case-insensitive retrieval", () => {
      const mockCommand = createMockCommand();
      registry.registerCommand("Test", mockCommand);

      const retrievedCommand = registry.getCommand("test");
      expect(retrievedCommand).toBe(mockCommand);
    });

    it("should return undefined for non-existent commands", () => {
      const retrievedCommand = registry.getCommand("nonexistent");
      expect(retrievedCommand).toBeUndefined();
    });

    it("should retrieve commands by alias", () => {
      const mockCommand = createMockCommand({
        aliases: ["t", "testing"],
      });
      registry.registerCommand("test", mockCommand);

      expect(registry.getCommand("t")).toBe(mockCommand);
      expect(registry.getCommand("testing")).toBe(mockCommand);
    });
  });

  describe("getAllCommands", () => {
    it("should return all registered commands", () => {
      const command1 = createMockCommand();
      const command2 = createMockCommand();

      registry.registerCommand("cmd1", command1);
      registry.registerCommand("cmd2", command2);

      const allCommands = registry.getAllCommands();
      expect(allCommands.get("cmd1")).toBe(command1);
      expect(allCommands.get("cmd2")).toBe(command2);
    });

    it("should include aliases in command map", () => {
      const mockCommand = createMockCommand({
        aliases: ["alias1", "alias2"],
      });
      registry.registerCommand("original", mockCommand);

      const allCommands = registry.getAllCommands();
      expect(allCommands.get("original")).toBe(mockCommand);
      expect(allCommands.get("alias1")).toBe(mockCommand);
      expect(allCommands.get("alias2")).toBe(mockCommand);
    });

    it("should return independent copy of commands map", () => {
      const mockCommand = createMockCommand();
      registry.registerCommand("test", mockCommand);

      const allCommands = registry.getAllCommands();
      allCommands.delete("test"); // Modify returned map

      // Original registry should still have the command
      expect(registry.getCommand("test")).toBe(mockCommand);
    });
  });

  describe("parseCommandLine", () => {
    it("should parse command lines correctly", () => {
      const result = registry.parseCommandLine(":sync myapp");
      expect(result.command).toBe("sync");
      expect(result.args).toEqual(["myapp"]);
    });

    it("should handle commands without arguments", () => {
      const result = registry.parseCommandLine(":help");
      expect(result.command).toBe("help");
      expect(result.args).toEqual([]);
    });

    it("should handle multiple arguments", () => {
      const result = registry.parseCommandLine(":command arg1 arg2 arg3");
      expect(result.command).toBe("command");
      expect(result.args).toEqual(["arg1", "arg2", "arg3"]);
    });

    it("should handle commands with extra whitespace", () => {
      const result = registry.parseCommandLine(":  sync   myapp   ");
      expect(result.command).toBe("sync");
      expect(result.args).toEqual(["myapp"]);
    });

    it("should return empty command for lines not starting with colon", () => {
      const result = registry.parseCommandLine("sync myapp");
      expect(result.command).toBe("");
      expect(result.args).toEqual([]);
    });

    it("should handle empty command lines", () => {
      const result = registry.parseCommandLine(":");
      expect(result.command).toBe("");
      expect(result.args).toEqual([]);
    });

    it("should handle whitespace-only command lines", () => {
      const result = registry.parseCommandLine(":   ");
      expect(result.command).toBe("");
      expect(result.args).toEqual([]);
    });

    it("should convert command names to lowercase", () => {
      const result = registry.parseCommandLine(":SYNC MyApp");
      expect(result.command).toBe("sync");
      expect(result.args).toEqual(["MyApp"]); // args should preserve case
    });

    it("should handle commands with quoted arguments", () => {
      const result = registry.parseCommandLine(':sync "my app" other');
      expect(result.command).toBe("sync");
      // Note: This tests current behavior - in real implementation you might want to handle quotes
      expect(result.args).toEqual(['"my', 'app"', "other"]);
    });
  });
});
