import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import {ALL_LICENSES} from '../data/licenses.js';
import {getPty} from "./pty";

export type LicenseSessionOptions = {
  // If true, forward process.stdin to the PTY (interactive paging). Defaults to true.
  forwardInput?: boolean;
  // Hooks for callers to toggle UI mode/state when entering/leaving external mode.
  onEnterExternal?: () => void;
  onExitExternal?: () => void;
  // Optional override for terminal columns/rows (otherwise read from process.stdout)
  cols?: number;
  rows?: number;
};

// Spawns a PTY to show license content using less
export async function runLicenseSession(opts: LicenseSessionOptions = {}): Promise<void> {
  const shell = 'bash';
  const cols = opts.cols || (process.stdout as any)?.columns || 80;
  const rows = opts.rows || (process.stdout as any)?.rows || 24;

  // Create temporary file with license content
  const licenseFile = path.join(os.tmpdir(), `licenses-${Date.now()}.txt`);
  await fs.writeFile(licenseFile, ALL_LICENSES, 'utf8');

  const cmd = `
set -e
if command -v less >/dev/null 2>&1; then
  less -r -+X -K "${licenseFile}"
else
  sh -c "cat '${licenseFile}'; printf '\\n[Press Enter to close] '; read -r _"
fi
`;

  const args = process.platform === 'win32'
    ? ['-NoProfile', '-NonInteractive', '-Command', cmd]
    : ['-lc', cmd];

  opts.onEnterExternal?.();

  const spawnPty = await getPty();
  const pty = spawnPty(shell, args, {
    name: 'xterm-256color',
    cols: process.stdout.columns,
    rows: process.stdout.rows,
    cwd: process.cwd(),
    env: { ...(process.env as any),
    cols,
    rows,
    COLORTERM: 'truecolor'
    },
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

  // cleanup temp file
  try {
    await fs.unlink(licenseFile);
  } catch { /* noop */ }

  opts.onExitExternal?.();
}