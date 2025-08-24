import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import {execa} from 'execa';
import {getManagedResourceDiffs} from '../api/applications.query';
import {getManifests as getManifestsApi} from '../api/rollback';
import type {Server} from '../types/server';
import {getPty} from "./pty";

const TTY_MODE = process.env.ARGONAUT_TTY_MODE || (process.versions?.bun ? "inherit" : "pty");
import { rawStdoutWrite, beginExclusiveInput, endExclusiveInput } from "../ink-control";

function enterAltScreen() {
  // save cursor; enter alt screen; clear; home; hide cursor; enable wrap
  try { process.stdout.write('\x1b7\x1b[?1049h\x1b[2J\x1b[H\x1b[?25l\x1b[?7h'); } catch {}
}
function leaveAltScreen() {
  // show cursor; leave alt; restore cursor; reset SGR; clear scrollback; clear + home
  // order matters: exit alt first, then clean the main buffer
  try { process.stdout.write('\x1b[?25h\x1b[?1049l\x1b8\x1b[0m\x1b[3J\x1b[2J\x1b[H'); } catch {}
}

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
  const shell = process.platform === "win32" ? "powershell.exe" : "bash";
  const cols = opts.cols || (process.stdout as any)?.columns || 80;
  const rows = opts.rows || (process.stdout as any)?.rows || 24;
  const width = opts.width || cols; // intentionally unused by delta; tools will use $COLUMNS
  const pager = process.platform === 'darwin'
    ? "less -R"
    : "less -R";

  // Quiet check first: exit early if no differences
  try {
    await execa('git', ['--no-pager', 'diff', '--no-index', '--quiet', '--', fileLeft, fileRight]);
    return; // no diffs
  } catch { /* has diffs: continue */ }

  const cmd = `
set -e
if command -v delta >/dev/null 2>&1; then
  DELTA_PAGER='${pager}' delta --paging=always --line-numbers --side-by-side "${fileLeft}" "${fileRight}" || true
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

  // common env
  const env = {
    ...process.env,
    TERM: "xterm-256color",
    COLORTERM: "truecolor",
    LANG: process.env.LANG || "en_US.UTF-8",
    LC_ALL: process.env.LC_ALL || "en_US.UTF-8",
    COLUMNS: String(cols),
    LINES: String(rows),
    LESS: "R", // no -X/-K
  } as Record<string, string>;

  opts.onEnterExternal?.();
  try { beginExclusiveInput(); } catch {}

  if (TTY_MODE === "inherit") {
    // ---- Bun: give the real TTY to the child, no PTY layer ----
    // (works in Node too, but we keep node-pty as default there)
    if ((globalThis as any).Bun?.spawn) {
      // Use Bun.spawn
      const child = (globalThis as any).Bun.spawn({
        cmd: [shell, ...args],
        cwd: process.cwd(),
        env,
        stdio: ["inherit", "inherit", "inherit"],
      });
      await child.exited;
    } else {
      // Use Node.js child_process.spawn
      const spawn = require("node:child_process").spawn;
      const child = spawn(shell, args, {
        cwd: process.cwd(),
        env,
        stdio: ["inherit", "inherit", "inherit"],
      });
      await new Promise<void>((res, rej) => {
        child.on("exit", () => res());
        child.on("error", rej);
      });
    }
  } else if (TTY_MODE === "script" && process.platform !== "win32") {
    // ---- Force OS-level PTY via `script` ----
    // macOS/BSD: script [-q] [-F] command log
    // util-linux: script -qfec "cmd" /dev/null
    const cmdline = os.platform() === "darwin"
      ? ["script","-q","-F", shell, ...args, "/dev/null"]
      : ["script","-qfec", [shell, ...args].map(a => a.includes(" ") ? `"${a}"` : a).join(" "), "/dev/null"];

    if ((globalThis as any).Bun?.spawn) {
      // Use Bun.spawn
      const child = (globalThis as any).Bun.spawn({
        cmd: cmdline,
        cwd: process.cwd(),
        env,
        stdio: ["inherit", "inherit", "inherit"],
      });
      await child.exited;
    } else {
      // Use Node.js child_process.spawn
      const spawn = require("node:child_process").spawn;
      const child = spawn(cmdline[0], cmdline.slice(1), {
        cwd: process.cwd(),
        env,
        stdio: ["inherit", "inherit", "inherit"],
      });
      await new Promise<void>((res, rej) => {
        child.on("exit", () => res());
        child.on("error", rej);
      });
    }
  } else {
    // ---- Existing PTY path (node-pty on Node, bun-pty on Bun) ----
    // Enter alternate screen to isolate PTY rendering
    enterAltScreen();

    const spawnPty = await getPty();
    const pty = spawnPty(shell, args, {
      name: 'xterm-256color',
      cols, rows,
      cwd: process.cwd(),
      env: env as any,
    });

    const applyResize = () => { try { pty.resize((process.stdout as any)?.columns || 80, (process.stdout as any)?.rows || 24); } catch {} };
    applyResize(); setTimeout(applyResize, 0); setTimeout(applyResize, 30);
    const onResize = () => applyResize();
    process.stdout.on("resize", onResize);

    const sub = pty.onData((d: string) => { try { rawStdoutWrite(d); } catch {} });

    const stdinAny = process.stdin as any;
    const onStdin = (b: Buffer) => { try { pty.write(b.toString("utf8")); } catch {} };
    try { stdinAny.setRawMode?.(true); stdinAny.resume?.(); } catch {}
    if (opts.forwardInput !== false) process.stdin.on("data", onStdin);

    await new Promise<void>((resolve) => pty.onExit(() => resolve()));

    try { process.stdin.off("data", onStdin); process.stdout.off("resize", onResize); sub?.dispose?.(); } catch {}
    try { stdinAny.setRawMode?.(false); stdinAny.pause?.(); } catch {}
    
    // Leave alternate screen and restore UI
    leaveAltScreen();
  }

  try { endExclusiveInput(); } catch {}

  // Clean up temporary command file
  try {
    await fs.unlink(cmdFile);
  } catch { /* noop */ }

  opts.onExitExternal?.();
}

// High-level helpers that prepare data and run the session
export async function runAppDiffSession(server: Server, app: string, opts: DiffSessionOptions = {}): Promise<boolean> {
  // Load diffs from API
  const diffsResult = await getManagedResourceDiffs(server, app);
  const diffs = diffsResult.isOk() ? diffsResult.value : [] as any[];
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
