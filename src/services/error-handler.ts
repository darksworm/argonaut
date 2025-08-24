import type React from "react";
import { log } from "./logger";

// Global error handling setup
export function setupGlobalErrorHandlers(): void {
  // Handle unhandled promise rejections
  process.on("unhandledRejection", (reason: unknown, promise: Promise<any>) => {
    const error = reason instanceof Error ? reason : new Error(String(reason));

    log.error("Unhandled promise rejection", "global", {
      message: error.message,
      stack: error.stack,
      promise: promise.toString(),
    });

    // Don't exit the process for unhandled rejections in development
    // In production, this might be more critical
    console.error("⚠️  Unhandled Promise Rejection:", error.message);
    console.error("💡 View logs with: argonaut --tail-logs");
  });

  // Handle uncaught exceptions
  process.on("uncaughtException", (error: Error) => {
    log.error("Uncaught exception", "global", {
      message: error.message,
      stack: error.stack,
      name: error.name,
    });

    console.error("💥 Uncaught Exception:", error.message);
    console.error("💡 View logs with: argonaut --tail-logs");

    // For uncaught exceptions, we should exit gracefully
    process.exit(1);
  });

  // Handle process warnings
  process.on("warning", (warning: Error) => {
    log.warn("Process warning", "global", {
      name: warning.name,
      message: warning.message,
      stack: warning.stack,
    });
  });

  // Log process exit
  process.on("exit", (code: number) => {
    log.info("Process exiting", "global", { exitCode: code });
  });

  // Log various signals
  ["SIGINT", "SIGTERM", "SIGHUP"].forEach((signal) => {
    process.on(signal, () => {
      log.info(`Received ${signal} signal`, "global");
    });
  });
}

// React Error logging utility
export function logReactError(error: Error, errorInfo?: React.ErrorInfo): void {
  // Include full stack trace in the main message for complete visibility
  const fullMessage = `React error: ${error.message}\n${error.stack || "No stack trace available"}${
    errorInfo?.componentStack
      ? `\n\nComponent Stack:${errorInfo.componentStack}`
      : ""
  }`;

  log.error(fullMessage, "react", {
    name: error.name,
    originalMessage: error.message,
    stack: error.stack,
    componentStack: errorInfo?.componentStack,
  });

  // Also log to console for immediate visibility during development
  console.error("💥 React component error:", error.message);
  if (errorInfo?.componentStack) {
    console.error("Component stack:", errorInfo.componentStack);
  }
}

// Wrapper function to catch and log errors from async functions
export function withErrorLogging<T extends any[], R>(
  fn: (...args: T) => Promise<R>,
  context: string,
): (...args: T) => Promise<R> {
  return async (...args: T): Promise<R> => {
    try {
      return await fn(...args);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));

      log.error(`Error in ${context}`, context, {
        message: err.message,
        stack: err.stack,
        args: args.length > 0 ? args : undefined,
      });

      // Re-throw the error so the caller can handle it
      throw error;
    }
  };
}

// Wrapper for synchronous functions
export function withErrorLoggingSync<T extends any[], R>(
  fn: (...args: T) => R,
  context: string,
): (...args: T) => R {
  return (...args: T): R => {
    try {
      return fn(...args);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));

      log.error(`Error in ${context}`, context, {
        message: err.message,
        stack: err.stack,
        args: args.length > 0 ? args : undefined,
      });

      // Re-throw the error so the caller can handle it
      throw error;
    }
  };
}
