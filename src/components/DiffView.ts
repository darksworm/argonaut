import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import {execa} from 'execa';
import {getManagedResourceDiffs} from '../api/applications.query';
import {getManifests as getManifestsApi} from '../api/rollback';
import type {Server} from '../types/server';
import {getPty} from "./pty";

const TTY_MODE = process.env.ARGONAUT_TTY_MODE || (process.versions?.bun ? "pty-direct" : "pty");

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

import { rawStdoutWrite, beginExclusiveInput, endExclusiveInput, enterExternal, exitExternal } from "../ink-control";

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

// Simple Ink-based pager with scroll and search
async function showInInkPager(content: string) {
  const lines = content.split('\n');
  const terminalRows = (process.stdout as any)?.rows || 24;
  // Leave reasonable space for both content and status - use about 60% of terminal height
  const maxRows = Math.max(8, terminalRows - 1); // -1 for status line
  let topLine = 0;
  let searchTerm = '';
  let searchMatches: number[] = [];
  let currentMatch = 0;

  const updateSearch = (term: string) => {
    searchTerm = term;
    searchMatches = [];
    if (term) {
      lines.forEach((line, idx) => {
        if (line.toLowerCase().includes(term.toLowerCase())) {
          searchMatches.push(idx);
        }
      });
    }
    currentMatch = 0;
  };

  const render = () => {
    // Clear screen and move to top
    rawStdoutWrite('\x1b[2J\x1b[H');
    
    const visibleLines = lines.slice(topLine, topLine + maxRows);
    visibleLines.forEach((line, idx) => {
      const lineNum = topLine + idx;
      let displayLine = line;
      
      // Highlight search matches
      if (searchTerm && line.toLowerCase().includes(searchTerm.toLowerCase())) {
        const isCurrentMatch = searchMatches[currentMatch] === lineNum;
        const highlightColor = isCurrentMatch ? '\x1b[43m\x1b[30m' : '\x1b[33m'; // Yellow bg for current, yellow text for others
        const regex = new RegExp(`(${searchTerm.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
        displayLine = line.replace(regex, `${highlightColor}$1\x1b[0m`);
      }
      
      rawStdoutWrite(displayLine + '\n');
    });

    // Status line
    const totalLines = lines.length;
    const progress = totalLines > 0 ? Math.round((topLine / totalLines) * 100) : 0;
    let statusLine = `Lines ${topLine + 1}-${Math.min(topLine + maxRows, totalLines)} of ${totalLines} (${progress}%)`;
    
    if (searchTerm) {
      statusLine += ` | Search: "${searchTerm}" (${searchMatches.length} matches)`;
      if (searchMatches.length > 0) {
        statusLine += ` [${currentMatch + 1}/${searchMatches.length}]`;
      }
    }
    statusLine += ' | q:quit j:down k:up /:search n:next N:prev';
    
    rawStdoutWrite(`\x1b[7m${statusLine.padEnd((process.stdout as any)?.columns || 80)}\x1b[0m`);
  };

  // Properly hand over stdin control from Ink
  enterExternal();
  beginExclusiveInput();

  return new Promise<void>((resolve) => {
    const stdin = process.stdin;
    try { 
      (stdin as any).setRawMode?.(true);
      stdin.resume();
    } catch {}
    
    let inputBuffer = '';
    let inSearchMode = false;

    const handleInput = (chunk: Buffer) => {
      const key = chunk.toString();
      
      if (inSearchMode) {
        if (key === '\r' || key === '\n') {
          // Enter - finish search
          inSearchMode = false;
          updateSearch(inputBuffer);
          inputBuffer = '';
          if (searchMatches.length > 0) {
            topLine = Math.max(0, searchMatches[0] - Math.floor(maxRows / 2));
          }
          render();
        } else if (key === '\x1b' || key === '\x7f') {
          // Escape or backspace - cancel search
          inSearchMode = false;
          inputBuffer = '';
          render();
        } else if (key === '\x7f' || key === '\b') {
          // Backspace in search
          inputBuffer = inputBuffer.slice(0, -1);
          rawStdoutWrite('\x1b[2K\rSearch: ' + inputBuffer);
        } else if (key.length === 1 && key >= ' ') {
          // Regular character in search
          inputBuffer += key;
          rawStdoutWrite('\x1b[2K\rSearch: ' + inputBuffer);
        }
        return;
      }

      switch (key) {
        case 'q':
        case 'Q':
          // Clean up
          stdin.removeListener('data', handleInput);
          try {
            (stdin as any).setRawMode?.(false);
            stdin.pause();
          } catch {}
          rawStdoutWrite('\x1b[2J\x1b[H'); // Clear screen
          
          // Hand stdin back to Ink
          endExclusiveInput();
          exitExternal();
          resolve();
          break;
        case 'j':
        case '\x1b[B': // Down arrow
          topLine = Math.min(topLine + 1, Math.max(0, lines.length - maxRows));
          render();
          break;
        case 'k':
        case '\x1b[A': // Up arrow
          topLine = Math.max(topLine - 1, 0);
          render();
          break;
        case ' ':
        case '\x1b[6~': // Page Down
          topLine = Math.min(topLine + maxRows, Math.max(0, lines.length - maxRows));
          render();
          break;
        case 'b':
        case '\x1b[5~': // Page Up
          topLine = Math.max(topLine - maxRows, 0);
          render();
          break;
        case '/':
          inSearchMode = true;
          inputBuffer = '';
          rawStdoutWrite('\x1b[2K\rSearch: ');
          break;
        case 'n':
          if (searchMatches.length > 0) {
            currentMatch = (currentMatch + 1) % searchMatches.length;
            topLine = Math.max(0, searchMatches[currentMatch] - Math.floor(maxRows / 2));
            render();
          }
          break;
        case 'N':
          if (searchMatches.length > 0) {
            currentMatch = (currentMatch - 1 + searchMatches.length) % searchMatches.length;
            topLine = Math.max(0, searchMatches[currentMatch] - Math.floor(maxRows / 2));
            render();
          }
          break;
        case 'g':
          topLine = 0;
          render();
          break;
        case 'G':
          topLine = Math.max(0, lines.length - maxRows);
          render();
          break;
      }
    };

    stdin.on('data', handleInput);
    render();
  });
}

// Runs a command in a bun-pty inside a minimal process that owns the TTY.
// No Ink, no logs. Low-latency, correct rendering.
export async function runBunPtyRunner(shell: string, args: string[]) {
  // Only used on Bun
  const { spawn: spawnPty } = await import("bun-pty");

  const cols = (process.stdout as any)?.columns || 80;
  const rows = (process.stdout as any)?.rows || 24;

  const enterAlt = () => {
    try { process.stdout.write('\x1b7\x1b[?1049h\x1b[2J\x1b[H\x1b[?25l\x1b[?7h'); } catch {}
  };
  const leaveAlt = () => {
    try { process.stdout.write('\x1b[?25h\x1b[?1049l\x1b8\x1b[0m\x1b[3J\x1b[2J\x1b[H'); } catch {}
  };

  const env = {
    ...process.env,
    TERM: "xterm-256color",
    COLORTERM: "truecolor",
    LANG: process.env.LANG || "en_US.UTF-8",
    LC_ALL: process.env.LC_ALL || "en_US.UTF-8",
    COLUMNS: String(cols),
    LINES: String(rows),
    LESS: "R",
  } as Record<string,string>;

  const pty = spawnPty(shell, args, {
    name: "xterm-256color",
    cols, rows,
    cwd: process.cwd(),
    env: env as any,
  });

  enterAlt();

  const sub = pty.onData((d: string) => { try { process.stdout.write(d); } catch {} });

  const onResize = () => {
    try { pty.resize((process.stdout as any)?.columns || 80, (process.stdout as any)?.rows || 24); } catch {}
  };
  onResize(); setTimeout(onResize, 0); setTimeout(onResize, 30);
  process.stdout.on("resize", onResize);

  const sin: any = process.stdin;
  try { sin.setRawMode?.(true); sin.resume?.(); } catch {}
  const onStdin = (b: Buffer) => { try { pty.write(b.toString("utf8")); } catch {} };
  process.stdin.on("data", onStdin);

  await new Promise<void>((resolve) => pty.onExit(() => resolve()));

  try { process.stdin.off("data", onStdin); process.stdout.off("resize", onResize); sub?.dispose?.(); } catch {}
  try { sin.setRawMode?.(false); sin.pause?.(); } catch {}

  leaveAlt();
}

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
  const cols = opts.cols || (process.stdout as any)?.columns || 80;
  const rows = opts.rows || (process.stdout as any)?.rows || 24;
  const width = opts.width || cols; // intentionally unused by delta; tools will use $COLUMNS
  const pager = process.platform === 'darwin'
    ? "less -R"
    : "less -R";


  // Direct command construction - much faster than complex shell strings
  const shell = 'bash';
  let argsLC: string[];
  
  if (checkDelta()) {
    // Use delta directly
    argsLC = ['-c', `DELTA_PAGER='${pager}' exec delta --paging=always --line-numbers --side-by-side "${fileLeft}" "${fileRight}"`];
  } else {
    // Use git diff with pager
    argsLC = ['-c', `exec git --no-pager diff --no-index --color=always -- "${fileLeft}" "${fileRight}" | ${pager}`];
  }

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

  if (TTY_MODE === "ink-pager") {
    // ---- No PTY at all: capture output and show in Ink pager ----
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
    await showInInkPager(cleanedOutput);
  } else if (TTY_MODE === "pty-direct" && process.versions?.bun) {
    // ---- Clean bun-pty runner: no Ink interference, optimal scrolling ----
    await runBunPtyRunner(shell, argsLC);
  } else if (TTY_MODE === "inherit") {
    // ---- Bun: give the real TTY to the child, no PTY layer ----
    // (works in Node too, but we keep node-pty as default there)
    enterExternal();
    beginExclusiveInput();
    
    // Properly detach stdin from parent process
    const stdinAny = process.stdin as any;
    try {
      stdinAny.setRawMode?.(false);
      stdinAny.pause?.();
      // Flush any pending input
      stdinAny.removeAllListeners?.('data');
    } catch {}

    if ((globalThis as any).Bun?.spawn) {
      const child = (globalThis as any).Bun.spawn({
        cmd: [shell, ...argsLC],
        cwd: process.cwd(),
        env,
        stdio: ["inherit","inherit","inherit"],
      });
      await child.exited;
    } else {
      const {spawn} = await import("node:child_process");
      const cp = spawn(shell, argsLC, {cwd: process.cwd(), env, stdio: ["inherit","inherit","inherit"]});
      await new Promise<void>((res, rej) => { cp.on("exit", () => res()); cp.on("error", rej); });
    }

    endExclusiveInput();
    exitExternal();
  } else if (TTY_MODE === "script" && process.platform !== "win32") {
    // ---- Force OS-level PTY via `script` ----
    enterExternal();
    beginExclusiveInput();
    
    // Properly detach stdin from parent process
    const stdinAny = process.stdin as any;
    try {
      stdinAny.setRawMode?.(false);
      stdinAny.pause?.();
      // Flush any pending input
      stdinAny.removeAllListeners?.('data');
    } catch {}

    const cmdline = process.platform === "darwin"
      ? ["script","-q","/dev/null", shell, ...argsLC]
      : ["script","-qfec", [shell, ...argsLC].join(' '), "/dev/null"];

    if ((globalThis as any).Bun?.spawn) {
      const child = (globalThis as any).Bun.spawn({ cmd: cmdline, cwd: process.cwd(), env, stdio: ["inherit","inherit","inherit"] });
      await child.exited;
    } else {
      const {spawn} = await import("node:child_process");
      const cp = spawn(cmdline[0], cmdline.slice(1), {cwd: process.cwd(), env, stdio: ["inherit","inherit","inherit"]});
      await new Promise<void>((res, rej) => { cp.on("exit", () => res()); cp.on("error", rej); });
    }

    endExclusiveInput();
    exitExternal();
  } else {
    // ---- Existing PTY path (node-pty on Node, bun-pty on Bun) ----
    // Enter alternate screen to isolate PTY rendering
    enterAltScreen();

    const spawnPty = await getPty();
    const pty = spawnPty(shell, argsLC, {
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
