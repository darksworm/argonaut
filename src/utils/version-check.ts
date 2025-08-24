import { promises as fs } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { err, ResultAsync } from "neverthrow";

interface NpmRegistryResponse {
  version: string;
}

interface VersionCheckResult {
  currentVersion: string;
  latestVersion?: string;
  isOutdated: boolean;
  lastChecked?: number;
}

const CACHE_DURATION = 4 * 60 * 60 * 1000; // 4 hours in milliseconds
const CACHE_FILE = join(tmpdir(), ".argonaut-version-cache.json");

async function getCachedResult(): Promise<VersionCheckResult | null> {
  try {
    const data = await fs.readFile(CACHE_FILE, "utf-8");
    const result = JSON.parse(data) as VersionCheckResult;

    if (!result.lastChecked) return null;

    const now = Date.now();
    if (now - result.lastChecked > CACHE_DURATION) {
      return null;
    }

    return result;
  } catch {
    return null;
  }
}

async function setCachedResult(result: VersionCheckResult): Promise<void> {
  try {
    result.lastChecked = Date.now();
    await fs.writeFile(CACHE_FILE, JSON.stringify(result), "utf-8");
  } catch {
    // Ignore storage errors
  }
}

function compareVersions(current: string, latest: string): boolean {
  const currentParts = current.split(".").map(Number);
  const latestParts = latest.split(".").map(Number);

  for (let i = 0; i < Math.max(currentParts.length, latestParts.length); i++) {
    const currentPart = currentParts[i] || 0;
    const latestPart = latestParts[i] || 0;

    if (latestPart > currentPart) return true;
    if (latestPart < currentPart) return false;
  }

  return false;
}

function fetchLatestVersion(): ResultAsync<string, { message: string }> {
  return ResultAsync.fromPromise(
    fetch("https://registry.npmjs.org/argonaut-cli/latest"),
    (error: any) => ({
      message: error?.message || "Network error while checking for updates",
    }),
  )
    .andThen((response) => {
      if (!response.ok) {
        return err({
          message: `Failed to fetch from npm registry: HTTP ${response.status}`,
        });
      }
      return ResultAsync.fromPromise(
        response.json() as Promise<NpmRegistryResponse>,
        (error: any) => ({
          message: error?.message || "Failed to parse registry response",
        }),
      );
    })
    .map((data) => data.version);
}

export function checkVersion(
  currentVersion: string,
): ResultAsync<VersionCheckResult, { message: string }> {
  return ResultAsync.fromPromise(getCachedResult(), () => ({
    message: "Cache read error",
  })).andThen((cached) => {
    if (cached && cached.currentVersion === currentVersion) {
      return ResultAsync.fromSafePromise(Promise.resolve(cached));
    }

    return fetchLatestVersion().map((latestVersion) => {
      const result: VersionCheckResult = {
        currentVersion,
        latestVersion,
        isOutdated: compareVersions(currentVersion, latestVersion),
      };

      // Cache the result (fire and forget)
      setCachedResult(result).catch(() => {});

      return result;
    });
  });
}
