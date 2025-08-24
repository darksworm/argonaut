import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { execa } from "execa";
import YAML from "yaml";
import { getManagedResourceDiffs } from "../api/applications.query";
import { getManifests as getManifestsApi } from "../api/rollback";
import {
  type ArgoResourceDiff,
  cleanArgoDiffs,
  createNeatConfig,
  DEFAULT_NEAT_CONFIG,
  type NeatConfig,
} from "../services/k8s-manifest-neat";
import type { Server } from "../types/server";
import { showInkPager } from "./InkPager";

// Cache delta availability at startup
let deltaAvailable: boolean | null = null;
const checkDelta = (): boolean => {
  if (deltaAvailable === null) {
    try {
      require("node:child_process").execSync("command -v delta", {
        stdio: "ignore",
      });
      deltaAvailable = true;
    } catch {
      deltaAvailable = false;
    }
  }
  return deltaAvailable;
};

// Strip the temp file header from diff output
function stripDiffHeader(diffOutput: string): string {
  const lines = diffOutput.split("\n");
  let startIndex = -1;

  // Look for the first line that contains actual diff content
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Skip empty lines
    if (line.trim() === "") {
      continue;
    }

    // Skip any line that contains temp file patterns (be very aggressive)
    if (
      line.includes("/var/folders/") ||
      line.includes("/tmp/") ||
      line.includes("/T/") ||
      line.includes("/TemporaryItems/") ||
      line.includes(".yaml") ||
      line.includes("⟶") ||
      line.includes("-->") ||
      line.includes("→") ||
      line.includes("differ")
    ) {
      continue;
    }

    // Skip separator lines and box drawing characters
    if (
      line.match(/^[─━═—\-_=┌┐┘└│├┤┬┴┼]{5,}$/) ||
      line.match(/^\s*[─━═—\-_=┌┐┘└│├┤┬┴┼\s]{10,}\s*$/) ||
      line.match(/^\d+:\s*\w+:\s*│$/) || // Lines like "39: spec: │"
      line.match(/^──────────┘$/) || // Box drawing end markers
      (line.match(/^.*│\s*$/) &&
        !line.includes("apiVersion") &&
        !line.includes("kind"))
    ) {
      // Single │ without content
      continue;
    }

    // Look for actual diff content - first line that has meaningful YAML or diff content
    if (
      line.match(/^\s*│\s*\d+\s*│/) || // line numbers with content
      (line.includes("│") &&
        (line.includes("apiVersion") ||
          line.includes("kind") ||
          line.includes("metadata") ||
          line.includes("image:") ||
          line.includes("name:"))) || // side-by-side diff with meaningful YAML
      line.match(/^[+-]\s*(apiVersion|kind|metadata|spec|data):/) || // git diff style changes with YAML
      line.startsWith("@@") || // git diff hunk headers
      (line.match(/^\s*(apiVersion|kind|metadata|spec|data):/) &&
        !line.includes("│"))
    ) {
      // Direct YAML keys
      startIndex = i;
      break;
    }
  }

  return lines.slice(startIndex).join("\n");
}

export function toYamlDoc(input?: string | any): string | null {
  if (!input) return null;

  try {
    // If input is already an object, stringify it
    if (typeof input === "object") {
      return YAML.stringify(input, { lineWidth: 120 } as any);
    }

    // If input is a string, try to parse it as JSON first
    const obj = JSON.parse(input);
    return YAML.stringify(obj, { lineWidth: 120 } as any);
  } catch {
    // assume already YAML
    return typeof input === "string" ? input : null;
  }
}

export async function writeTmp(docs: string[], label: string): Promise<string> {
  const file = path.join(os.tmpdir(), `${label}-${Date.now()}.yaml`);
  const content = docs.filter(Boolean).join("\n---\n");
  await fs.writeFile(file, content, "utf8");
  return file;
}

export type DiffSessionOptions = {
  title?: string;
  /**
   * Configuration for cleaning Kubernetes manifests
   */
  neatConfig?: NeatConfig;
  /**
   * Whether to enable manifest cleaning (default: true)
   */
  enableCleaning?: boolean;
};

// Shows diff between two files using delta if available, otherwise git+less.
export async function runExternalDiffSession(
  fileLeft: string,
  fileRight: string,
  opts: DiffSessionOptions = {},
): Promise<void> {
  const cols = (process.stdout as any)?.columns || 80;

  // Capture diff output
  let stdout: string;
  if (checkDelta()) {
    try {
      const result = await execa("bash", [
        "-lc",
        `delta --paging=never --line-numbers --side-by-side --width=${cols} "${fileLeft}" "${fileRight}"`,
      ]);
      stdout = result.stdout;
    } catch (error: any) {
      // Delta returns exit code 1 when differences are found - this is normal
      if (error.exitCode === 1 && error.stdout) {
        stdout = error.stdout;
      } else {
        throw error; // Re-throw unexpected errors
      }
    }
  } else {
    try {
      const result = await execa("git", [
        "--no-pager",
        "diff",
        "--no-index",
        "--color=always",
        "--",
        fileLeft,
        fileRight,
      ]);
      stdout = result.stdout;
    } catch (error: any) {
      // Git diff returns exit code 1 when differences are found - this is normal
      if (error.exitCode === 1 && error.stdout) {
        stdout = error.stdout;
      } else {
        throw error; // Re-throw unexpected errors
      }
    }
  }

  // Strip the unhelpful temp file header from the diff output
  const cleanedOutput = stripDiffHeader(stdout);

  // Show in the reusable ink pager
  await showInkPager(cleanedOutput, {
    title: opts.title || "Diff View",
    searchEnabled: true,
  });
}

// High-level helpers that prepare data and run the session
export async function runAppDiffSession(
  server: Server,
  app: string,
  opts: DiffSessionOptions = {},
): Promise<boolean> {
  const neatConfig = opts.neatConfig || DEFAULT_NEAT_CONFIG;
  const enableCleaning = opts.enableCleaning !== false; // default to true

  // Load diffs from API
  const diffsResult = await getManagedResourceDiffs(server, app);
  const diffs = diffsResult.isOk()
    ? diffsResult.value
    : ([] as ArgoResourceDiff[]);

  const desiredDocs: string[] = [];
  const liveDocs: string[] = [];

  if (enableCleaning && diffs.length > 0) {
    // Use the manifest cleaning approach
    const cleanedDiffs = cleanArgoDiffs(
      diffs as ArgoResourceDiff[],
      neatConfig,
    );

    for (const cleanedDiff of cleanedDiffs) {
      const targetYaml = toYamlDoc(cleanedDiff.target);
      const liveYaml = toYamlDoc(cleanedDiff.live);

      if (targetYaml) desiredDocs.push(targetYaml);
      if (liveYaml) liveDocs.push(liveYaml);
    }
  } else {
    // Fallback to original approach
    for (const d of diffs as any[]) {
      const tgt = toYamlDoc((d as any)?.targetState);
      const live = toYamlDoc((d as any)?.liveState);
      if (tgt) desiredDocs.push(tgt);
      if (live) liveDocs.push(live);
    }
  }

  if (desiredDocs.length === 0 && liveDocs.length === 0) {
    return false; // no diffs to show
  }

  const desiredFile = await writeTmp(desiredDocs, `${app}-desired`);
  const liveFile = await writeTmp(liveDocs, `${app}-live`);

  // Quiet check — tell caller if there are no diffs
  try {
    await execa("git", [
      "--no-pager",
      "diff",
      "--no-index",
      "--quiet",
      "--",
      desiredFile,
      liveFile,
    ]);
    return false; // no diffs
  } catch {
    /* has diffs */
  }

  await runExternalDiffSession(liveFile, desiredFile, {
    title: `${app} - Live vs Desired${enableCleaning ? " (Cleaned)" : ""}`,
  });
  return true;
}

export async function runRollbackDiffSession(
  server: Server,
  app: string,
  revision: string,
  opts: DiffSessionOptions = {},
  appNamespace?: string,
): Promise<boolean> {
  const current = await getManifestsApi(
    server,
    app,
    undefined,
    undefined,
    appNamespace,
  ).catch(() => []);
  const target = await getManifestsApi(
    server,
    app,
    revision,
    undefined,
    appNamespace,
  ).catch(() => []);

  let currentDocs: string[] = [];
  let targetDocs: string[] = [];

  if (
    opts.enableCleaning !== false &&
    (current.length > 0 || target.length > 0)
  ) {
    // Apply manifest cleaning to rollback diffs as well
    const neatConfig = opts.neatConfig || DEFAULT_NEAT_CONFIG;

    currentDocs = current
      .map((manifest) => {
        try {
          const parsed =
            typeof manifest === "string" ? JSON.parse(manifest) : manifest;
          const cleaned = cleanArgoDiffs(
            [
              {
                kind: parsed.kind || "Unknown",
                namespace: parsed.metadata?.namespace || "",
                name: parsed.metadata?.name || "",
                targetState: JSON.stringify(parsed),
              },
            ],
            neatConfig,
          )[0];
          return cleaned ? toYamlDoc(cleaned.target) : null;
        } catch {
          return toYamlDoc(manifest);
        }
      })
      .filter(Boolean) as string[];

    targetDocs = target
      .map((manifest) => {
        try {
          const parsed =
            typeof manifest === "string" ? JSON.parse(manifest) : manifest;
          const cleaned = cleanArgoDiffs(
            [
              {
                kind: parsed.kind || "Unknown",
                namespace: parsed.metadata?.namespace || "",
                name: parsed.metadata?.name || "",
                targetState: JSON.stringify(parsed),
              },
            ],
            neatConfig,
          )[0];
          return cleaned ? toYamlDoc(cleaned.target) : null;
        } catch {
          return toYamlDoc(manifest);
        }
      })
      .filter(Boolean) as string[];
  } else {
    // Fallback to original approach
    currentDocs = current.map(toYamlDoc).filter(Boolean) as string[];
    targetDocs = target.map(toYamlDoc).filter(Boolean) as string[];
  }

  if (currentDocs.length === 0 && targetDocs.length === 0) {
    return false;
  }

  const currentFile = await writeTmp(currentDocs, `${app}-current`);
  const targetFile = await writeTmp(targetDocs, `${app}-target-${revision}`);

  try {
    await execa("git", [
      "--no-pager",
      "diff",
      "--no-index",
      "--quiet",
      "--",
      currentFile,
      targetFile,
    ]);
    return false;
  } catch {
    /* has diffs */
  }

  await runExternalDiffSession(currentFile, targetFile, {
    title: `${app} - Current vs ${revision}${opts.enableCleaning !== false ? " (Cleaned)" : ""}`,
  });
  return true;
}

/**
 * Create a custom neat configuration for specific use cases
 *
 * Example usage:
 * const config = createAppSpecificNeatConfig('my-app', {
 *   removeDefaults: false,  // Keep defaults for this app
 *   metadataAnnotations: ['my-custom-annotation'] // Remove additional annotations
 * });
 */
export function createAppSpecificNeatConfig(
  _appName: string,
  overrides: Partial<NeatConfig> = {},
): NeatConfig {
  return createNeatConfig({
    ...overrides,
    // Add app-specific ArgoCD annotations to strip
    metadataAnnotations: [
      ...DEFAULT_NEAT_CONFIG.metadataAnnotations,
      `argocd.argoproj.io/tracking-id`, // ArgoCD tracking annotation
      ...(overrides.metadataAnnotations || []),
    ],
  });
}

/**
 * Debug function to compare cleaned vs uncleaned diffs
 */
export async function runDebugDiffSession(
  server: Server,
  app: string,
): Promise<void> {
  console.log("🔍 Running debug diff session...");

  // Run both cleaned and uncleaned versions
  console.log("\\n📋 Generating cleaned diff...");
  const hasCleanedDiff = await runAppDiffSession(server, app, {
    enableCleaning: true,
    title: `${app} - Cleaned Diff`,
  });

  if (hasCleanedDiff) {
    console.log("\\n⏳ Press any key to see the uncleaned diff...");
    process.stdin.setRawMode(true);
    process.stdin.resume();
    await new Promise((resolve) => process.stdin.once("data", resolve));
    process.stdin.setRawMode(false);
    process.stdin.pause();
  }

  console.log("\\n📄 Generating uncleaned diff...");
  await runAppDiffSession(server, app, {
    enableCleaning: false,
    title: `${app} - Full Diff (Uncleaned)`,
  });
}

export { DEFAULT_NEAT_CONFIG, createNeatConfig, type NeatConfig };
