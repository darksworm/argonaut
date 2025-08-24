import { promises as fs } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { err, ok, ResultAsync } from "neverthrow";
import pino from "pino";

export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: string;
  data?: any;
}

interface LoggerConfig {
  sessionId: string;
  keepSessions?: number; // Number of old sessions to keep (default: 5)
}

class Logger {
  private pinoLogger: pino.Logger;
  private sessionId: string;
  private logFilePath: string;
  private keepSessions: number;

  constructor(config: LoggerConfig) {
    this.sessionId = config.sessionId;
    this.keepSessions = config.keepSessions ?? 5;
    this.logFilePath = join(tmpdir(), `argonaut-session-${this.sessionId}.log`);

    this.pinoLogger = pino(
      {
        level: "debug",
        timestamp: pino.stdTimeFunctions.isoTime,
      },
      pino.destination({
        dest: this.logFilePath,
        sync: false,
      }),
    );
  }

  debug(message: string, context?: string, data?: any): void {
    this.pinoLogger.debug({ context, data }, message);
  }

  info(message: string, context?: string, data?: any): void {
    this.pinoLogger.info({ context, data }, message);
  }

  warn(message: string, context?: string, data?: any): void {
    this.pinoLogger.warn({ context, data }, message);
  }

  error(message: string, context?: string, data?: any): void {
    this.pinoLogger.error({ context, data }, message);
  }

  getSessionId(): string {
    return this.sessionId;
  }

  getLogFilePath(): string {
    return this.logFilePath;
  }

  // Clean up old session files, keeping only the most recent N sessions
  cleanupOldSessions(): ResultAsync<void, { message: string }> {
    return ResultAsync.fromPromise(fs.readdir(tmpdir()), () => ({
      message: "Failed to read temp directory",
    })).andThen((files) => {
      const sessionFiles = files
        .filter((f) => f.startsWith("argonaut-session-") && f.endsWith(".log"))
        .map((f) => ({ name: f, path: join(tmpdir(), f) }))
        .sort((a, b) => b.name.localeCompare(a.name)); // Sort by name (timestamp) descending

      if (sessionFiles.length <= this.keepSessions) {
        return ResultAsync.fromSafePromise(Promise.resolve());
      }

      const filesToDelete = sessionFiles.slice(this.keepSessions);
      const deletePromises = filesToDelete.map((file) =>
        ResultAsync.fromPromise(fs.unlink(file.path), () => ({
          message: `Failed to delete old log file: ${file.name}`,
        })),
      );

      return ResultAsync.combine(deletePromises).map(() => undefined);
    });
  }

  // Get all available session files, sorted by modification time (newest first)
  static getAvailableSessions(): ResultAsync<string[], { message: string }> {
    return ResultAsync.fromPromise(fs.readdir(tmpdir()), () => ({
      message: "Failed to read temp directory",
    }))
      .andThen((files) => {
        const sessionFiles = files.filter(
          (f) => f.startsWith("argonaut-session-") && f.endsWith(".log"),
        );

        return ResultAsync.combine(
          sessionFiles.map((f) =>
            ResultAsync.fromPromise(fs.stat(join(tmpdir(), f)), () => ({
              message: `Failed to stat ${f}`,
            })).map((stats) => ({ file: f, mtime: stats.mtime })),
          ),
        );
      })
      .map((fileStats) =>
        fileStats
          .sort((a, b) => b.mtime.getTime() - a.mtime.getTime()) // Sort by modification time descending
          .map((fs) =>
            fs.file.replace("argonaut-session-", "").replace(".log", ""),
          ),
      );
  }

  // Get the path for a specific session file
  static getSessionFilePath(sessionId: string): string {
    return join(tmpdir(), `argonaut-session-${sessionId}.log`);
  }

  // Get the most recent session file
  static getLatestSessionFile(): ResultAsync<string, { message: string }> {
    return Logger.getAvailableSessions().andThen((sessions) => {
      if (sessions.length === 0) {
        return err({ message: "No session files found" });
      }
      return ok(Logger.getSessionFilePath(sessions[0]));
    });
  }

  // Read and parse log entries from a session file
  static readSessionLogs(
    sessionId: string,
  ): ResultAsync<LogEntry[], { message: string }> {
    const filePath = Logger.getSessionFilePath(sessionId);

    return ResultAsync.fromPromise(fs.readFile(filePath, "utf-8"), () => ({
      message: `Failed to read session file: ${sessionId}`,
    })).map((content) => {
      const lines = content
        .trim()
        .split("\n")
        .filter((line) => line.length > 0);
      return lines.map((line) => {
        try {
          const parsed = JSON.parse(line);

          // Convert pino numeric levels to string levels
          let level: LogLevel = "info";
          if (typeof parsed.level === "number") {
            switch (parsed.level) {
              case 10:
                level = "debug";
                break;
              case 20:
                level = "debug";
                break;
              case 30:
                level = "info";
                break;
              case 40:
                level = "warn";
                break;
              case 50:
                level = "error";
                break;
              case 60:
                level = "error";
                break;
              default:
                level = "info";
            }
          } else if (typeof parsed.level === "string") {
            level = parsed.level as LogLevel;
          }

          return {
            timestamp:
              parsed.time || parsed.timestamp || new Date().toISOString(),
            level,
            message: parsed.msg || parsed.message || "Unknown message",
            context: parsed.context,
            data: parsed.data,
          } as LogEntry;
        } catch {
          // Fallback for malformed log lines
          return {
            timestamp: new Date().toISOString(),
            level: "info" as LogLevel,
            message: line,
          } as LogEntry;
        }
      });
    });
  }

  // Flush any pending writes and close the logger
  close(): ResultAsync<void, { message: string }> {
    return ResultAsync.fromPromise(
      new Promise<void>((resolve, reject) => {
        this.pinoLogger.flush((error) => {
          if (error) reject(error);
          else resolve();
        });
      }),
      (error: any) => ({ message: error?.message || "Failed to flush logger" }),
    );
  }
}

// Singleton logger instance
let globalLogger: Logger | null = null;

// Global view context for logging
let globalViewContext: string | null = null;

// Initialize the global logger with a session ID based on current timestamp
export function initializeLogger(
  sessionId?: string,
): ResultAsync<Logger, { message: string }> {
  const id =
    sessionId ||
    new Date()
      .toISOString()
      .replace(/[:.]/g, "-")
      .replace("T", "-")
      .split("Z")[0];

  return ResultAsync.fromPromise(Promise.resolve(), () => ({
    message: "Logger initialization failed",
  })).map(() => {
    globalLogger = new Logger({ sessionId: id });

    // Clean up old sessions in the background
    globalLogger.cleanupOldSessions().mapErr((error) => {
      console.warn("Failed to cleanup old log sessions:", error.message);
    });

    return globalLogger;
  });
}

// Get the global logger instance
export function getLogger(): Logger | null {
  return globalLogger;
}

// Set the current view context for all logs
export function setCurrentView(view: string): void {
  globalViewContext = view;
}

// Get the current view context
export function getCurrentView(): string | null {
  return globalViewContext;
}

// Helper function to combine context with view
function combineContext(context?: string): string | undefined {
  const parts: string[] = [];

  if (globalViewContext) {
    parts.push(`view:${globalViewContext}`);
  }

  if (context) {
    parts.push(context);
  }

  return parts.length > 0 ? parts.join("|") : undefined;
}

// Convenience functions for logging
export const log = {
  debug: (message: string, context?: string, data?: any) =>
    globalLogger?.debug(message, combineContext(context), data),
  info: (message: string, context?: string, data?: any) =>
    globalLogger?.info(message, combineContext(context), data),
  warn: (message: string, context?: string, data?: any) =>
    globalLogger?.warn(message, combineContext(context), data),
  error: (message: string, context?: string, data?: any) =>
    globalLogger?.error(message, combineContext(context), data),
};

export { Logger };
