import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import { Logger } from './logger';
import { getPty } from '../components/pty';

export interface LogViewerOptions {
  onEnterExternal?: () => void;
  onExitExternal?: () => void;
}

export async function runLogViewerSession(opts: LogViewerOptions = {}) {
  const shell = 'bash';
  const cols = (process.stdout as any)?.columns || 80;
  const rows = (process.stdout as any)?.rows || 24;
  
  const result = await Logger.getLatestSessionFile();
  if (result.isErr()) {
    throw new Error(`No log files found: ${result.error.message}`);
  }
  
  const logFilePath = result.value;

  const cmd = `
set -e
if command -v less >/dev/null 2>&1; then
  less -r -+X -K "${logFilePath}"
else
  sh -c "cat '${logFilePath}'; printf '\\n[Press Enter to close] '; read -r _"
fi
`;

  const cmdFile = path.join(os.tmpdir(), `argonaut-logs-cmd.sh`);
  await fs.writeFile(cmdFile, cmd, 'utf8');
  await fs.chmod(cmdFile, 0o755);

  const args = ['-lc', cmdFile];

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

  onStdin = (chunk: Buffer) => {
    try { pty.write(chunk.toString('utf8')); } catch { /* noop */ }
  };
  process.stdin.on('data', onStdin);

  await new Promise<void>((resolve) => { pty.onExit(() => resolve()); });

  try {
    if (onStdin) process.stdin.off('data', onStdin);
    process.stdout.off('resize', onResize);
  } catch { /* noop */ }
  try {
    stdinAny.setRawMode?.(false);
    stdinAny.resume?.();
  } catch { /* noop */ }

  try {
    await fs.unlink(cmdFile);
  } catch { /* noop */ }

  opts.onExitExternal?.();
}
