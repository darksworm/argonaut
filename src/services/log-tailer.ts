import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { err, ok, ResultAsync } from "neverthrow";
import { Logger } from "./logger";

export interface TailOptions {
  session?: string; // Session ID to tail, or 'latest' for most recent
}

// Simple log tailing for crash detection - just shows logs without complex session watching
export function tailLogs(
  options: TailOptions = {},
): ResultAsync<void, { message: string }> {
  const { session = "latest" } = options;

  return resolveSessionFile(session).andThen(({ filePath }) => {
    if (!existsSync(filePath)) {
      return err({ message: `Log file not found: ${filePath}` });
    }

    return ResultAsync.fromPromise(
      new Promise<void>((resolve, reject) => {
        try {
          // Simple tail with pino-pretty formatting
          const pipeCommand = `tail -f "${filePath}" | npx pino-pretty --colorize --translateTime SYS:HH:MM:ss --ignore hostname,pid`;

          const shellProcess = spawn("sh", ["-c", pipeCommand], {
            stdio: ["ignore", process.stdout, process.stderr],
          });

          // Handle errors
          shellProcess.on("error", (error) => {
            reject(
              new Error(`Failed to start shell process: ${error.message}`),
            );
          });

          // Handle process exits
          shellProcess.on("exit", (code, signal) => {
            if (code !== 0 && signal !== "SIGTERM") {
              console.error(
                `Shell process exited with code ${code}, signal ${signal}`,
              );
            }
            resolve();
          });

          // Give process a moment to start up
          setTimeout(() => resolve(), 100);
        } catch (error: any) {
          reject(new Error(`Failed to create tail process: ${error.message}`));
        }
      }),
      (error: any) => ({
        message: error?.message || "Failed to spawn tail process",
      }),
    );
  });
}

// Resolve session identifier to actual file path
function resolveSessionFile(
  session: string,
): ResultAsync<{ sessionId: string; filePath: string }, { message: string }> {
  if (session === "latest") {
    return Logger.getLatestSessionFile().andThen((filePath) => {
      // Extract session ID from file path
      const match = filePath.match(/argonaut-session-(.+)\.log$/);
      if (!match) {
        return err({
          message: "Could not extract session ID from latest log file",
        });
      }
      return ok({ sessionId: match[1], filePath });
    });
  }

  // Treat as specific session ID
  const filePath = Logger.getSessionFilePath(session);
  return ResultAsync.fromSafePromise(
    Promise.resolve({ sessionId: session, filePath }),
  );
}
