// src/__tests__/commands/ErrorHandlingTests.test.ts

import type { Command, CommandContext } from "../../commands/types";
import { createMockContext, createMockState } from "../test-utils";

// Error-prone command implementations for testing error handling
class DispatchErrorCommand implements Command {
  aliases = [];
  description = "Command that fails on dispatch";

  execute(context: CommandContext): void {
    // This will throw if dispatch is mocked to throw
    context.dispatch({ type: "SET_MODE", payload: "normal" });
  }
}

class StatusLogErrorCommand implements Command {
  aliases = [];
  description = "Command that fails on status log";

  execute(context: CommandContext): void {
    context.statusLog.info("This will fail if statusLog is mocked to throw");
  }
}

class ContextErrorCommand implements Command {
  aliases = [];
  description = "Command that accesses problematic context properties";

  execute(context: CommandContext): void {
    const { state } = context;
    // Access potentially undefined properties
    const selectedApps = state.selections.selectedApps;
    const appCount = selectedApps.size;
    context.statusLog.info(`Found ${appCount} selected apps`);
  }
}

class MemoryLeakCommand implements Command {
  aliases = [];
  description = "Command that could cause memory issues";
  private static instances: MemoryLeakCommand[] = [];

  constructor() {
    MemoryLeakCommand.instances.push(this);
  }

  execute(context: CommandContext): void {
    // Create large objects that might not be cleaned up
    const largeData = new Array(10000).fill(0).map((_, i) => ({
      id: i,
      data: new Array(1000).fill(`data-${i}`).join(""),
    }));

    context.statusLog.info(`Processed ${largeData.length} items`);
  }

  static getInstanceCount(): number {
    return MemoryLeakCommand.instances.length;
  }

  static clearInstances(): void {
    MemoryLeakCommand.instances.length = 0;
  }
}

class AsyncErrorCommand implements Command {
  aliases = [];
  description = "Async command with various error scenarios";

  async execute(_context: CommandContext, errorType?: string): Promise<void> {
    const type = errorType || "timeout";

    switch (type) {
      case "timeout":
        await new Promise((_, reject) => {
          setTimeout(() => reject(new Error("Operation timed out")), 50);
        });
        break;

      case "network":
        await new Promise((_, reject) => {
          setTimeout(() => reject(new Error("Network connection failed")), 10);
        });
        break;

      case "permission":
        throw new Error("Permission denied");

      case "resource":
        throw new Error("Insufficient resources");

      case "interrupt": {
        // Simulate operation that gets interrupted
        let interrupted = false;
        setTimeout(() => {
          interrupted = true;
        }, 25);

        await new Promise((_resolve, reject) => {
          const checkInterrupt = () => {
            if (interrupted) {
              reject(new Error("Operation interrupted"));
            } else {
              setTimeout(checkInterrupt, 5);
            }
          };
          checkInterrupt();
        });
        break;
      }

      default:
        throw new Error(`Unknown error type: ${type}`);
    }
  }
}

class RecursiveCommand implements Command {
  aliases = [];
  description = "Command that might cause infinite recursion";
  private static depth = 0;

  execute(context: CommandContext, maxDepth?: string): void {
    const max = parseInt(maxDepth || "5", 10);
    RecursiveCommand.depth++;

    try {
      if (RecursiveCommand.depth > max) {
        throw new Error(
          `Maximum recursion depth exceeded: ${RecursiveCommand.depth}`,
        );
      }

      context.statusLog.info(`Recursion depth: ${RecursiveCommand.depth}`);

      // Always recurse to test the limit
      this.execute(context, maxDepth);
    } finally {
      RecursiveCommand.depth--;
    }
  }

  static reset(): void {
    RecursiveCommand.depth = 0;
  }
}

class StateCorruptionCommand implements Command {
  aliases = [];
  description = "Command that might corrupt state";

  execute(context: CommandContext): void {
    const { dispatch } = context;

    // Try to corrupt various state properties
    try {
      dispatch({ type: "SET_SELECTED_APPS", payload: null as any });
    } catch (_e1) {
      // If that fails, try corrupting navigation
      try {
        dispatch({ type: "SET_MODE", payload: { invalid: "object" } as any });
      } catch (_e2) {
        // If that fails too, try corrupting server state
        dispatch({ type: "SET_SERVER", payload: "invalid-server" as any });
      }
    }
  }
}

describe("Command Error Handling and Recovery", () => {
  let context: CommandContext;
  let mockDispatch: jest.Mock;
  let mockStatusLog: any;
  let mockCleanupAndExit: jest.Mock;

  beforeEach(() => {
    mockDispatch = jest.fn();
    mockStatusLog = {
      info: jest.fn(),
      warn: jest.fn(),
      error: jest.fn(),
      debug: jest.fn(),
      set: jest.fn(),
      clear: jest.fn(),
    };
    mockCleanupAndExit = jest.fn();

    context = createMockContext({
      dispatch: mockDispatch,
      statusLog: mockStatusLog,
      cleanupAndExit: mockCleanupAndExit,
    });

    // Reset static counters
    MemoryLeakCommand.clearInstances();
    RecursiveCommand.reset();
  });

  describe("dispatch error handling", () => {
    it("should handle dispatch failures gracefully", () => {
      // Arrange
      const command = new DispatchErrorCommand();
      mockDispatch.mockImplementation(() => {
        throw new Error("Dispatch failed");
      });

      // Act & Assert
      expect(() => command.execute(context)).toThrow("Dispatch failed");
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });

    it("should handle partial dispatch failures", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Multi-dispatch command";

        execute(context: CommandContext): void {
          context.dispatch({ type: "SET_MODE", payload: "normal" });
          context.dispatch({ type: "SET_VIEW", payload: "apps" });
          context.dispatch({ type: "SET_SELECTED_IDX", payload: 0 });
        }
      })();

      // Mock to fail on second dispatch
      mockDispatch
        .mockImplementationOnce(() => {
          /* Success */
        })
        .mockImplementationOnce(() => {
          throw new Error("Second dispatch failed");
        });

      // Act & Assert
      expect(() => command.execute(context)).toThrow("Second dispatch failed");
      expect(mockDispatch).toHaveBeenCalledTimes(2);
    });

    it("should handle dispatch with invalid payloads", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Invalid payload command";

        execute(context: CommandContext): void {
          context.dispatch({ type: "SET_MODE", payload: undefined as any });
        }
      })();

      mockDispatch.mockImplementation((action) => {
        if (action.payload === undefined) {
          throw new Error("Invalid payload: undefined");
        }
      });

      // Act & Assert
      expect(() => command.execute(context)).toThrow(
        "Invalid payload: undefined",
      );
    });
  });

  describe("statusLog error handling", () => {
    it("should handle statusLog method failures", () => {
      // Arrange
      const command = new StatusLogErrorCommand();
      mockStatusLog.info.mockImplementation(() => {
        throw new Error("StatusLog info failed");
      });

      // Act & Assert
      expect(() => command.execute(context)).toThrow("StatusLog info failed");
    });

    it("should handle multiple statusLog errors", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Multi-status command";

        execute(context: CommandContext): void {
          context.statusLog.info("First message");
          context.statusLog.warn("Second message");
          context.statusLog.error("Third message");
        }
      })();

      // Mock different methods to fail
      mockStatusLog.info.mockImplementation(() => {
        throw new Error("Info failed");
      });

      // Act & Assert
      expect(() => command.execute(context)).toThrow("Info failed");
      expect(mockStatusLog.info).toHaveBeenCalledWith("First message");
    });

    it("should handle statusLog with corrupted context", () => {
      // Arrange
      const command = new StatusLogErrorCommand();
      const corruptedContext = {
        ...context,
        statusLog: null as any,
      };

      // Act & Assert
      expect(() => command.execute(corruptedContext)).toThrow();
    });
  });

  describe("context and state error handling", () => {
    it("should handle missing context properties gracefully", () => {
      // Arrange
      const command = new ContextErrorCommand();
      const incompleteContext = {
        state: {
          selections: undefined as any,
        },
      } as CommandContext;

      // Act & Assert
      expect(() => command.execute(incompleteContext)).toThrow();
    });

    it("should handle corrupted state properties", () => {
      // Arrange
      const command = new ContextErrorCommand();
      const corruptedContext = createMockContext({
        state: createMockState({
          selections: {
            selectedApps: null as any,
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
      });

      // Act & Assert
      expect(() => command.execute(corruptedContext)).toThrow();
    });

    it("should handle state modifications during error conditions", () => {
      // Arrange
      const command = new StateCorruptionCommand();
      let dispatchCallCount = 0;

      mockDispatch.mockImplementation((_action) => {
        dispatchCallCount++;
        if (dispatchCallCount === 1) {
          throw new Error("First dispatch failed");
        }
        if (dispatchCallCount === 2) {
          throw new Error("Second dispatch failed");
        }
        // Third dispatch succeeds
      });

      // Act & Assert
      expect(() => command.execute(context)).not.toThrow();
      expect(mockDispatch).toHaveBeenCalledTimes(3);
    });
  });

  describe("async command error scenarios", () => {
    it("should handle timeout errors", async () => {
      // Arrange
      const command = new AsyncErrorCommand();

      // Act & Assert
      await expect(command.execute(context, "timeout")).rejects.toThrow(
        "Operation timed out",
      );
    });

    it("should handle network errors", async () => {
      // Arrange
      const command = new AsyncErrorCommand();

      // Act & Assert
      await expect(command.execute(context, "network")).rejects.toThrow(
        "Network connection failed",
      );
    });

    it("should handle permission errors", async () => {
      // Arrange
      const command = new AsyncErrorCommand();

      // Act & Assert
      await expect(command.execute(context, "permission")).rejects.toThrow(
        "Permission denied",
      );
    });

    it("should handle resource exhaustion", async () => {
      // Arrange
      const command = new AsyncErrorCommand();

      // Act & Assert
      await expect(command.execute(context, "resource")).rejects.toThrow(
        "Insufficient resources",
      );
    });

    it("should handle operation interruption", async () => {
      // Arrange
      const command = new AsyncErrorCommand();

      // Act & Assert
      await expect(command.execute(context, "interrupt")).rejects.toThrow(
        "Operation interrupted",
      );
    });

    it("should handle concurrent async errors", async () => {
      // Arrange
      const command = new AsyncErrorCommand();
      const promises = [
        command.execute(context, "timeout").catch((e) => e.message),
        command.execute(context, "network").catch((e) => e.message),
        command.execute(context, "permission").catch((e) => e.message),
      ];

      // Act
      const results = await Promise.all(promises);

      // Assert
      expect(results).toContain("Operation timed out");
      expect(results).toContain("Network connection failed");
      expect(results).toContain("Permission denied");
    });
  });

  describe("memory and resource management", () => {
    it("should handle potential memory leaks", () => {
      // Arrange
      const initialInstanceCount = MemoryLeakCommand.getInstanceCount();
      const commands = Array.from(
        { length: 10 },
        () => new MemoryLeakCommand(),
      );

      // Act
      commands.forEach((cmd) => {
        try {
          cmd.execute(context);
        } catch (_e) {
          // Ignore execution errors, focus on memory
        }
      });

      const finalInstanceCount = MemoryLeakCommand.getInstanceCount();

      // Assert
      expect(finalInstanceCount).toBe(initialInstanceCount + 10);
      expect(mockStatusLog.info).toHaveBeenCalledTimes(10);
    });

    it("should handle resource cleanup on errors", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Resource cleanup command";

        execute(context: CommandContext): void {
          const resources: any[] = [];

          try {
            // Allocate resources
            for (let i = 0; i < 5; i++) {
              resources.push({ id: i, data: new Array(1000).fill(i) });
            }

            // Simulate error
            throw new Error("Resource allocation failed");
          } finally {
            // Cleanup resources
            resources.forEach((resource) => {
              resource.data = null;
            });
            resources.length = 0;
            context.statusLog.debug("Resources cleaned up");
          }
        }
      })();

      // Act & Assert
      expect(() => command.execute(context)).toThrow(
        "Resource allocation failed",
      );
      expect(mockStatusLog.debug).toHaveBeenCalledWith("Resources cleaned up");
    });
  });

  describe("recursion and stack overflow protection", () => {
    it("should handle controlled recursion", () => {
      // Arrange - Use a command that actually limits recursion
      const controlledCommand = new (class implements Command {
        aliases = [];
        description = "Controlled recursion command";
        private depth = 0;

        execute(context: CommandContext, maxDepth?: string): void {
          const max = parseInt(maxDepth || "5", 10);
          this.depth++;

          try {
            context.statusLog.info(`Controlled depth: ${this.depth}`);

            // Only recurse if within limit
            if (this.depth < max) {
              this.execute(context, maxDepth);
            }
          } finally {
            this.depth--;
          }
        }
      })();

      // Act & Assert - Should complete within depth limit
      expect(() => controlledCommand.execute(context, "3")).not.toThrow();
      expect(mockStatusLog.info).toHaveBeenCalledTimes(3);
    });

    it("should prevent excessive recursion", () => {
      // Arrange
      const command = new RecursiveCommand();

      // Act & Assert
      expect(() => command.execute(context, "15")).toThrow(
        "Maximum recursion depth exceeded",
      );
    });

    it("should handle recursive state cleanup", () => {
      // Arrange
      const command = new (class extends RecursiveCommand {
        execute(context: CommandContext, maxDepth?: string): void {
          try {
            super.execute(context, maxDepth);
          } catch (error) {
            context.statusLog.error(`Recursion error: ${error}`, "recursion");
            throw error;
          }
        }
      })();

      // Act & Assert
      expect(() => command.execute(context, "15")).toThrow(
        "Maximum recursion depth exceeded",
      );

      expect(mockStatusLog.error).toHaveBeenCalledWith(
        expect.stringContaining("Recursion error:"),
        "recursion",
      );
    });
  });

  describe("error recovery and resilience", () => {
    it("should maintain system stability after command failures", () => {
      // Arrange
      const failingCommand = new AsyncErrorCommand();
      const workingCommand = new (class implements Command {
        aliases = [];
        description = "Working command";

        execute(context: CommandContext): void {
          context.statusLog.info("This command works fine");
        }
      })();

      // Act - Execute failing command first
      expect(async () => {
        await failingCommand.execute(context, "permission");
      }).rejects.toThrow();

      // Then execute working command
      expect(() => workingCommand.execute(context)).not.toThrow();

      // Assert
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "This command works fine",
      );
    });

    it("should handle error cascade prevention", async () => {
      // Arrange
      const cascadingCommand = new (class implements Command {
        aliases = [];
        description = "Command that might cause cascading errors";

        async execute(context: CommandContext): Promise<void> {
          const errors: Error[] = [];

          // Try multiple operations that might fail
          const operations = [
            () => context.dispatch({ type: "SET_MODE", payload: "normal" }),
            () => context.statusLog.info("Test message"),
            () => context.dispatch({ type: "SET_VIEW", payload: "apps" }),
            () => context.statusLog.warn("Test warning"),
          ];

          for (const operation of operations) {
            try {
              operation();
            } catch (error) {
              errors.push(error as Error);
              // Continue with other operations despite errors
            }
          }

          if (errors.length > 0) {
            context.statusLog.error(
              `${errors.length} operations failed`,
              "cascade",
            );
          } else {
            context.statusLog.info("All operations succeeded", "cascade");
          }
        }
      })();

      // Mock some operations to fail
      mockDispatch
        .mockImplementationOnce(() => {
          throw new Error("First dispatch failed");
        })
        .mockImplementationOnce(() => {
          /* Second succeeds */
        });

      // Act
      await cascadingCommand.execute(context);

      // Assert - Should complete despite partial failures
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "1 operations failed",
        "cascade",
      );
    });

    it("should handle error state recovery", () => {
      // Arrange
      const recoveryCommand = new (class implements Command {
        aliases = [];
        description = "Command with recovery mechanism";

        execute(context: CommandContext): void {
          const originalMode = context.state.mode;

          try {
            context.dispatch({ type: "SET_MODE", payload: "normal" });

            // Simulate operation that might fail
            throw new Error("Operation failed");
          } catch (_error) {
            // Attempt recovery
            try {
              context.dispatch({ type: "SET_MODE", payload: originalMode });
              context.statusLog.warn("Recovered from error", "recovery");
            } catch (recoveryError) {
              context.statusLog.error("Recovery failed", "recovery");
              throw recoveryError;
            }
          }
        }
      })();

      // Act & Assert - Should not throw due to recovery
      expect(() => recoveryCommand.execute(context)).not.toThrow();
      expect(mockStatusLog.warn).toHaveBeenCalledWith(
        "Recovered from error",
        "recovery",
      );
      expect(mockDispatch).toHaveBeenCalledTimes(2); // Original + recovery
    });
  });

  describe("boundary condition error handling", () => {
    it("should handle null/undefined command arguments", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Null-safe command";

        execute(context: CommandContext, arg1?: string, arg2?: string): void {
          const safeArg1 = arg1 || "default1";
          const safeArg2 = arg2 || "default2";

          context.statusLog.info(`Args: ${safeArg1}, ${safeArg2}`);
        }
      })();

      // Act & Assert
      expect(() =>
        command.execute(context, undefined, null as any),
      ).not.toThrow();
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Args: default1, default2",
      );
    });

    it("should handle extremely long command arguments", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Long argument handler";

        execute(context: CommandContext, longArg?: string): void {
          if (longArg && longArg.length > 1000) {
            throw new Error("Argument too long");
          }
          context.statusLog.info(`Arg length: ${longArg?.length || 0}`);
        }
      })();

      const veryLongArg = "a".repeat(2000);

      // Act & Assert
      expect(() => command.execute(context, veryLongArg)).toThrow(
        "Argument too long",
      );
    });

    it("should handle special characters in arguments", () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Special character handler";

        execute(context: CommandContext, specialArg?: string): void {
          // Handle various special characters
          const sanitized = specialArg?.replace(/[<>"'&]/g, "") || "";
          context.statusLog.info(`Sanitized: ${sanitized}`);
        }
      })();

      const specialCharsArg = '<script>alert("test")</script>&';

      // Act & Assert
      expect(() => command.execute(context, specialCharsArg)).not.toThrow();
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Sanitized: scriptalert(test)/script",
      );
    });
  });
});
