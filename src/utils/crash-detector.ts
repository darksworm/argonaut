import { spawn } from "node:child_process";

export interface CrashDetectorOptions {
  command: string;
  args: string[];
  onCrash?: () => Promise<void> | void;
  showLogsOnCrash?: boolean;
}

// Detects if a process crashed and optionally shows logs
export async function runWithCrashDetection(
  options: CrashDetectorOptions,
): Promise<void> {
  const { command, args, onCrash, showLogsOnCrash = true } = options;

  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      stdio: "inherit",
    });

    child.on("exit", async (code, signal) => {
      // Consider it a crash if:
      // - Exit code is non-zero
      // - Process was killed by a signal (except SIGINT/SIGTERM which are normal)
      const crashed =
        code !== 0 || (signal && !["SIGINT", "SIGTERM"].includes(signal));

      if (crashed) {
        console.error(
          `💥 Process crashed with ${code !== null ? `code ${code}` : `signal ${signal}`}`,
        );

        // Run custom crash handler
        if (onCrash) {
          try {
            await onCrash();
          } catch (error) {
            console.error("Error in crash handler:", error);
          }
        }

        // Auto-show logs if requested
        if (showLogsOnCrash) {
          console.error("💡 Opening logs to help diagnose the crash...\n");

          try {
            // Import here to avoid circular dependencies
            const { tailLogs } = await import("../services/log-tailer");
            const result = await tailLogs({ session: "latest" });

            if (result.isErr()) {
              console.error(`Failed to tail logs: ${result.error.message}`);
              console.error("💡 Try manually: argonaut --tail-logs");
            }
          } catch (error) {
            console.error("Failed to show logs:", error);
            console.error("💡 Try manually: argonaut --tail-logs");
          }
        }

        reject(
          new Error(
            `Process crashed with ${code !== null ? `code ${code}` : `signal ${signal}`}`,
          ),
        );
      } else {
        resolve();
      }
    });

    child.on("error", (error) => {
      console.error("💥 Failed to start process:", error.message);
      reject(error);
    });
  });
}
