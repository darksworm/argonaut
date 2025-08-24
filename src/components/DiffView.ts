import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import {execa} from 'execa';
import {getManagedResourceDiffs} from '../api/applications.query';
import {getManifests as getManifestsApi} from '../api/rollback';
import type {Server} from '../types/server';
import {showInkPager} from './InkPager';
import {stripKubernetesFields} from '../utils/kube-strip';

// Cache delta availability at startup
let deltaAvailable: boolean | null = null;
const checkDelta = (): boolean => {
  if (deltaAvailable === null) {
    try {
      require('child_process').execSync('command -v delta', { stdio: 'ignore' });
      deltaAvailable = true;
    } catch {
      deltaAvailable = false;
    }
  }
  return deltaAvailable;
};

// Strip the temp file header from diff output
function stripDiffHeader(diffOutput: string): string {
  const lines = diffOutput.split('\n');
  let startIndex = -1;
  
  // Look for the first line that contains actual diff content
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    
    // Skip empty lines
    if (line === '') {
      continue;
    }
    
    // Skip temp file paths (contain /T/, /tmp/, /var/folders/, etc.)
    if (line.includes('/T/') || line.includes('/tmp/') || line.includes('/var/folders/') || line.includes('/TemporaryItems/')) {
      continue;
    }
    
    // Skip separator lines (any line that's mostly dashes/unicode box chars)
    if (line.match(/^[─━═—-]{10,}$/) || line.match(/^─+$/)) {
      continue;
    }
    
    // Skip arrows and file transitions
    if (line.includes('⟶') || line.includes('-->') || line.includes('→')) {
      continue;
    }
    
    // Look for actual diff content markers:
    // - Lines that look like "kind: Service" sections
    // - Lines with line numbers and pipes │
    // - Lines that start with actual YAML/content
    if (line.match(/^[─┐┌┘└│├┤┬┴┼]+$/) || // box drawing characters for section headers
        line.includes('│') ||  // side-by-side diff separator
        line.match(/^\s*\d+\s*│/) ||  // line numbers
        line.match(/^[a-zA-Z].*:/) ||  // YAML keys
        line.match(/^\s*-\s+/) ||  // YAML lists
        line.startsWith('apiVersion:') ||  // common YAML start
        line.startsWith('kind:') ||  // common YAML start
        line.startsWith('metadata:')) {  // common YAML start
      startIndex = i;
      break;
    }
  }
  
  // If we couldn't find a good starting point, just return original (safety)
  if (startIndex === -1) {
    return diffOutput;
  }
  
  return lines.slice(startIndex).join('\n');
}

export function toYamlDoc(input?: string, strip: boolean = true): string | null {
  if (!input) return null;
  try {
    const obj = JSON.parse(input);
    const processed = strip ? stripKubernetesFields(obj) : obj;
    return YAML.stringify(processed, {lineWidth: 120} as any);
  } catch {
    // assume already YAML
    return input;
  }
}

export async function writeTmp(docs: string[], label: string): Promise<string> {
  const file = path.join(os.tmpdir(), `${label}-${Date.now()}.yaml`);
  const content = docs.filter(Boolean).join("\n---\n");
  await fs.writeFile(file, content, 'utf8');
  return file;
}

export type DiffSessionOptions = {
  title?: string;
  /**
   * Whether to strip system fields for cleaner diffs (default: true)
   */
  stripFields?: boolean;
};

// Shows diff between two files using delta if available, otherwise git+less.
export async function runExternalDiffSession(fileLeft: string, fileRight: string, opts: DiffSessionOptions = {}): Promise<void> {
  const cols = (process.stdout as any)?.columns || 80;
  
  // Capture diff output
  let stdout: string;
  if (checkDelta()) {
    try {
      const result = await execa("bash", ["-lc", `delta --paging=never --line-numbers --side-by-side --width=${cols} "${fileLeft}" "${fileRight}"`]);
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
      const result = await execa("git", ["--no-pager", "diff", "--no-index", "--color=always", "--", fileLeft, fileRight]);
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
    title: opts.title || 'Diff View',
    searchEnabled: true,
  });
}

// High-level helpers that prepare data and run the session
export async function runAppDiffSession(server: Server, app: string, opts: DiffSessionOptions = {}): Promise<boolean> {
  const { stripFields = true } = opts;
  
  // Load diffs from API
  const diffsResult = await getManagedResourceDiffs(server, app);
  const diffs = diffsResult.isOk() ? diffsResult.value : [] as any[];
  const desiredDocs: string[] = [];
  const liveDocs: string[] = [];
  for (const d of diffs as any[]) {
    const tgt = toYamlDoc((d as any)?.targetState, stripFields);
    const live = toYamlDoc((d as any)?.liveState, stripFields);
    if (tgt) desiredDocs.push(tgt);
    if (live) liveDocs.push(live);
  }
  const desiredFile = await writeTmp(desiredDocs, `${app}-desired`);
  const liveFile = await writeTmp(liveDocs, `${app}-live`);

  // Quiet check — tell caller if there are no diffs
  try {
    await execa('git', ['--no-pager', 'diff', '--no-index', '--quiet', '--', desiredFile, liveFile]);
    return false; // no diffs
  } catch { /* has diffs */ }

  await runExternalDiffSession(liveFile, desiredFile, { 
    title: `${app} - Live vs Desired`
  });
  return true;
}

export async function runRollbackDiffSession(server: Server, app: string, revision: string, opts: DiffSessionOptions = {}, appNamespace?: string): Promise<boolean> {
  const { stripFields = true } = opts;
  
  const current = await getManifestsApi(server, app, undefined, undefined, appNamespace).catch(() => []);
  const target = await getManifestsApi(server, app, revision, undefined, appNamespace).catch(() => []);
  const currentDocs = current.map(manifest => toYamlDoc(manifest, stripFields)).filter(Boolean) as string[];
  const targetDocs = target.map(manifest => toYamlDoc(manifest, stripFields)).filter(Boolean) as string[];
  const currentFile = await writeTmp(currentDocs, `${app}-current`);
  const targetFile = await writeTmp(targetDocs, `${app}-target-${revision}`);

  try {
    await execa('git', ['--no-pager','diff','--no-index','--quiet','--', currentFile, targetFile]);
    return false;
  } catch { /* has diffs */ }

  await runExternalDiffSession(currentFile, targetFile, {
    title: `${app} - Current vs ${revision}`
  });
  return true;
}