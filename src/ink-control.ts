import { MutableStdin } from "./stdin/mutableStdin";
import { MutableStdout } from "./stdout/mutableStdout";

// A mutable stdout that Ink will write to (so we can mute/unmute without unmounting)
export const mutableStdout = new MutableStdout(process.stdout as any);

// A shared MutableStdin so this module and main can manage input handoff
export const mutableStdin = new MutableStdin();

// Mute Ink/logs (those writing via Ink's stdout) while PTY owns the terminal
export function enterExternal() {
  try {
    mutableStdout.mute();
  } catch {}
}

// Restore stdout for Ink/logs
export function exitExternal() {
  try {
    mutableStdout.unmute();
  } catch {}
}

// Exclusive stdin handoff helpers
export function beginExclusiveInput() {
  try {
    mutableStdin.detach(process.stdin as any);
  } catch {}
}
export function endExclusiveInput() {
  try {
    mutableStdin.attach(process.stdin as any);
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
