import { beforeEach, describe, expect, mock, spyOn, test } from "bun:test";
import type { CrashDetectorOptions } from "../../utils/crash-detector";
import { runWithCrashDetection } from "../../utils/crash-detector";

// Mock child_process.spawn
const mockSpawn = mock();
const mockChild = {
  on: mock(),
};

mock.module("node:child_process", () => ({
  spawn: mockSpawn,
}));

describe("runWithCrashDetection", () => {
  let mockOnCrash: ReturnType<typeof mock>;
  let mockConsoleError: ReturnType<typeof mock>;

  beforeEach(() => {
    mockOnCrash = mock();
    mockChild.on.mockReset();
    mockSpawn.mockReset();

    // Mock console.error to prevent test output noise
    mockConsoleError = spyOn(console, "error").mockImplementation(() => {});

    // Default spawn mock setup
    mockSpawn.mockReturnValue(mockChild);
    mockChild.on.mockImplementation(
      (event: string, handler: (...args: any[]) => void) => {
        // Store handlers for later invocation
        (mockChild as any)[`_${event}Handler`] = handler;
      },
    );
  });

  describe("successful execution", () => {
    test("should resolve on successful exit", async () => {
      const options: CrashDetectorOptions = {
        command: "echo",
        args: ["hello"],
      };

      const promise = runWithCrashDetection(options);

      // Simulate successful exit
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(0, null); // Exit code 0, no signal
      }

      await expect(promise).resolves.toBeUndefined();

      expect(mockSpawn).toHaveBeenCalledWith("echo", ["hello"], {
        stdio: "inherit",
      });
    });

    test("should resolve on SIGINT termination", async () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: [],
      };

      const promise = runWithCrashDetection(options);

      // Simulate SIGINT termination (normal)
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(null, "SIGINT");
      }

      await expect(promise).resolves.toBeUndefined();
      expect(mockConsoleError).not.toHaveBeenCalled();
    });

    test("should resolve on SIGTERM termination", async () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: [],
      };

      const promise = runWithCrashDetection(options);

      // Simulate SIGTERM termination (normal)
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(null, "SIGTERM");
      }

      await expect(promise).resolves.toBeUndefined();
      expect(mockConsoleError).not.toHaveBeenCalled();
    });
  });

  describe("crash detection", () => {
    test("should detect crash on non-zero exit code", async () => {
      const options: CrashDetectorOptions = {
        command: "failing-command",
        args: ["--fail"],
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash with exit code 1
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(1, null);
      }

      await expect(promise).rejects.toThrow("Process crashed with code 1");

      expect(mockConsoleError).toHaveBeenCalledWith(
        "💥 Process crashed with code 1",
      );
    });

    test("should detect crash on unexpected signal", async () => {
      const options: CrashDetectorOptions = {
        command: "killed-command",
        args: [],
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash with SIGKILL
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(null, "SIGKILL");
      }

      await expect(promise).rejects.toThrow(
        "Process crashed with signal SIGKILL",
      );

      expect(mockConsoleError).toHaveBeenCalledWith(
        "💥 Process crashed with signal SIGKILL",
      );
    });

    test("should call onCrash callback when provided", async () => {
      const options: CrashDetectorOptions = {
        command: "failing-command",
        args: [],
        onCrash: mockOnCrash,
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(1, null);
      }

      await expect(promise).rejects.toThrow();
      expect(mockOnCrash).toHaveBeenCalled();
    });

    test("should handle async onCrash callback", async () => {
      const asyncOnCrash = mock().mockResolvedValue(undefined);
      const options: CrashDetectorOptions = {
        command: "failing-command",
        args: [],
        onCrash: asyncOnCrash,
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(1, null);
      }

      await expect(promise).rejects.toThrow();
      expect(asyncOnCrash).toHaveBeenCalled();
    });

    test("should handle onCrash callback error", async () => {
      const failingOnCrash = mock().mockRejectedValue(
        new Error("Callback failed"),
      );
      const options: CrashDetectorOptions = {
        command: "failing-command",
        args: [],
        onCrash: failingOnCrash,
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(1, null);
      }

      await expect(promise).rejects.toThrow();
      expect(mockConsoleError).toHaveBeenCalledWith(
        "Error in crash handler:",
        expect.any(Error),
      );
    });
  });

  describe("log showing behavior", () => {
    test("should not show logs when showLogsOnCrash is false", async () => {
      const options: CrashDetectorOptions = {
        command: "failing-command",
        args: [],
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(1, null);
      }

      await expect(promise).rejects.toThrow();
      
      // Should not attempt to show logs
      expect(mockConsoleError).toHaveBeenCalledWith(
        "💥 Process crashed with code 1",
      );
      expect(mockConsoleError).not.toHaveBeenCalledWith(
        "💡 Opening logs to help diagnose the crash...\n",
      );
    });

    test("should attempt to show logs on crash by default", async () => {
      const options: CrashDetectorOptions = {
        command: "failing-command",
        args: [],
        // showLogsOnCrash defaults to true
      };

      const promise = runWithCrashDetection(options);

      // Simulate crash
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(1, null);
      }

      await expect(promise).rejects.toThrow();

      expect(mockConsoleError).toHaveBeenCalledWith(
        "💡 Opening logs to help diagnose the crash...\n",
      );
    });
  });

  describe("process error handling", () => {
    test("should handle spawn error", async () => {
      const spawnError = new Error("Command not found");
      const options: CrashDetectorOptions = {
        command: "nonexistent-command",
        args: [],
      };

      const promise = runWithCrashDetection(options);

      // Simulate spawn error
      const errorHandler = (mockChild as any)._errorHandler;
      if (errorHandler) {
        errorHandler(spawnError);
      }

      await expect(promise).rejects.toThrow("Command not found");

      expect(mockConsoleError).toHaveBeenCalledWith(
        "💥 Failed to start process:",
        "Command not found",
      );
    });
  });

  describe("edge cases", () => {
    test("should handle null exit code and signal", async () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: [],
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Simulate exit with null code and null signal (unusual but possible)
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(null, null);
      }

      await expect(promise).rejects.toThrow("Process crashed with signal null");
    });

    test("should detect crash when exit code 0 has unexpected signal", async () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: [],
        showLogsOnCrash: false,
      };

      const promise = runWithCrashDetection(options);

      // Exit code 0 but with unexpected signal should be treated as crash
      const exitHandler = (mockChild as any)._exitHandler;
      if (exitHandler) {
        exitHandler(0, "SIGUSR1");
      }

      // The implementation prioritizes code over signal in error message
      await expect(promise).rejects.toThrow("Process crashed with code 0");
    });

    test("should properly set up stdio inheritance", () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: ["arg1", "arg2"],
      };

      runWithCrashDetection(options);

      expect(mockSpawn).toHaveBeenCalledWith("test-command", ["arg1", "arg2"], {
        stdio: "inherit",
      });
    });

    test("should register both exit and error handlers", () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: [],
      };

      runWithCrashDetection(options);

      expect(mockChild.on).toHaveBeenCalledWith("exit", expect.any(Function));
      expect(mockChild.on).toHaveBeenCalledWith("error", expect.any(Function));
    });

    test("should use default showLogsOnCrash value", () => {
      const options: CrashDetectorOptions = {
        command: "test-command",
        args: [],
        // No showLogsOnCrash specified, should default to true
      };

      // Test that the function doesn't throw on setup
      expect(() => runWithCrashDetection(options)).not.toThrow();
    });
  });
});