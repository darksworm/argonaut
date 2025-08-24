import {render} from "ink";
import {App} from "./components/App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { initializeLogger, log } from './services/logger';
import { setupGlobalErrorHandlers } from './services/error-handler';
import { mutableStdout } from './ink-control';
import { MutableStdin } from './stdin/mutableStdin';
import {ReadStream, WriteStream} from "node:tty";

// Export a shared MutableStdin so other modules (ink-control) can manage input handoff
export const mutableStdin = new MutableStdin();

function setupAlternateScreen() {
    if (typeof process === 'undefined') return;
    const out = process.stdout as any;
    const isTTY = !!out && typeof out.isTTY === 'boolean' ? out.isTTY : false;
    if (!isTTY) return;

    let cleaned = false;
    const enable = () => {
        try {
            out.write("\u001B[?1049h");
        } catch {
        }
    };
    const disable = () => {
        if (cleaned) return;
        cleaned = true;
        try {
            out.write("\u001B[?1049l");
        } catch {
        }
    };

    enable();

    process.on('exit', disable);
    process.on('SIGINT', () => {
        disable();
        process.exit(130);
    });
    process.on('SIGTERM', () => {
        disable();
        process.exit(143);
    });
    process.on('SIGHUP', () => {
        disable();
        process.exit(129);
    });
    process.on('uncaughtException', (err) => {
        disable();
        console.error(err);
        process.exit(1);
    });
    process.on('unhandledRejection', (reason) => {
        disable();
        console.error(reason);
        process.exit(1);
    });
};

async function main() {
    // Initialize logger for normal app mode
    const loggerResult = await initializeLogger();
    if (loggerResult.isErr()) {
        console.error(`❌ Failed to initialize logger: ${loggerResult.error.message}`);
        process.exit(1);
    }
    
    const logger = loggerResult.value;
    
    // Setup global error handlers (after logger is initialized)
    setupGlobalErrorHandlers();
    
    log.info('Argonaut session started', 'main', { 
        sessionId: logger.getSessionId(),
        logFile: logger.getLogFilePath() 
    });

    setupAlternateScreen();
    
    try {
        render(
            <ErrorBoundary>
                <App/>
            </ErrorBoundary>,
            {
                stdout: mutableStdout as any as WriteStream,
                stderr: process.stderr,
                stdin: mutableStdin as any as ReadStream,
                patchConsole: false,
                exitOnCtrlC: false
            }
        );
        try { mutableStdin.attach(process.stdin as any); } catch {}
    } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        log.error('Failed to render React application', 'main', {
            message: err.message,
            stack: err.stack,
        });
        console.error('❌ Failed to render application:', err.message);
        process.exit(1);
    }
}

main().catch((error) => {
    console.error('❌ Failed to start application:', error);
    process.exit(1);
});
