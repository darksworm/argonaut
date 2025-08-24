#!/usr/bin/env bun
import { spawn, type ChildProcess } from 'child_process';
import { existsSync } from 'fs';
import { ResultAsync, err, ok } from 'neverthrow';
import { Logger } from '../src/services/logger';

interface TailOptions {
  session?: string; // Session ID to tail, or 'latest' for most recent
  follow?: boolean; // Whether to follow the file (like tail -f), default true
  lines?: number;   // Number of lines to show initially, default 50
}

interface TailProcess {
  process: ChildProcess;
  sessionId: string;
  filePath: string;
}

class DevLogTailer {
  private tailProcess: TailProcess | null = null;
  private sessionWatchInterval: NodeJS.Timeout | null = null;

  // Start tailing logs with pino-pretty formatting
  start(options: TailOptions = {}): ResultAsync<TailProcess, { message: string }> {
    if (this.tailProcess) {
      return err({ message: 'Log tailing is already running' });
    }

    const { session = 'latest', follow = true, lines = 50 } = options;

    return this.resolveSessionFile(session)
      .andThen(({ sessionId, filePath }) => {
        if (!existsSync(filePath)) {
          return err({ message: `Log file not found: ${filePath}` });
        }

        return this.spawnTailProcess(sessionId, filePath, { follow, lines })
          .map(tailProcess => {
            // If we're tailing 'latest', watch for new sessions
            if (session === 'latest' && follow) {
              this.startSessionWatching();
            }
            return tailProcess;
          });
      });
  }

  // Resolve session identifier to actual file path
  private resolveSessionFile(session: string): ResultAsync<{ sessionId: string, filePath: string }, { message: string }> {
    if (session === 'latest') {
      return Logger.getLatestSessionFile()
        .andThen(filePath => {
          // Extract session ID from file path
          const match = filePath.match(/argonaut-session-(.+)\.log$/);
          if (!match) {
            return err({ message: 'Could not extract session ID from latest log file' });
          }
          return ok({ sessionId: match[1], filePath });
        });
    }

    // Treat as specific session ID
    const filePath = Logger.getSessionFilePath(session);
    return ok({ sessionId: session, filePath });
  }

  // Spawn the actual tail process with pino-pretty
  private spawnTailProcess(
    sessionId: string, 
    filePath: string, 
    options: { follow: boolean; lines: number }
  ): ResultAsync<TailProcess, { message: string }> {
    return ResultAsync.fromPromise(
      new Promise<TailProcess>((resolve, reject) => {
        try {
          // Build tail command arguments
          const tailArgs = ['-n', options.lines.toString()];
          if (options.follow) {
            tailArgs.push('-f');
          }
          tailArgs.push(filePath);

          // Use shell piping instead of stdio piping to avoid Bun issues
          // Use npx to ensure we use the local pino-pretty installation
          const pipeCommand = `tail ${tailArgs.join(' ')} | npx pino-pretty --colorize --translateTime SYS:HH:MM:ss --ignore hostname,pid`;
          
          const shellProcess = spawn('sh', ['-c', pipeCommand], {
            stdio: ['ignore', process.stdout, process.stderr]
          });

          // Handle errors
          shellProcess.on('error', (error) => {
            reject(new Error(`Failed to start shell process: ${error.message}`));
          });

          // Handle process exits
          shellProcess.on('exit', (code, signal) => {
            if (code !== 0 && signal !== 'SIGTERM') {
              console.error(`Shell process exited with code ${code}, signal ${signal}`);
            }
          });

          const tailProcessInfo: TailProcess = {
            process: shellProcess,
            sessionId,
            filePath
          };

          this.tailProcess = tailProcessInfo;
          
          // Give process a moment to start up
          setTimeout(() => resolve(tailProcessInfo), 100);

        } catch (error: any) {
          reject(new Error(`Failed to create tail process: ${error.message}`));
        }
      }),
      (error: any) => ({ message: error?.message || 'Failed to spawn tail process' })
    );
  }

  // Start watching for new sessions when tailing 'latest'
  private startSessionWatching(): void {
    if (this.sessionWatchInterval) {
      return; // Already watching
    }

    const currentSessionId = this.tailProcess?.sessionId;
    if (!currentSessionId) return;

    // Check for new sessions every 2 seconds
    this.sessionWatchInterval = setInterval(async () => {
      try {
        const latestResult = await Logger.getLatestSessionFile();
        if (latestResult.isErr()) {
          return; // Continue with current session
        }

        const latestPath = latestResult.value;
        const match = latestPath.match(/argonaut-session-(.+)\.log$/);
        if (!match) {
          return;
        }

        const latestSessionId = match[1];
        
        // If we found a newer session, switch to it
        if (latestSessionId !== currentSessionId && existsSync(latestPath)) {
          console.log(`\n🔄 New session detected: ${latestSessionId}`);
          console.log('Switching to follow new session...\n');
          
          // Stop current process and start new one
          this.stopSessionWatching();
          if (this.tailProcess) {
            this.tailProcess.process.kill('SIGTERM');
            this.tailProcess = null;
          }

          // Start tailing 'latest' again to keep watching for newer sessions
          setTimeout(() => {
            this.start({ session: 'latest', follow: true })
              .mapErr(error => {
                console.error(`❌ Failed to switch to new session: ${error.message}`);
              });
          }, 500);

          return;
        }
      } catch (error) {
        // Silently continue - don't spam errors
      }
    }, 2000);
  }

  // Stop watching for new sessions
  private stopSessionWatching(): void {
    if (this.sessionWatchInterval) {
      clearInterval(this.sessionWatchInterval);
      this.sessionWatchInterval = null;
    }
  }
}

async function main() {
  const args = process.argv.slice(2);
  const sessionArg = args.find(arg => arg.startsWith('--session='));
  const session = sessionArg ? sessionArg.split('=')[1] : 'latest';
  
  console.log(`📋 Tailing logs for session: ${session === 'latest' ? 'latest' : session}`);
  console.log('Press \'q\' or Ctrl+C to stop tailing...\n');
  
  const tailer = new DevLogTailer();
  const result = await tailer.start({ session });
  
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
}

main().catch(console.error);