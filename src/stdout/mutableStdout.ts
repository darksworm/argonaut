import { Writable } from "node:stream";

class NullSink extends Writable {
  _write(_chunk: any, _enc: BufferEncoding, cb: (err?: Error | null) => void) {
    cb();
  }
}

// A stdout wrapper that can be muted/unmuted but still reports terminal size
export class MutableStdout extends Writable {
  private target: any; // real TTY-like stdout
  private nullSink = new NullSink();
  private sink: Writable;
  private onResizeBound: (() => void) | null = null;

  constructor(target: any) {
    super();
    this.target = target;
    this.sink = target;

    // Forward resize events from the real TTY to this wrapper so Ink can react
    if (typeof target?.on === "function") {
      this.onResizeBound = () => {
        try {
          this.emit("resize");
        } catch {}
      };
      try {
        target.on("resize", this.onResizeBound);
      } catch {}
    }
  }

  // Ensure Ink can read current terminal width/height
  get columns(): number {
    return Number(this.target?.columns ?? 80);
  }
  get rows(): number {
    return Number(this.target?.rows ?? 24);
  }
  get isTTY(): boolean {
    return Boolean(this.target?.isTTY ?? true);
  }

  // Optional: expose getColorDepth to satisfy libraries that probe it
  getColorDepth?(env?: any): number;

  mute() {
    this.sink = this.nullSink;
  }

  unmute() {
    this.sink = this.target;
  }

  // Clean up listener if needed
  dispose() {
    try {
      if (this.onResizeBound && typeof this.target?.off === "function") {
        this.target.off("resize", this.onResizeBound);
      }
    } catch {}
    this.onResizeBound = null;
  }

  _write(chunk: any, enc: BufferEncoding, cb: (err?: Error | null) => void) {
    // Route writes either to the real stdout or to the null sink
    this.sink.write(chunk, enc, cb);
  }
}
