import { log } from "./logger";

/**
 * Status logger interface - pure function signatures
 */
export interface StatusLogger {
  info(message: string, context?: string): void;
  warn(message: string, context?: string): void;
  error(message: string, context?: string): void;
  debug(message: string, context?: string): void;
  set(message: string): void;
  clear(): void;
}

/**
 * Status change callback type
 */
export type StatusChangeHandler = (status: string) => void;

/**
 * Pure status service extracted from React hooks
 * Handles status messages and logging without React dependencies
 */
export class StatusService implements StatusLogger {
  private statusChangeHandler?: StatusChangeHandler;

  constructor(statusChangeHandler?: StatusChangeHandler) {
    this.statusChangeHandler = statusChangeHandler;
  }

  /**
   * Log info message and update status
   */
  info(message: string, context?: string): void {
    this.setStatus(message);
    log.info(message, context || "status");
  }

  /**
   * Log warning message and update status
   */
  warn(message: string, context?: string): void {
    this.setStatus(message);
    log.warn(message, context || "status");
  }

  /**
   * Log error message and update status
   */
  error(message: string, context?: string): void {
    this.setStatus(message);
    log.error(message, context || "status");
  }

  /**
   * Log debug message and update status
   */
  debug(message: string, context?: string): void {
    this.setStatus(message);
    log.debug(message, context || "status");
  }

  /**
   * Set status without logging
   */
  set(message: string): void {
    this.setStatus(message);
  }

  /**
   * Clear status
   */
  clear(): void {
    this.setStatus("");
  }

  /**
   * Update the status change handler
   */
  setStatusChangeHandler(handler: StatusChangeHandler): void {
    this.statusChangeHandler = handler;
  }

  /**
   * Remove the status change handler
   */
  clearStatusChangeHandler(): void {
    this.statusChangeHandler = undefined;
  }

  /**
   * Private method to notify status changes
   */
  private setStatus(message: string): void {
    if (this.statusChangeHandler) {
      this.statusChangeHandler(message);
    }
  }
}

/**
 * Factory function to create a status service with handler
 */
export function createStatusService(
  handler: StatusChangeHandler,
): StatusService {
  return new StatusService(handler);
}

/**
 * Create a no-op status service for testing
 */
export function createNoOpStatusService(): StatusService {
  return new StatusService();
}
