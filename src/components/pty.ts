/**
 * Creates and spawns a new PTY with the given command and arguments.
 *
 * @param file - Path to the executable to run.
 * @param args - Arguments for the executable.
 * @param options - Options for the PTY.
 * @returns A new PTY instance.
 */
type PtySpawn = (file: string, args?: string[], options?: IPtyForkOptions) => IPty;

let cached: PtySpawn | null = null;

export const getPty = async (): Promise<PtySpawn> => {
    if (cached) {
        return cached;
    }

    cached = await loadPty();
    return cached;
}

const loadPty = async (): Promise<PtySpawn> => {
    if (process.versions && (process.versions as any).bun) {
        return (await import("bun-pty")).spawn as PtySpawn;
    } else {
        return (await import("node-pty")).spawn as PtySpawn;
    }
}

export interface IDisposable {
    /**
     * Disposes the resource, performing any necessary cleanup.
     */
    dispose(): void;
}

export interface IPtyForkOptions {
    /**
     * The name of the terminal to be set in environment variables.
     */
    name: string;
    /**
     * The number of columns in the PTY.
     */
    cols?: number;
    /**
     * The number of rows in the PTY.
     */
    rows?: number;
    /**
     * The current working directory of the process.
     * Defaults to the current working directory of the parent process.
     */
    cwd?: string;
    /**
     * Environment variables to set for the process.
     */
    env?: Record<string, string>;
}
/**
 * Exit data for PTY process.
 */
export interface IExitEvent {
    /**
     * The process exit code.
     */
    exitCode: number;
    /**
     * The signal that caused the process to exit, if any.
     */
    signal?: number | string;
}
/**
 * Interface for interacting with a pseudo-terminal (PTY) instance.
 */
export interface IPty {
    /**
     * The PID of the process running in the PTY.
     */
    readonly pid: number;
    /**
     * The column size in characters.
     */
    readonly cols: number;
    /**
     * The row size in characters.
     */
    readonly rows: number;
    /**
     * The title of the active process.
     */
    readonly process: string;
    /**
     * Set a callback for when data is received from the PTY.
     */
    readonly onData: (listener: (data: string) => void) => IDisposable;
    /**
     * Event emitted when the PTY process exits.
     */
    readonly onExit: (listener: (event: IExitEvent) => void) => IDisposable;
    /**
     * Write data to the PTY.
     *
     * @param data - The data to write.
     */
    write(data: string): void;
    /**
     * Resize the PTY.
     *
     * @param columns - Number of columns (character width).
     * @param rows - Number of rows (character height).
     */
    resize(columns: number, rows: number): void;
    /**
     * Kill the process running in the PTY.
     *
     * @param signal - The signal to send to the process.
     * Defaults to "SIGTERM".
     */
    kill(signal?: string): void;
}
