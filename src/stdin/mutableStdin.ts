import { PassThrough } from "node:stream";

export class MutableStdin extends PassThrough {
  private attached = false;
  private onData?: (b: Buffer) => void;
  private real: NodeJS.ReadStream | null = null;
  // Pretend to be a TTY for Ink; manage raw mode locally without touching real stdin
  public isTTY: boolean = true;
  public setRawMode(_enable: boolean) {
    // No-op: Ink requires this to exist; we intentionally don't alter real stdin here
    // The enable parameter is intentionally ignored
  }

  // Provide ref/unref that Ink expects on TTY streams
  public ref() {
    try {
      (this.real as any)?.ref?.();
    } catch {}
  }
  public unref() {
    try {
      (this.real as any)?.unref?.();
    } catch {}
  }

  attach(real: NodeJS.ReadStream) {
    if (this.attached) return;
    this.attached = true;
    this.real = real;
    this.onData = (buf: Buffer) => {
      try {
        this.write(buf);
      } catch {}
    };
    try {
      real.on("data", this.onData);
    } catch {}
    // Ensure real stdin is in raw mode for immediate key events and resumed
    try {
      (real as any).setRawMode?.(true);
    } catch {}
    try {
      (real as any).resume?.();
    } catch {}
    try {
      this.resume();
    } catch {}
  }

  detach(real: NodeJS.ReadStream) {
    if (!this.attached) return;
    this.attached = false;
    try {
      if (this.onData) real.off("data", this.onData);
    } catch {}
    this.onData = undefined;
    // Do NOT end the stream; keep it alive for later re-attachment
    try {
      this.pause();
    } catch {}
    // Restore cooked mode and pause the real stdin; PTY will take ownership afterwards
    try {
      (real as any).setRawMode?.(false);
    } catch {}
    try {
      (real as any).pause?.();
    } catch {}
    this.real = null;
  }
}
