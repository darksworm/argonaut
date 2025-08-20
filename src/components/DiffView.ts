import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import {execa} from 'execa';
import {getManagedResourceDiffs} from '../api/applications.query';
import {getManifests as getManifestsApi} from '../api/rollback';
import type {Server} from '../types/server';
import {getPty} from "./pty";

export function toYamlDoc(input?: string): string | null {
  if (!input) return null;
  try {
    const obj = JSON.parse(input);
    return YAML.stringify(obj, {lineWidth: 120} as any);
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
  // If true, forward process.stdin to the PTY (interactive paging). Defaults to true.
  forwardInput?: boolean;
  // Hooks for callers to toggle UI mode/state when entering/leaving external mode.
  onEnterExternal?: () => void;
  onExitExternal?: () => void;
  // Optional override for terminal columns/rows (otherwise read from process.stdout)
  cols?: number;
  rows?: number;
  // Side-by-side option width; if not provided, computed from terminal cols.
  width?: number;
};

// Spawns a PTY to show diff between two files using delta if available, otherwise git+less.
// fileLeft/right: order matters for delta/git presentation; pass according to your desired semantics.
export async function runExternalDiffSession(fileLeft: string, fileRight: string, opts: DiffSessionOptions = {}): Promise<void> {
  const shell = 'bash';
  const cols = opts.cols || (process.stdout as any)?.columns || 80;
  const rows = opts.rows || (process.stdout as any)?.rows || 24;
  const width = opts.width || cols;
  const pager = process.platform === 'darwin' ? "less -r -+X -K" : "less -R -+X -K";

  // Quiet check first: exit early if no differences
  try {
    await execa('git', ['--no-pager', 'diff', '--no-index', '--quiet', '--', fileLeft, fileRight]);
    return; // no diffs
  } catch { /* has diffs: continue */ }

  const cmd = `
set -e
if command -v delta >/dev/null 2>&1; then
  DELTA_PAGER='${pager}' delta --paging=always --line-numbers --side-by-side --width=${width} "${fileLeft}" "${fileRight}" || true
else
  PAGER='${pager}'
  if ! command -v less >/dev/null 2>&1; then
    PAGER='sh -c "cat; printf \"\\n[Press Enter to close] \"; read -r _"'
  fi
  git --no-pager diff --no-index --color=always -- "${fileLeft}" "${fileRight}" | eval "$PAGER" || true
fi
`;

  // Write command to temporary file
  const cmdFile = path.join(os.tmpdir(), `argonaut-diff-cmd.sh`);
  await fs.writeFile(cmdFile, cmd, 'utf8');
  await fs.chmod(cmdFile, 0o755);

  const args = process.platform === 'win32'
    ? ['-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-File', cmdFile]
    : [cmdFile];

  opts.onEnterExternal?.();

  const spawnPty = await getPty();
  const pty = spawnPty(shell, args as any, {
    name: 'xterm-256color',
    cols, rows,
    cwd: process.cwd(),
    env: { ...(process.env as any), COLORTERM: 'truecolor' } as any,
  });

  const onResize = () => {
    try {
      pty.resize((process.stdout as any)?.columns || 80, (process.stdout as any)?.rows || 24);
    } catch { /* noop */ }
  };

  const onPtyData = (data: string) => {
    try { process.stdout.write(data); } catch { /* noop */ }
  };
  pty.onData(onPtyData);
  process.stdout.on('resize', onResize);

  const stdinAny = process.stdin as any;
  let onStdin: ((chunk: Buffer) => void) | null = null;
  try {
    stdinAny.resume?.();
    // In most cases we want raw mode for interactive paging
    stdinAny.setRawMode?.(true);
  } catch { /* noop */ }

  if (opts.forwardInput !== false) {
    onStdin = (chunk: Buffer) => {
      try { pty.write(chunk.toString('utf8')); } catch { /* noop */ }
    };
    process.stdin.on('data', onStdin);
  }

  await new Promise<void>((resolve) => { pty.onExit(() => resolve()); });

  // cleanup
  try {
    if (onStdin) process.stdin.off('data', onStdin);
    process.stdout.off('resize', onResize);
  } catch { /* noop */ }
  try {
    stdinAny.setRawMode?.(true);
    stdinAny.resume?.();
  } catch { /* noop */ }

  // Clean up temporary command file
  try {
    await fs.unlink(cmdFile);
  } catch { /* noop */ }

  opts.onExitExternal?.();
}

// High-level helpers that prepare data and run the session
export async function runAppDiffSession(server: Server, app: string, opts: DiffSessionOptions = {}): Promise<boolean> {
  // Load diffs from API
  const diffs = await getManagedResourceDiffs(server, app).catch(() => [] as any[]);
  const desiredDocs: string[] = [];
  const liveDocs: string[] = [];
  for (const d of diffs as any[]) {
    const tgt = toYamlDoc((d as any)?.targetState);
    const live = toYamlDoc((d as any)?.liveState);
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

  await runExternalDiffSession(liveFile, desiredFile, opts);
  return true;
}

export async function runRollbackDiffSession(server: Server, app: string, revision: string, opts: DiffSessionOptions = {}, appNamespace?: string): Promise<boolean> {
  const current = await getManifestsApi(server, app, undefined, undefined, appNamespace).catch(() => []);
  const target = await getManifestsApi(server, app, revision, undefined, appNamespace).catch(() => []);
  const currentDocs = current.map(toYamlDoc).filter(Boolean) as string[];
  const targetDocs = target.map(toYamlDoc).filter(Boolean) as string[];
  const currentFile = await writeTmp(currentDocs, `${app}-current`);
  const targetFile = await writeTmp(targetDocs, `${app}-target-${revision}`);

  try {
    await execa('git', ['--no-pager','diff','--no-index','--quiet','--', currentFile, targetFile]);
    return false;
  } catch { /* has diffs */ }

  await runExternalDiffSession(currentFile, targetFile, opts);
  return true;
}
