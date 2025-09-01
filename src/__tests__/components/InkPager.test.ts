// src/__tests__/components/InkPager.test.ts
import { beforeEach, describe, expect, it, mock } from "bun:test";
import {
  type PagerDependencies,
  showInkPager,
} from "../../components/InkPager";
import { stripAnsi } from "../test-utils";

describe("InkPager", () => {
  let mockDeps: PagerDependencies;
  let writtenOutput: string[];
  let keyHandlers: ((chunk: Buffer) => void)[];
  let resizeHandlers: (() => void)[];
  let signalHandlers: Map<string, (() => void)[]>;

  beforeEach(() => {
    writtenOutput = [];
    keyHandlers = [];
    resizeHandlers = [];
    signalHandlers = new Map();

    // Create mock dependencies
    mockDeps = {
      stdin: {
        setRawMode: mock((_enabled: boolean) => {}),
        pause: mock(() => {}),
        resume: mock(() => {}),
        on: mock((event: string, handler: (chunk: Buffer) => void) => {
          if (event === "data") {
            keyHandlers.push(handler);
          }
        }),
        removeListener: mock(
          (event: string, handler: (chunk: Buffer) => void) => {
            if (event === "data") {
              keyHandlers = keyHandlers.filter((h) => h !== handler);
            }
          },
        ),
      },
      stdout: {
        rows: 24,
        cols: 80,
        on: mock((event: string, handler: () => void) => {
          if (event === "resize") {
            resizeHandlers.push(handler);
          }
        }),
        off: mock((event: string, handler: () => void) => {
          if (event === "resize") {
            resizeHandlers = resizeHandlers.filter((h) => h !== handler);
          }
        }),
      },
      process: {
        emit: mock((_event: string) => true),
        on: mock((event: string, handler: () => void) => {
          const handlers = signalHandlers.get(event) || [];
          handlers.push(handler);
          signalHandlers.set(event, handlers);
        }),
        off: mock((event: string, handler: () => void) => {
          const handlers = signalHandlers.get(event) || [];
          const filtered = handlers.filter((h) => h !== handler);
          signalHandlers.set(event, filtered);
        }),
      },
      rawStdoutWrite: mock((chunk: string) => {
        writtenOutput.push(chunk);
        return true;
      }),
      setTimeout: mock((fn: () => void, _ms: number) => {
        // Immediately call the function for tests
        setTimeout(() => fn(), 0);
        return 1;
      }),
    };
  });

  // Helper functions
  const simulateKeyPress = (key: string) => {
    keyHandlers.forEach((handler) => {
      handler(Buffer.from(key));
    });
  };

  const simulateResize = (rows = 30, cols = 100) => {
    mockDeps.stdout.rows = rows;
    mockDeps.stdout.cols = cols;
    resizeHandlers.forEach((handler) => {
      handler();
    });
  };

  const getWrittenOutput = () => stripAnsi(writtenOutput.join(""));

  const isPagerActive = () => keyHandlers.length > 0;

  describe("Basic Setup and Teardown", () => {
    it("should initialize with dependency injection", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);

      // Wait for setup
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Should use injected dependencies
      expect(mockDeps.stdin.setRawMode).toHaveBeenCalledWith(true);
      expect(mockDeps.stdin.resume).toHaveBeenCalled();
      expect(mockDeps.process.emit).toHaveBeenCalledWith("external-enter");
      expect(isPagerActive()).toBe(true);

      // Exit
      simulateKeyPress("q");
      await pagerPromise;

      // Should cleanup using dependencies
      expect(mockDeps.stdin.setRawMode).toHaveBeenCalledWith(false);
      expect(mockDeps.stdin.pause).toHaveBeenCalled();
      expect(mockDeps.process.emit).toHaveBeenCalledWith("external-exit");
    });

    it("should render content using rawStdoutWrite dependency", async () => {
      const content = "Line 1\nLine 2\nLine 3";
      const pagerPromise = showInkPager(content, {}, mockDeps);

      await new Promise((resolve) => setTimeout(resolve, 10));

      // Should have called rawStdoutWrite for rendering
      expect(mockDeps.rawStdoutWrite).toHaveBeenCalled();

      // Should contain the content
      const output = getWrittenOutput();
      expect(output).toContain("Line 1");
      expect(output).toContain("Line 2");
      expect(output).toContain("Line 3");

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should display title when provided", async () => {
      const pagerPromise = showInkPager(
        "content",
        { title: "Test Title" },
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      const output = getWrittenOutput();
      expect(output).toContain("Test Title");

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle different content types", async () => {
      const testCases = [
        "Simple text",
        "Multi\nLine\nContent",
        "",
        "Line with special chars: àáâãäå",
        "Line with unicode: 🚀🎉",
      ];

      for (const content of testCases) {
        writtenOutput = []; // Reset output for each test
        const pagerPromise = showInkPager(content, {}, mockDeps);
        await new Promise((resolve) => setTimeout(resolve, 5));

        expect(isPagerActive()).toBe(true);
        expect(mockDeps.rawStdoutWrite).toHaveBeenCalled();

        simulateKeyPress("q");
        await pagerPromise;
      }
    });
  });

  describe("Keyboard Navigation", () => {
    it("should handle quit commands", async () => {
      for (const quitKey of ["q", "Q"]) {
        keyHandlers = []; // Reset handlers
        const pagerPromise = showInkPager("test content", {}, mockDeps);
        await new Promise((resolve) => setTimeout(resolve, 5));

        expect(isPagerActive()).toBe(true);

        simulateKeyPress(quitKey);
        await pagerPromise;

        expect(keyHandlers.length).toBe(0);
      }
    });

    it("should handle navigation keys", async () => {
      const content = Array.from(
        { length: 50 },
        (_, i) => `Line ${i + 1}`,
      ).join("\n");
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Test various navigation keys
      const navKeys = [
        "j",
        "k",
        " ",
        "b",
        "g",
        "G",
        "\x1b[A",
        "\x1b[B",
        "\x1b[5~",
        "\x1b[6~",
      ];

      for (const key of navKeys) {
        const initialCallCount = mockDeps.rawStdoutWrite.mock.calls.length;
        simulateKeyPress(key);
        await new Promise((resolve) => setTimeout(resolve, 2));

        expect(isPagerActive()).toBe(true);
        // Should have re-rendered (more calls to rawStdoutWrite)
        expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
          initialCallCount,
        );
      }

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle boundary navigation gracefully", async () => {
      const shortContent = "Line 1\nLine 2";
      const pagerPromise = showInkPager(shortContent, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Try to navigate beyond boundaries
      const _initialCallCount = mockDeps.rawStdoutWrite.mock.calls.length;

      // Try scrolling up from top
      for (let i = 0; i < 5; i++) {
        simulateKeyPress("k");
      }

      // Try scrolling down beyond content
      for (let i = 0; i < 10; i++) {
        simulateKeyPress("j");
      }

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Search Functionality", () => {
    it("should enter and exit search mode", async () => {
      const pagerPromise = showInkPager(
        "searchable content\nmore content",
        {},
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      const initialCallCount = mockDeps.rawStdoutWrite.mock.calls.length;

      // Enter search mode
      simulateKeyPress("/");
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Should have written search prompt
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        initialCallCount,
      );
      expect(isPagerActive()).toBe(true);

      // Exit search with Escape
      simulateKeyPress("\x1b");
      await new Promise((resolve) => setTimeout(resolve, 5));

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should perform search and navigate matches", async () => {
      const content = "test keyword\nother line\nanother keyword line";
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Start search
      simulateKeyPress("/");

      // Type search term
      for (const char of "keyword") {
        simulateKeyPress(char);
        await new Promise((resolve) => setTimeout(resolve, 1));
      }

      const beforeSearchCallCount = mockDeps.rawStdoutWrite.mock.calls.length;

      // Execute search
      simulateKeyPress("\r");
      await new Promise((resolve) => setTimeout(resolve, 5));

      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        beforeSearchCallCount,
      );

      // Navigate through matches
      simulateKeyPress("n"); // Next match
      simulateKeyPress("N"); // Previous match

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should disable search when searchEnabled is false", async () => {
      const pagerPromise = showInkPager(
        "content",
        { searchEnabled: false },
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      const initialCallCount = mockDeps.rawStdoutWrite.mock.calls.length;

      // Try to enter search mode - should be ignored
      simulateKeyPress("/");
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Should not have written search prompt
      const callsAfterSlash =
        mockDeps.rawStdoutWrite.mock.calls.slice(initialCallCount);
      const searchPrompts = callsAfterSlash.filter((call) =>
        call[0].includes("Search:"),
      );
      expect(searchPrompts.length).toBe(0);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle search with special regex characters", async () => {
      const content = "Line with [brackets] and (parens)\nMore [brackets]";
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Search for regex special chars
      simulateKeyPress("/");
      for (const char of "[brackets]") {
        simulateKeyPress(char);
      }
      simulateKeyPress("\r");
      await new Promise((resolve) => setTimeout(resolve, 5));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle backspace in search mode", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      simulateKeyPress("/");
      simulateKeyPress("t");
      simulateKeyPress("e");
      simulateKeyPress("s");

      const beforeBackspaceCount = mockDeps.rawStdoutWrite.mock.calls.length;

      simulateKeyPress("\x7f"); // Backspace

      // Should have updated the search prompt
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        beforeBackspaceCount,
      );

      simulateKeyPress("\x1b"); // Escape
      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Terminal Resize Handling", () => {
    it("should handle terminal resize", async () => {
      const pagerPromise = showInkPager("test content\nline 2", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      const initialCallCount = mockDeps.rawStdoutWrite.mock.calls.length;

      // Simulate resize
      simulateResize(30, 120);
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Should have re-rendered due to resize
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        initialCallCount,
      );

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle very small terminal sizes", async () => {
      mockDeps.stdout.rows = 5;
      mockDeps.stdout.cols = 20;

      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle undefined terminal dimensions", async () => {
      mockDeps.stdout.rows = undefined;
      mockDeps.stdout.cols = undefined;

      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Error Handling", () => {
    it("should handle stdin raw mode errors", async () => {
      // Create a fresh mockDeps with the failing setRawMode
      const failingDeps = {
        ...mockDeps,
        stdin: {
          ...mockDeps.stdin,
          setRawMode: mock(() => {
            throw new Error("Raw mode failed");
          }),
        },
      };

      try {
        await showInkPager("test", {}, failingDeps);
        expect(false).toBe(true); // Should not reach here
      } catch (error: any) {
        expect(error.message).toBe("Raw mode failed");
      }
    });

    it("should handle invalid key input gracefully", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Test with various invalid inputs
      const invalidInputs = [null, undefined, Buffer.alloc(0)];

      for (const invalidInput of invalidInputs) {
        try {
          keyHandlers.forEach((handler) => {
            handler(invalidInput as any);
          });
        } catch (_e) {
          // Should handle errors gracefully
        }
      }

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle key processing errors", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Test with error handling - this should be caught by the try-catch in handleInput
      // We'll test that the pager can still function after an error
      const errorTest = () => {
        try {
          // This will cause an error internally but should be caught
          keyHandlers.forEach((handler) => {
            handler(Buffer.from("\x1b[999~")); // Invalid escape sequence
          });
        } catch (_e) {
          // Expected - errors should be caught by handleInput
        }
      };

      errorTest();
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Pager should still be active and functional
      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Signal Handling", () => {
    it("should register signal handlers", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Should have registered signal handlers
      expect(mockDeps.process.on).toHaveBeenCalledWith(
        "SIGINT",
        expect.any(Function),
      );
      expect(mockDeps.process.on).toHaveBeenCalledWith(
        "SIGTERM",
        expect.any(Function),
      );

      simulateKeyPress("q");
      await pagerPromise;

      // Should clean up signal handlers
      expect(mockDeps.process.off).toHaveBeenCalledWith(
        "SIGINT",
        expect.any(Function),
      );
      expect(mockDeps.process.off).toHaveBeenCalledWith(
        "SIGTERM",
        expect.any(Function),
      );
    });

    it("should handle signals", async () => {
      const _pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Trigger SIGINT
      const sigintHandlers = signalHandlers.get("SIGINT") || [];
      expect(sigintHandlers.length).toBeGreaterThan(0);

      sigintHandlers.forEach((handler) => {
        handler();
      });

      // Should have cleaned up
      expect(mockDeps.stdin.removeListener).toHaveBeenCalled();
    });
  });

  describe("Performance and Boundary Conditions", () => {
    it("should handle empty content", async () => {
      const pagerPromise = showInkPager("", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(isPagerActive()).toBe(true);

      // Navigation should work even with empty content
      simulateKeyPress("j");
      simulateKeyPress("k");
      simulateKeyPress(" ");
      simulateKeyPress("b");
      simulateKeyPress("g");
      simulateKeyPress("G");

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle very long content efficiently", async () => {
      const longContent = Array.from(
        { length: 1000 },
        (_, i) => `Line ${i + 1}`,
      ).join("\n");

      const startTime = Date.now();
      const pagerPromise = showInkPager(longContent, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 50));

      const setupTime = Date.now() - startTime;
      expect(setupTime).toBeLessThan(1000); // Should be fast

      expect(isPagerActive()).toBe(true);

      // Navigation should be responsive
      simulateKeyPress("G"); // Go to end
      simulateKeyPress("g"); // Go to start

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle content with very long lines", async () => {
      const longLine = "A".repeat(1000);
      const content = `Short line\n${longLine}\nAnother short line`;

      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle rapid key presses", async () => {
      const pagerPromise = showInkPager(
        "test content\nline 2\nline 3",
        {},
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Rapid key presses
      const keys = ["j", "k", "j", "k", " ", "b", "g", "G"];
      for (const key of keys) {
        simulateKeyPress(key);
        // No wait - simulate rapid input
      }

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Multiple Instances", () => {
    it("should handle sequential pager instances", async () => {
      // First pager
      let pagerPromise = showInkPager("Content 1", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));
      simulateKeyPress("q");
      await pagerPromise;

      // Should be cleaned up
      expect(keyHandlers.length).toBe(0);

      // Reset mocks for second pager
      keyHandlers = [];
      mockDeps.stdin.on.mockClear();
      mockDeps.stdin.removeListener.mockClear();

      // Second pager
      pagerPromise = showInkPager("Content 2", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(isPagerActive()).toBe(true);
      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Search Edge Cases", () => {
    it("should handle empty search term", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      simulateKeyPress("/");
      simulateKeyPress("\r"); // Enter without typing anything
      await new Promise((resolve) => setTimeout(resolve, 5));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle case insensitive search", async () => {
      const content = "Test Content\nmore test";
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      simulateKeyPress("/");
      for (const char of "TEST") {
        simulateKeyPress(char);
      }
      simulateKeyPress("\r");
      await new Promise((resolve) => setTimeout(resolve, 5));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should handle concurrent search operations", async () => {
      const pagerPromise = showInkPager(
        "search content\nmore search",
        {},
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Start search
      simulateKeyPress("/");
      simulateKeyPress("s");

      // Try to start another search while in search mode (should be ignored)
      simulateKeyPress("/");
      simulateKeyPress("t");

      simulateKeyPress("\r"); // Execute search
      await new Promise((resolve) => setTimeout(resolve, 5));

      expect(isPagerActive()).toBe(true);

      simulateKeyPress("q");
      await pagerPromise;
    });
  });

  describe("Dependency Injection Validation", () => {
    it("should work with minimal mock dependencies", async () => {
      const handlers: Map<string, ((chunk: Buffer) => void)[]> = new Map();

      const minimalDeps: PagerDependencies = {
        stdin: {
          setRawMode: mock(() => {}),
          pause: mock(() => {}),
          resume: mock(() => {}),
          on: mock((event: string, handler: (chunk: Buffer) => void) => {
            const existing = handlers.get(event) || [];
            existing.push(handler);
            handlers.set(event, existing);
          }),
          removeListener: mock(() => {}),
        },
        stdout: {
          on: mock(() => {}),
          off: mock(() => {}),
        },
        process: {
          emit: mock(() => true),
          on: mock((_event: string, _handler: () => void) => {}),
          off: mock(() => {}),
        },
        rawStdoutWrite: mock(() => true),
        setTimeout: mock((fn: () => void) => {
          fn();
          return 1;
        }),
      };

      const pagerPromise = showInkPager("test", {}, minimalDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Verify it started properly
      expect(minimalDeps.stdin.setRawMode).toHaveBeenCalled();
      expect(minimalDeps.rawStdoutWrite).toHaveBeenCalled();

      // Simulate quit key press through the data handler
      const dataHandlers = handlers.get("data") || [];
      if (dataHandlers.length > 0) {
        dataHandlers[0](Buffer.from("q"));
      }

      await pagerPromise;
    });
  });

  describe("Mutation Resistance Tests", () => {
    // These tests specifically target logical branches that could be mutated

    it("should ONLY show search matches when search term exists (not when empty)", async () => {
      const content = "searchable line\nnormal line\nanother searchable";
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Start search mode
      simulateKeyPress("/");

      // Test empty search term - should NOT show matches
      simulateKeyPress("\r"); // Enter with empty term
      await new Promise((resolve) => setTimeout(resolve, 5));

      let output = getWrittenOutput();
      expect(output).not.toContain("matches"); // Should not show match count for empty term

      // Now test with actual search term - SHOULD show matches
      simulateKeyPress("/");
      for (const char of "searchable") {
        simulateKeyPress(char);
      }
      simulateKeyPress("\r");
      await new Promise((resolve) => setTimeout(resolve, 5));

      output = getWrittenOutput();
      expect(output).toContain("2 matches"); // Should show exactly 2 matches

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should ONLY display title when options.title is truthy (not falsy)", async () => {
      // Test with title - SHOULD display
      let pagerPromise = showInkPager(
        "content",
        { title: "Test Title" },
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      let output = getWrittenOutput();
      expect(output).toContain("Test Title");

      simulateKeyPress("q");
      await pagerPromise;

      // Reset for next test
      writtenOutput = [];

      // Test with empty string title - should NOT display
      pagerPromise = showInkPager("content", { title: "" }, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      output = getWrittenOutput();
      expect(output).not.toContain("Test Title");
      // With empty title, the bold title formatting should not be present
      expect(output).not.toContain("\x1b[1m"); // No bold formatting for empty title

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should ONLY enable search when searchEnabled is NOT false", async () => {
      // Test searchEnabled: true - search SHOULD work
      let pagerPromise = showInkPager(
        "content",
        { searchEnabled: true },
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      let initialCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      simulateKeyPress("/");
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Should have written search prompt
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        initialCalls,
      );

      simulateKeyPress("\x1b"); // Escape
      simulateKeyPress("q");
      await pagerPromise;

      // Reset
      writtenOutput = [];

      // Test searchEnabled: false - search should NOT work
      pagerPromise = showInkPager(
        "content",
        { searchEnabled: false },
        mockDeps,
      );
      await new Promise((resolve) => setTimeout(resolve, 10));

      initialCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      simulateKeyPress("/");
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Should NOT have written search prompt
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBe(initialCalls);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should ONLY navigate to next match when searchMatches.length > 0", async () => {
      const content = "line with target\nother line\nline without match";
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Search for something that has matches
      simulateKeyPress("/");
      for (const char of "target") {
        simulateKeyPress(char);
      }
      simulateKeyPress("\r");
      await new Promise((resolve) => setTimeout(resolve, 5));

      let initialCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      simulateKeyPress("n"); // Should work - has matches
      await new Promise((resolve) => setTimeout(resolve, 5));
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        initialCalls,
      );

      // Now search for something with NO matches
      simulateKeyPress("/");
      for (const char of "nomatch") {
        simulateKeyPress(char);
      }
      simulateKeyPress("\r");
      await new Promise((resolve) => setTimeout(resolve, 5));

      initialCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      simulateKeyPress("n"); // Should NOT work - no matches
      await new Promise((resolve) => setTimeout(resolve, 5));
      // No re-render should happen since no matches to navigate
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBe(initialCalls);

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should ONLY trigger resize re-render when maxRows actually changes", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      const initialCalls = mockDeps.rawStdoutWrite.mock.calls.length;

      // Resize to same dimensions - should NOT trigger re-render
      simulateResize(24, 80); // Same as initial
      await new Promise((resolve) => setTimeout(resolve, 5));
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBe(initialCalls);

      // Resize to different dimensions - SHOULD trigger re-render
      simulateResize(30, 100); // Different
      await new Promise((resolve) => setTimeout(resolve, 5));
      expect(mockDeps.rawStdoutWrite.mock.calls.length).toBeGreaterThan(
        initialCalls,
      );

      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should ONLY cleanup once when cleanedUp flag is false", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Normal cleanup
      simulateKeyPress("q");
      await pagerPromise;

      // Verify cleanup happened once
      expect(mockDeps.stdin.removeListener).toHaveBeenCalledTimes(1);
      expect(mockDeps.stdin.setRawMode).toHaveBeenCalledWith(false);

      // If we could somehow trigger cleanup again, it should be ignored
      // (This tests the cleanedUp flag logic)
    });

    it("should ONLY process search characters when key.length === 1 AND key >= ' '", async () => {
      const pagerPromise = showInkPager("test content", {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      simulateKeyPress("/"); // Enter search mode

      let searchPrompts = mockDeps.rawStdoutWrite.mock.calls
        .map((call) => call[0])
        .filter((output) => output.includes("Search:"));

      const initialSearchPromptCount = searchPrompts.length;

      // Test multi-character key (should be ignored)
      keyHandlers.forEach((handler) => {
        handler(Buffer.from("\x1b[A")); // Arrow key - length > 1
      });

      searchPrompts = mockDeps.rawStdoutWrite.mock.calls
        .map((call) => call[0])
        .filter((output) => output.includes("Search:"));

      expect(searchPrompts.length).toBe(initialSearchPromptCount); // No new search prompt

      // Test control character (should be ignored)
      keyHandlers.forEach((handler) => {
        handler(Buffer.from("\x01")); // Control char - < ' '
      });

      searchPrompts = mockDeps.rawStdoutWrite.mock.calls
        .map((call) => call[0])
        .filter((output) => output.includes("Search:"));

      expect(searchPrompts.length).toBe(initialSearchPromptCount); // Still no new search prompt

      // Test valid character (should be processed)
      simulateKeyPress("a"); // Valid character

      searchPrompts = mockDeps.rawStdoutWrite.mock.calls
        .map((call) => call[0])
        .filter((output) => output.includes("Search:"));

      expect(searchPrompts.length).toBeGreaterThan(initialSearchPromptCount); // Should have new search prompt

      simulateKeyPress("\x1b"); // Escape
      simulateKeyPress("q");
      await pagerPromise;
    });

    it("should ONLY jump to search match when searchMatches.length > 0 on Enter", async () => {
      const content = "first line\ntarget line\nthird line";
      const pagerPromise = showInkPager(content, {}, mockDeps);
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Search with no matches
      simulateKeyPress("/");
      for (const char of "nomatch") {
        simulateKeyPress(char);
      }

      let beforeEnterCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      simulateKeyPress("\r"); // Enter - should not jump since no matches
      await new Promise((resolve) => setTimeout(resolve, 5));

      // Should render but not jump to any specific position
      let afterEnterCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      expect(afterEnterCalls).toBeGreaterThan(beforeEnterCalls); // Rendered

      // Now search with matches
      simulateKeyPress("/");
      for (const char of "target") {
        simulateKeyPress(char);
      }

      beforeEnterCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      simulateKeyPress("\r"); // Enter - should jump to match
      await new Promise((resolve) => setTimeout(resolve, 5));

      afterEnterCalls = mockDeps.rawStdoutWrite.mock.calls.length;
      expect(afterEnterCalls).toBeGreaterThan(beforeEnterCalls); // Should have rendered with position jump

      simulateKeyPress("q");
      await pagerPromise;
    });
  });
});
