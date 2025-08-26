// src/__tests__/commands/PerformanceTests.test.ts
import { mock } from "bun:test";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

import type { Command, CommandContext } from "../../commands/types";
import { createMockContext, createMockState } from "../test-utils";

// Performance test implementations
class BenchmarkCommand implements Command {
  aliases = [];
  description = "Command for performance benchmarking";

  execute(context: CommandContext, iterations?: string): void {
    const count = parseInt(iterations || "100", 10);
    const startTime = performance.now();

    for (let i = 0; i < count; i++) {
      context.dispatch({ type: "SET_SELECTED_IDX", payload: i % 10 });
    }

    const endTime = performance.now();
    const duration = endTime - startTime;

    context.statusLog.info(
      `Completed ${count} operations in ${duration.toFixed(2)}ms`,
      "benchmark",
    );
  }
}

class AsyncBenchmarkCommand implements Command {
  aliases = [];
  description = "Async command for performance testing";

  async execute(context: CommandContext, delay?: string): Promise<void> {
    const delayMs = parseInt(delay || "10", 10);
    const startTime = performance.now();

    // Simulate async operation
    await new Promise((resolve) => setTimeout(resolve, delayMs));

    const endTime = performance.now();
    const actualDuration = endTime - startTime;

    context.statusLog.info(
      `Async operation took ${actualDuration.toFixed(2)}ms (expected ~${delayMs}ms)`,
      "async-benchmark",
    );
  }
}

class MemoryIntensiveCommand implements Command {
  aliases = [];
  description = "Command that uses significant memory";

  execute(context: CommandContext, size?: string): void {
    const arraySize = parseInt(size || "10000", 10);
    const startMemory = this.getMemoryUsage();

    // Create large data structure
    const largeArray = new Array(arraySize).fill(0).map((_, i) => ({
      id: i,
      data: new Array(100).fill(`item-${i}`),
      timestamp: Date.now(),
    }));

    const endMemory = this.getMemoryUsage();
    const memoryDelta = endMemory - startMemory;

    context.statusLog.info(
      `Created array of ${arraySize} items, memory delta: ${memoryDelta.toFixed(2)}MB`,
      "memory-test",
    );

    // Clean up immediately
    largeArray.length = 0;
  }

  private getMemoryUsage(): number {
    if (typeof process !== "undefined" && process.memoryUsage) {
      return process.memoryUsage().heapUsed / 1024 / 1024; // MB
    }
    return 0; // Fallback for browser environments
  }
}

class ConcurrentCommand implements Command {
  aliases = [];
  description = "Command that can run concurrently";
  private static activeInstances = 0;

  async execute(context: CommandContext, delay?: string): Promise<void> {
    ConcurrentCommand.activeInstances++;
    const instanceId = ConcurrentCommand.activeInstances;
    const delayMs = parseInt(delay || "50", 10);

    try {
      context.statusLog.info(
        `Instance ${instanceId} starting (${ConcurrentCommand.activeInstances} active)`,
        "concurrent",
      );

      await new Promise((resolve) => setTimeout(resolve, delayMs));

      context.statusLog.info(`Instance ${instanceId} completed`, "concurrent");
    } finally {
      ConcurrentCommand.activeInstances--;
    }
  }

  static getActiveCount(): number {
    return ConcurrentCommand.activeInstances;
  }

  static reset(): void {
    ConcurrentCommand.activeInstances = 0;
  }
}

class TimeoutTestCommand implements Command {
  aliases = [];
  description = "Command for timeout testing";

  async execute(context: CommandContext, timeout?: string): Promise<void> {
    const timeoutMs = parseInt(timeout || "100", 10);

    const timeoutPromise = new Promise<never>((_, reject) => {
      setTimeout(
        () => reject(new Error(`Operation timed out after ${timeoutMs}ms`)),
        timeoutMs,
      );
    });

    const operationPromise = new Promise<void>((resolve) => {
      // Simulate operation that takes longer than timeout
      setTimeout(resolve, timeoutMs + 50);
    });

    try {
      await Promise.race([operationPromise, timeoutPromise]);
      context.statusLog.info(
        "Operation completed within timeout",
        "timeout-test",
      );
    } catch (error: any) {
      context.statusLog.error(error.message, "timeout-test");
      throw error;
    }
  }
}

class ResourceLeakCommand implements Command {
  aliases = [];
  description = "Command that might leak resources";
  private static resources: any[] = [];

  execute(context: CommandContext): void {
    // Simulate resource allocation
    const resource = {
      id: Math.random(),
      data: new Array(1000).fill(0),
      cleanup: () => {
        const index = ResourceLeakCommand.resources.indexOf(resource);
        if (index > -1) {
          ResourceLeakCommand.resources.splice(index, 1);
        }
      },
    };

    ResourceLeakCommand.resources.push(resource);

    context.statusLog.info(
      `Allocated resource ${resource.id}, total: ${ResourceLeakCommand.resources.length}`,
      "resource-test",
    );

    // Intentionally don't clean up to test leak detection
  }

  static getResourceCount(): number {
    return ResourceLeakCommand.resources.length;
  }

  static cleanup(): void {
    ResourceLeakCommand.resources.forEach((resource) => {
      resource.cleanup?.();
    });
    ResourceLeakCommand.resources.length = 0;
  }
}

class CPUIntensiveCommand implements Command {
  aliases = [];
  description = "CPU intensive command for performance testing";

  execute(context: CommandContext, complexity?: string): void {
    const iterations = parseInt(complexity || "100000", 10);
    const startTime = performance.now();

    // Simulate CPU intensive operation
    let result = 0;
    for (let i = 0; i < iterations; i++) {
      result += Math.sqrt(i) * Math.sin(i) * Math.cos(i);
    }

    const endTime = performance.now();
    const duration = endTime - startTime;

    context.statusLog.info(
      `CPU intensive operation (${iterations} iterations) took ${duration.toFixed(2)}ms, result: ${result.toFixed(2)}`,
      "cpu-test",
    );
  }
}

describe("Command Performance and Timeout Tests", () => {
  let context: CommandContext;
  let mockDispatch: mock.Mock;
  let mockStatusLog: any;

  beforeEach(() => {
    mockDispatch = mock();
    mockStatusLog = {
      info: mock(),
      warn: mock(),
      error: mock(),
      debug: mock(),
      set: mock(),
      clear: mock(),
    };

    context = createMockContext({
      dispatch: mockDispatch,
      statusLog: mockStatusLog,
    });

    // Reset static state
    ConcurrentCommand.reset();
    ResourceLeakCommand.cleanup();

    // Increase timeout for performance tests
    
  });

  afterEach(() => {
    
  });

  describe("synchronous performance tests", () => {
    it("should complete fast operations efficiently", () => {
      // Arrange
      const command = new BenchmarkCommand();
      const startTime = performance.now();

      // Act
      command.execute(context, "1000");

      const endTime = performance.now();
      const totalTime = endTime - startTime;

      // Assert
      expect(totalTime).toBeLessThan(100); // Should complete in under 100ms
      expect(mockDispatch).toHaveBeenCalledTimes(1000);
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        expect.stringContaining("Completed 1000 operations"),
        "benchmark",
      );
    });

    it("should handle varying workload sizes efficiently", () => {
      // Arrange
      const command = new BenchmarkCommand();
      const sizes = [10, 100, 1000];
      const results: number[] = [];

      // Act
      sizes.forEach((size, _index) => {
        mockDispatch.mockClear();
        const startTime = performance.now();

        command.execute(context, size.toString());

        const endTime = performance.now();
        results.push(endTime - startTime);

        expect(mockDispatch).toHaveBeenCalledTimes(size);
      });

      // Assert - Performance should scale reasonably (but may vary due to Node.js performance)
      expect(results[2]).toBeLessThan(200); // Even 1000 iterations should be < 200ms
      // Note: Individual timing variations are expected due to JS engine optimizations
    });

    it("should maintain performance with large state objects", () => {
      // Arrange
      const largeApps = Array.from({ length: 10000 }, (_, i) => ({
        name: `app-${i}`,
        sync: "Synced",
        health: "Healthy",
        clusterId: `cluster-${i % 10}`,
        clusterLabel: `cluster-${i % 10}`,
        namespace: `namespace-${i % 100}`,
        appNamespace: "argocd",
        project: `project-${i % 50}`,
        lastSyncAt: "2023-12-01T10:00:00Z",
      }));

      const largeContext = createMockContext({
        state: createMockState({ apps: largeApps }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      const command = new BenchmarkCommand();
      const startTime = performance.now();

      // Act
      command.execute(largeContext, "100");

      const endTime = performance.now();
      const duration = endTime - startTime;

      // Assert - Should still be fast even with large state
      expect(duration).toBeLessThan(20);
      expect(mockDispatch).toHaveBeenCalledTimes(100);
    });
  });

  describe("asynchronous performance tests", () => {
    it("should handle async operations with predictable timing", async () => {
      // Arrange
      const command = new AsyncBenchmarkCommand();
      const expectedDelay = 25;

      const startTime = performance.now();

      // Act
      await command.execute(context, expectedDelay.toString());

      const endTime = performance.now();
      const actualDelay = endTime - startTime;

      // Assert - Should be close to expected delay (within 10ms tolerance)
      expect(actualDelay).toBeGreaterThanOrEqual(expectedDelay * 0.9);
      expect(actualDelay).toBeLessThan(expectedDelay + 50); // More tolerance for CI environments
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        expect.stringContaining(`expected ~${expectedDelay}ms`),
        "async-benchmark",
      );
    });

    it("should handle multiple concurrent async operations", async () => {
      // Arrange
      const command = new ConcurrentCommand();
      const concurrency = 5;
      const delay = 30;

      const startTime = performance.now();

      // Act - Start multiple concurrent operations
      const promises = Array.from({ length: concurrency }, (_, _i) =>
        command.execute(context, delay.toString()),
      );

      await Promise.all(promises);

      const endTime = performance.now();
      const totalTime = endTime - startTime;

      // Assert - Should complete concurrently, not sequentially
      expect(totalTime).toBeLessThan(delay * concurrency); // Much faster than sequential
      expect(totalTime).toBeGreaterThanOrEqual(delay * 0.9); // But at least one delay period, with some leniency for CI timing
      expect(mockStatusLog.info).toHaveBeenCalledTimes(concurrency * 2); // Start + complete for each
      expect(ConcurrentCommand.getActiveCount()).toBe(0); // All should be done
    });

    it("should handle rapid sequential async operations", async () => {
      // Arrange
      const command = new AsyncBenchmarkCommand();
      const operations = 10;
      const delay = 5;

      const startTime = performance.now();

      // Act
      for (let i = 0; i < operations; i++) {
        await command.execute(context, delay.toString());
      }

      const endTime = performance.now();
      const totalTime = endTime - startTime;

      // Assert
      expect(totalTime).toBeGreaterThanOrEqual(delay * operations);
      expect(totalTime).toBeLessThan(delay * operations + 200); // Allow even more overhead for CI
      expect(mockStatusLog.info).toHaveBeenCalledTimes(operations);
    });
  });

  describe("memory performance tests", () => {
    it("should handle memory-intensive operations efficiently", () => {
      // Arrange
      const command = new MemoryIntensiveCommand();
      const _initialMemory =
        typeof process !== "undefined" ? process.memoryUsage().heapUsed : 0;

      // Act
      command.execute(context, "5000");

      // Assert
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        expect.stringContaining("Created array of 5000 items"),
        "memory-test",
      );

      // Memory should be cleaned up after command
      if (typeof global !== "undefined" && global.gc) {
        global.gc(); // Force garbage collection if available
      }
    });

    it("should detect potential memory leaks", () => {
      // Arrange
      const command = new ResourceLeakCommand();
      const initialCount = ResourceLeakCommand.getResourceCount();

      // Act - Allocate multiple resources
      for (let i = 0; i < 5; i++) {
        command.execute(context);
      }

      const finalCount = ResourceLeakCommand.getResourceCount();

      // Assert - Resources should accumulate (indicating potential leak)
      expect(finalCount).toBe(initialCount + 5);
      expect(mockStatusLog.info).toHaveBeenCalledTimes(5);

      // Cleanup
      ResourceLeakCommand.cleanup();
      expect(ResourceLeakCommand.getResourceCount()).toBe(0);
    });

    it("should handle memory pressure gracefully", () => {
      // Arrange
      const command = new MemoryIntensiveCommand();

      // Act - Try to allocate very large array
      const startTime = performance.now();

      try {
        command.execute(context, "1000000"); // Very large allocation
      } catch (error) {
        // Handle potential out-of-memory errors
        expect(error).toBeDefined();
      }

      const endTime = performance.now();

      // Assert - Should complete or fail quickly, not hang
      expect(endTime - startTime).toBeLessThan(5000); // More reasonable for large allocations
    });
  });

  describe("timeout and cancellation tests", () => {
    it("should handle operation timeouts correctly", async () => {
      // Arrange
      const command = new TimeoutTestCommand();

      // Act & Assert
      await expect(command.execute(context, "50")).rejects.toThrow(
        "Operation timed out after 50ms",
      );

      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Operation timed out after 50ms",
        "timeout-test",
      );
    });

    it("should handle operations that complete within timeout", async () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Fast operation command";

        async execute(context: CommandContext): Promise<void> {
          const timeoutMs = 100;

          const timeoutPromise = new Promise<never>((_, reject) => {
            setTimeout(() => reject(new Error("Timed out")), timeoutMs);
          });

          const quickOperation = new Promise<void>((resolve) => {
            setTimeout(resolve, 20); // Completes well before timeout
          });

          await Promise.race([quickOperation, timeoutPromise]);
          context.statusLog.info("Operation completed quickly", "timeout-test");
        }
      })();

      // Act
      await command.execute(context);

      // Assert
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Operation completed quickly",
        "timeout-test",
      );
    });

    it("should handle timeout edge cases", async () => {
      // Arrange
      const command = new (class implements Command {
        aliases = [];
        description = "Edge case timeout command";

        async execute(context: CommandContext): Promise<void> {
          const promises = [
            // Operation that completes exactly at timeout
            new Promise<string>((resolve) => {
              setTimeout(() => resolve("completed"), 50);
            }),
            // Timeout that triggers at same time
            new Promise<never>((_, reject) => {
              setTimeout(() => reject(new Error("timeout")), 50);
            }),
          ];

          try {
            const result = await Promise.race(promises);
            context.statusLog.info(`Result: ${result}`, "timeout-edge");
          } catch (error: any) {
            context.statusLog.error(error.message, "timeout-edge");
            throw error;
          }
        }
      })();

      // Act & Assert - Either outcome is acceptable for edge case
      try {
        await command.execute(context);
        expect(mockStatusLog.info).toHaveBeenCalledWith(
          "Result: completed",
          "timeout-edge",
        );
      } catch (error: any) {
        expect(error.message).toBe("timeout");
        expect(mockStatusLog.error).toHaveBeenCalledWith(
          "timeout",
          "timeout-edge",
        );
      }
    });
  });

  describe("CPU performance tests", () => {
    it("should handle CPU-intensive operations within reasonable time", () => {
      // Arrange
      const command = new CPUIntensiveCommand();
      const startTime = performance.now();

      // Act
      command.execute(context, "50000");

      const endTime = performance.now();
      const duration = endTime - startTime;

      // Assert
      expect(duration).toBeLessThan(1000); // Should complete within 1 second
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        expect.stringContaining("CPU intensive operation (50000 iterations)"),
        "cpu-test",
      );
    });

    it("should scale CPU performance predictably", () => {
      // Arrange
      const command = new CPUIntensiveCommand();
      const complexities = [1000, 10000, 100000];
      const durations: number[] = [];

      // Act
      complexities.forEach((complexity) => {
        mockStatusLog.info.mockClear();
        const startTime = performance.now();

        command.execute(context, complexity.toString());

        const endTime = performance.now();
        durations.push(endTime - startTime);
      });

      // Assert - Duration should increase with complexity
      expect(durations[0]).toBeLessThan(durations[1]);
      expect(durations[1]).toBeLessThan(durations[2]);
      expect(durations[2]).toBeLessThan(500); // Even heavy workload should be reasonable
    });
  });

  describe("stress tests", () => {
    it("should handle high-frequency command execution", () => {
      // Arrange
      const command = new BenchmarkCommand();
      const executions = 100;
      const startTime = performance.now();

      // Act - Execute same command many times rapidly
      for (let i = 0; i < executions; i++) {
        command.execute(context, "10");
      }

      const endTime = performance.now();
      const totalTime = endTime - startTime;

      // Assert
      expect(totalTime).toBeLessThan(500); // Should handle rapid execution
      expect(mockDispatch).toHaveBeenCalledTimes(executions * 10); // 10 dispatches per execution
    });

    it("should maintain performance under memory pressure", () => {
      // Arrange
      const allocCommand = new MemoryIntensiveCommand();
      const benchCommand = new BenchmarkCommand();

      // Act - Allocate memory then run performance test
      for (let i = 0; i < 5; i++) {
        allocCommand.execute(context, "1000");
      }

      const startTime = performance.now();
      benchCommand.execute(context, "500");
      const endTime = performance.now();

      const duration = endTime - startTime;

      // Assert - Should still perform reasonably under memory pressure
      expect(duration).toBeLessThan(100);
    });

    it("should handle mixed async and sync workloads", async () => {
      // Arrange
      const asyncCommand = new AsyncBenchmarkCommand();
      const syncCommand = new BenchmarkCommand();

      const startTime = performance.now();

      // Act - Mix async and sync operations
      const asyncPromises = Array.from({ length: 3 }, () =>
        asyncCommand.execute(context, "10"),
      );

      // Run sync commands while async are running
      for (let i = 0; i < 5; i++) {
        syncCommand.execute(context, "50");
      }

      await Promise.all(asyncPromises);

      const endTime = performance.now();
      const totalTime = endTime - startTime;

      // Assert
      expect(totalTime).toBeLessThan(200); // Should complete efficiently
      expect(mockStatusLog.info).toHaveBeenCalledTimes(8); // 3 async + 5 sync
    });
  });
});
