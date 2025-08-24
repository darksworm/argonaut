import { useState, useEffect } from 'react';
import { Box, Text } from 'ink';
import { runLogViewerSession } from '../services/log-viewer';
import { log } from '../services/logger';

interface LogViewerProps {
  onClose: () => void;
}

export default function LogViewer({ onClose }: LogViewerProps) {
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(true);

  useEffect(() => {
    const startLogViewer = async () => {
      try {
        setStarting(false);
        
        await runLogViewerSession({
          onEnterExternal: () => {
            log.debug("Entering log viewer");
          }
        });

        onClose();

      } catch (err) {
        setError(`Failed to initialize log viewer: ${err instanceof Error ? err.message : String(err)}`);
        setStarting(false);
      }
    };

    startLogViewer();
  }, [onClose]);

  if (starting) {
    return (
      <Box flexDirection="column" justifyContent="center" alignItems="center" height="100%">
        <Text color="yellow">📋 Starting log viewer...</Text>
        <Text dimColor>Loading logs...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box flexDirection="column" justifyContent="center" alignItems="center" height="100%">
        <Text color="red">❌ Error: {error}</Text>
        <Text dimColor>Press any key to close</Text>
      </Box>
    );
  }

  // The PTY process handles all the UI, we just need a placeholder
  return (
    <Box height="100%">
      {/* This is rendered but invisible - the PTY process owns the terminal */}
    </Box>
  );
}
