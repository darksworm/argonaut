import fs from "node:fs/promises";
import { showInkPager } from "../components/InkPager";
import { Logger } from "./logger";

export interface LogViewerOptions {
  title?: string;
}

export async function runLogViewerSession(opts: LogViewerOptions = {}) {
  const result = await Logger.getLatestSessionFile();
  if (result.isErr()) {
    throw new Error(`No log files found: ${result.error.message}`);
  }

  const logFilePath = result.value;

  // Read the log file content
  const logContent = await fs.readFile(logFilePath, "utf8");

  // Show in the reusable ink pager
  await showInkPager(logContent, {
    title: opts.title || "Session Logs",
    searchEnabled: true,
  });
}
