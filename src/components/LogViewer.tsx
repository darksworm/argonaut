import React, { useState, useEffect, useRef } from 'react';
import { Box, Text, useInput } from 'ink';
import { spawn, ChildProcess } from 'child_process';

interface LogViewerProps {
  onClose: () => void;
  termRows?: number;
  termCols?: number;
}

export default function LogViewer({ onClose }: LogViewerProps) {
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(true);
  const childProcessRef = useRef<ChildProcess | null>(null);

  useEffect(() => {
    const startLogViewer = async () => {
      try {
        // First, get the latest session file path
        const { Logger } = await import('../services/logger');
        const result = await Logger.getLatestSessionFile();
        
        if (result.isErr()) {
          setError(`No log files found: ${result.error.message}`);
          return;
        }
        
        const logFilePath = result.value;
        
        // Use less to view the log file with proper formatting
        // -R: raw control chars (for colors when using pino-pretty)
        // -S: chop long lines  
        // -F: quit if content fits on one screen
        // -X: no init/deinit sequences
        // +G: start at end of file
        const shellCommand = `cat "${logFilePath}" | npx pino-pretty | less -RSX +G`;
        
        const child = spawn('sh', ['-c', shellCommand], {
          stdio: 'inherit',
          env: { ...process.env },
        });

        childProcessRef.current = child;
        setStarting(false);

        child.on('error', (err) => {
          setError(`Failed to start log viewer: ${err.message}`);
        });

        child.on('exit', (code) => {
          // For less, exit code 0 or 1 are normal (user pressed 'q')
          if (code !== 0 && code !== null && code !== 1) { 
            setError(`Log viewer exited with code ${code}`);
          }
          // Always call onClose when less exits (whether by 'q' or any other reason)
          onClose();
        });

      } catch (err) {
        setError(`Failed to initialize log viewer: ${err instanceof Error ? err.message : String(err)}`);
      }
    };

    startLogViewer();

    return () => {
      if (childProcessRef.current && !childProcessRef.current.killed) {
        childProcessRef.current.kill('SIGTERM');
      }
    };
  }, [onClose]);

  // Handle keyboard input to close
  useInput((input, key) => {
    if (input === 'q' || key.escape) {
      if (childProcessRef.current && !childProcessRef.current.killed) {
        childProcessRef.current.kill();
      }
      onClose();
    }
  });

  if (starting) {
    return (
      <Box flexDirection="column" justifyContent="center" alignItems="center" height="100%">
        <Text color="yellow">📋 Starting log viewer...</Text>
        <Text dimColor>Press 'q' to cancel</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column" justifyContent="center" alignItems="center" height="100%">
        <Text color="red">❌ Error: {error}</Text>
        <Text dimColor>Press 'q' to close</Text>
      </Box>
    );
  }

  // The child process handles all the UI, we just need to provide a way to exit
  return (
    <Box height="100%">
      {/* This is rendered but invisible - the child process owns the terminal */}
    </Box>
  );
}