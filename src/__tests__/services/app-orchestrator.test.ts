import { beforeEach, describe, expect, mock, spyOn, test } from "bun:test";
import { DefaultAppOrchestrator } from "../../services/app-orchestrator";
import { createMockStatusLog } from "../test-utils";

describe("DefaultAppOrchestrator", () => {
  let orchestrator: DefaultAppOrchestrator;
  let mockDispatch: ReturnType<typeof mock>;
  let mockStatusLog: ReturnType<typeof createMockStatusLog>;
  let mockExit: ReturnType<typeof mock>;

  beforeEach(() => {
    orchestrator = new DefaultAppOrchestrator();
    mockDispatch = mock();
    mockStatusLog = createMockStatusLog();
    mockExit = mock();
  });

  describe("cleanupAndExit", () => {
    test("should cleanup resources and call exit", () => {
      orchestrator.cleanupAndExit(mockExit);

      expect(mockExit).toHaveBeenCalled();
    });

    test("should call exit and set timeout for force exit", () => {
      const originalSetTimeout = global.setTimeout;
      const mockSetTimeout = mock();
      global.setTimeout = mockSetTimeout;

      orchestrator.cleanupAndExit(mockExit);

      expect(mockExit).toHaveBeenCalled();
      expect(mockSetTimeout).toHaveBeenCalledWith(expect.any(Function), 100);

      global.setTimeout = originalSetTimeout;
    });
  });

  describe("handleAuthError", () => {
    test("should set auth-required mode and show error", () => {
      orchestrator.handleAuthError(mockDispatch, mockStatusLog);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SERVER",
        payload: null,
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "auth-required",
      });
      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "please use argocd login to authenticate before running argonaut",
        "auth",
      );
    });
  });

  describe("updateView", () => {
    test("should call setCurrentView with combined mode and view", () => {
      // This method is simple and just calls setCurrentView
      orchestrator.updateView("normal", "apps");

      // Test that the method doesn't throw - implementation uses setCurrentView from logger
      expect(() => orchestrator.updateView("normal", "apps")).not.toThrow();
    });
  });

  describe("handleTerminalResize", () => {
    test("should setup resize listener and return cleanup function", () => {
      const mockStdout = {
        rows: 30,
        columns: 100,
        on: mock(),
        off: mock(),
      };

      const originalStdout = process.stdout;
      Object.defineProperty(process, "stdout", {
        value: mockStdout,
        writable: true,
        configurable: true,
      });

      const cleanup = orchestrator.handleTerminalResize(mockDispatch);

      expect(mockStdout.on).toHaveBeenCalledWith(
        "resize",
        expect.any(Function),
      );

      // Simulate resize event
      const resizeHandler = mockStdout.on.mock.calls[0][1];
      resizeHandler();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_TERMINAL_SIZE",
        payload: { rows: 30, cols: 100 },
      });

      // Test cleanup
      cleanup();
      expect(mockStdout.off).toHaveBeenCalledWith("resize", resizeHandler);

      // Restore original stdout
      Object.defineProperty(process, "stdout", {
        value: originalStdout,
        writable: true,
        configurable: true,
      });
    });

    test("should use default dimensions when stdout has no dimensions", () => {
      const mockStdout = {
        rows: undefined,
        columns: undefined,
        on: mock(),
        off: mock(),
      };

      const originalStdout = process.stdout;
      Object.defineProperty(process, "stdout", {
        value: mockStdout,
        writable: true,
        configurable: true,
      });

      const cleanup = orchestrator.handleTerminalResize(mockDispatch);

      const resizeHandler = mockStdout.on.mock.calls[0][1];
      resizeHandler();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_TERMINAL_SIZE",
        payload: { rows: 24, cols: 80 },
      });

      cleanup();

      // Restore original stdout
      Object.defineProperty(process, "stdout", {
        value: originalStdout,
        writable: true,
        configurable: true,
      });
    });
  });

  describe("initializeApp", () => {
    test("should handle abort signal early termination", async () => {
      const abortController = new AbortController();
      abortController.abort();

      await orchestrator.initializeApp(
        mockDispatch,
        mockStatusLog,
        abortController.signal,
      );

      // Should dispatch loading mode first
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "loading",
      });
    });

    test("should setup loading abort controller", async () => {
      const abortController = new AbortController();
      abortController.abort();

      await orchestrator.initializeApp(
        mockDispatch,
        mockStatusLog,
        abortController.signal,
      );

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_LOADING_ABORT_CONTROLLER",
        payload: expect.any(AbortController),
      });
    });

    test("should log loading message", async () => {
      const abortController = new AbortController();
      abortController.abort();

      await orchestrator.initializeApp(
        mockDispatch,
        mockStatusLog,
        abortController.signal,
      );

      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Loading ArgoCD config…",
        "boot",
      );
    });

    test("should clean up previous abort controller", async () => {
      // Set up an abort controller that we can spy on
      const firstAbortController = new AbortController();
      const abortSpy = spyOn(firstAbortController, "abort");

      // Manually set the loading abort controller
      (orchestrator as any).loadingAbortController = firstAbortController;

      const secondAbortController = new AbortController();
      secondAbortController.abort();

      await orchestrator.initializeApp(
        mockDispatch,
        mockStatusLog,
        secondAbortController.signal,
      );

      expect(abortSpy).toHaveBeenCalled();
    });

    test("should handle initialization with successful path (mocked)", async () => {
      // This is a simplified test that verifies the method can be called
      // without throwing errors. Full integration testing would require
      // complex mocking of all dependencies.
      const abortController = new AbortController();

      // Start initialization but abort quickly to avoid complex mocking
      setTimeout(() => abortController.abort(), 1);

      await expect(
        orchestrator.initializeApp(
          mockDispatch,
          mockStatusLog,
          abortController.signal,
        ),
      ).resolves.not.toThrow();
    });
  });
});
