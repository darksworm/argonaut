import {render} from "ink";
import {App} from "./components/App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { initializeLogger, log } from './services/logger';
import { tailLogs } from './services/log-tailer';
import { setupGlobalErrorHandlers } from './services/error-handler';

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

const args = process.argv.slice(2);
const isLogTailing = args.includes('--tail-logs');

async function main() {
    if (isLogTailing) {
        const sessionArg = args.find(arg => arg.startsWith('--session='));
        const session = sessionArg ? sessionArg.split('=')[1] : 'latest';
        
        console.log(`📋 Tailing logs for session: ${session === 'latest' ? 'latest' : session}`);
        console.log('Press \'q\' or Ctrl+C to stop tailing...\n');
        
        const result = await tailLogs({ session });
        
        if (result.isErr()) {
            console.error(`❌ Failed to tail logs: ${result.error.message}`);
            process.exit(1);
        }
        
        // Setup input handling for 'q' to quit
        process.stdin.setRawMode?.(true);
        process.stdin.resume();
        process.stdin.setEncoding('utf8');
        
        process.stdin.on('data', (key: string) => {
            if (key === 'q' || key === 'Q' || key === '\u0003') { // 'q', 'Q', or Ctrl+C
                console.log('\n👋 Stopping log tail...');
                process.exit(0);
            }
        });
        
        // Keep the process alive while tailing
        process.on('SIGINT', () => {
            console.log('\n👋 Stopping log tail...');
            process.exit(0);
        });
        
        return; // Don't start the main app
    }
    
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
            </ErrorBoundary>
        );
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
