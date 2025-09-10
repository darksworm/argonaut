import { describe, expect, mock, test } from "bun:test";
import {
  createNoOpStatusService,
  createStatusService,
  StatusService,
} from "../../services/status-service";

describe("StatusService", () => {
  test("should call status change handler when setting status", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.set("Test status");

    expect(mockHandler).toHaveBeenCalledWith("Test status");
  });

  test("should call status change handler with info method", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.info("Info message");

    expect(mockHandler).toHaveBeenCalledWith("Info message");
  });

  test("should call status change handler with warn method", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.warn("Warning message");

    expect(mockHandler).toHaveBeenCalledWith("Warning message");
  });

  test("should call status change handler with error method", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.error("Error message");

    expect(mockHandler).toHaveBeenCalledWith("Error message");
  });

  test("should call status change handler with debug method", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.debug("Debug message");

    expect(mockHandler).toHaveBeenCalledWith("Debug message");
  });

  test("should clear status", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.clear();

    expect(mockHandler).toHaveBeenCalledWith("");
  });

  test("should update status change handler", () => {
    const service = new StatusService();
    const newHandler = mock(() => {});

    service.setStatusChangeHandler(newHandler);
    service.set("Test message");

    expect(newHandler).toHaveBeenCalledWith("Test message");
  });

  test("should remove status change handler", () => {
    const mockHandler = mock(() => {});
    const service = createStatusService(mockHandler);

    service.clearStatusChangeHandler();
    service.set("Test message");

    // Handler should not be called after clearing
    expect(mockHandler).toHaveBeenCalledTimes(0);
  });

  test("createNoOpStatusService should not throw errors", () => {
    const service = createNoOpStatusService();

    // These should not throw even without a handler
    expect(() => service.set("test")).not.toThrow();
    expect(() => service.info("test")).not.toThrow();
    expect(() => service.warn("test")).not.toThrow();
    expect(() => service.error("test")).not.toThrow();
    expect(() => service.debug("test")).not.toThrow();
    expect(() => service.clear()).not.toThrow();
  });

  test("should not call handler when none is set", () => {
    const service = new StatusService();

    // These should not throw even without a handler
    expect(() => service.set("test")).not.toThrow();
    expect(() => service.info("test")).not.toThrow();
    expect(() => service.warn("test")).not.toThrow();
    expect(() => service.error("test")).not.toThrow();
    expect(() => service.debug("test")).not.toThrow();
    expect(() => service.clear()).not.toThrow();
  });
});
