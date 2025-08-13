import {promises as fs} from 'fs';
import {join} from 'path';
import {tmpdir} from 'os';

interface NpmRegistryResponse {
  version: string;
}

interface VersionCheckResult {
  currentVersion: string;
  latestVersion?: string;
  isOutdated: boolean;
  lastChecked?: number;
  error?: string;
}

const CACHE_DURATION = 60 * 60 * 1000; // 1 hour in milliseconds
const CACHE_FILE = join(tmpdir(), '.argonaut-version-cache.json');

async function getCachedResult(): Promise<VersionCheckResult | null> {
  try {
    const data = await fs.readFile(CACHE_FILE, 'utf-8');
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
    await fs.writeFile(CACHE_FILE, JSON.stringify(result), 'utf-8');
  } catch {
    // Ignore storage errors
  }
}

function compareVersions(current: string, latest: string): boolean {
  const currentParts = current.split('.').map(Number);
  const latestParts = latest.split('.').map(Number);
  
  for (let i = 0; i < Math.max(currentParts.length, latestParts.length); i++) {
    const currentPart = currentParts[i] || 0;
    const latestPart = latestParts[i] || 0;
    
    if (latestPart > currentPart) return true;
    if (latestPart < currentPart) return false;
  }
  
  return false;
}

async function fetchLatestVersion(): Promise<string> {
  const response = await fetch('https://registry.npmjs.org/argonaut-cli/latest');
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  
  const data: NpmRegistryResponse = await response.json();
  return data.version;
}

export async function checkVersion(currentVersion: string): Promise<VersionCheckResult> {
  // Check cache first
  const cached = await getCachedResult();
  if (cached && cached.currentVersion === currentVersion) {
    return cached;
  }
  
  const result: VersionCheckResult = {
    currentVersion,
    isOutdated: false,
  };
  
  try {
    const latestVersion = await fetchLatestVersion();
    result.latestVersion = latestVersion;
    result.isOutdated = compareVersions(currentVersion, latestVersion);
  } catch (error) {
    result.error = error instanceof Error ? error.message : 'Unknown error';
  }
  
  await setCachedResult(result);
  return result;
}
