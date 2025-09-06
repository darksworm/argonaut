// Set NODE_ENV before any other imports to ensure production builds are used
// This ensures react-reconciler and other packages use their production builds
if (typeof process !== "undefined" && !process.env.NODE_ENV) {
  process.env.NODE_ENV = "production";
}

import type { ReadStream, WriteStream } from "node:tty";
import packageJson from "../package.json";

if (process.argv.includes("--version") || process.argv.includes("-v")) {
  console.log(packageJson.version);
  process.exit(0);
}

function setupAlternateScreen() {
  if (typeof process === "undefined") return;
  const out = process.stdout as any;
  const isTTY = !!out && typeof out.isTTY === "boolean" ? out.isTTY : false;
  if (!isTTY) return;

  let cleaned = false;
  const enable = () => {
    try {
      out.write("\u001B[?1049h");
    } catch {}
  };
  const disable = () => {
    if (cleaned) return;
    cleaned = true;
    try {
      out.write("\u001B[?1049l");
    } catch {}
  };

  enable();

  process.on("exit", disable);
  // Don't handle SIGINT here - let the App component handle it for proper cleanup
  process.on("SIGTERM", () => {
    disable();
    process.exit(143);
  });
  process.on("SIGHUP", () => {
    disable();
    process.exit(129);
  });
  process.on("uncaughtException", (err) => {
    disable();
    console.error(err);
    process.exit(1);
  });
  process.on("unhandledRejection", (reason) => {
    disable();
    console.error(reason);
    process.exit(1);
  });
}

async function main() {
  const { render } = await import("ink");
  const { App } = await import("./components/App");
  const { ErrorBoundary } = await import("./components/ErrorBoundary");
  const {
    beginExclusiveInput,
    endExclusiveInput,
    enterExternal,
    exitExternal,
    mutableStdin,
    mutableStdout,
  } = await import("./ink-control");
  const { setupGlobalErrorHandlers } = await import("./services/error-handler");
  const { initializeLogger, log } = await import("./services/logger");

  // Initialize logger for normal app mode
  const loggerResult = await initializeLogger();
  if (loggerResult.isErr()) {
    console.error(
      `❌ Failed to initialize logger: ${loggerResult.error.message}`,
    );
    process.exit(1);
  }

  const logger = loggerResult.value;

  // Setup global error handlers (after logger is initialized)
  setupGlobalErrorHandlers();

  log.info("Argonaut session started", "main", {
    sessionId: logger.getSessionId(),
    logFile: logger.getLogFilePath(),
  });

  setupAlternateScreen();

  process.on("external-enter", () => {
    beginExclusiveInput();
    enterExternal();
  });

  process.on("external-exit", async () => {
    endExclusiveInput();
    exitExternal();

    const oldCols = process.stdout.columns;
    process.stdout.columns = oldCols - 1;
    process.stdout.emit("resize");

    await new Promise((resolve) => setTimeout(resolve, 0));

    process.stdout.columns = oldCols;
    process.stdout.emit("resize");
  });

  try {
    render(
      <ErrorBoundary>
        <App />
      </ErrorBoundary>,
      {
        stdout: mutableStdout as any as WriteStream,
        stderr: process.stderr,
        stdin: mutableStdin as any as ReadStream,
        patchConsole: false,
        exitOnCtrlC: false,
      },
    );
    try {
      mutableStdin.attach(process.stdin as any);
    } catch {}
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    log.error("Failed to render React application", "main", {
      message: err.message,
      stack: err.stack,
    });
    console.error("❌ Failed to render application:", err.message);
    process.exit(1);
  }
}

main().catch((error) => {
  console.error("❌ Failed to start application:", error);
  process.exit(1);
});
