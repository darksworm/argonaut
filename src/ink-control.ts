import type {Instance as InkInstance} from 'ink';
import { MutableStdout } from './stdout/mutableStdout';

// Keeps a reference to the current Ink instance and a function to re-render the App
let inkInstance: InkInstance | null = null;
let renderApp: (() => void) | null = null;

// A mutable stdout that Ink will write to (so we can mute/unmute without unmounting)
export const mutableStdout = new MutableStdout(process.stdout as any);

export function setInkInstance(instance: InkInstance, renderFn: () => void) {
  inkInstance = instance;
  renderApp = renderFn;
}

// Mute Ink/logs (those writing via Ink's stdout) while PTY owns the terminal
export function enterExternal() {
  try { mutableStdout.mute(); } catch {}
}

// Restore stdout for Ink/logs
export function exitExternal() {
  try { mutableStdout.unmute(); } catch {}
  // Ensure Ink can receive input again after PTY restored stdin to cooked/paused
  try {
    const stdinAny = process.stdin as any;
    stdinAny.setRawMode?.(true);
    stdinAny.resume?.();
  } catch {}
}

// Write directly to the real stdout even when muted
export function rawStdoutWrite(chunk: any): boolean {
  const writer = (process.stdout as any).write;
  try {
    return writer.call(process.stdout, chunk);
  } catch {
    return false;
  }
}
