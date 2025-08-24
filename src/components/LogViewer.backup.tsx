import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { Logger, type LogEntry } from '../services/logger';
import chalk from 'chalk';

interface LogViewerProps {
  onClose: () => void;
  termRows?: number;
  termCols?: number;
}

const LOG_LEVEL_COLORS: Record<string, (text: string) => string> = {
  debug: chalk.gray,
  info: chalk.blue,
  warn: chalk.yellow,
  error: chalk.red,
};

const LOG_LEVEL_ICONS: Record<string, string> = {
  debug: '🔍',
  info: 'ℹ️ ',
  warn: '⚠️ ',
  error: '❌',
};

function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    });
  } catch {
    return timestamp.slice(0, 8); // Fallback to first 8 chars
  }
}

function LogLine({ log, maxWidth }: { log: LogEntry; maxWidth: number }) {
  const levelColor = LOG_LEVEL_COLORS[log.level] || chalk.white;
  const icon = LOG_LEVEL_ICONS[log.level] || '•';
  const time = formatTimestamp(log.timestamp);
  
  return (
    <Box width={maxWidth}>
      <Text color="gray">{time}</Text>
      <Text> {icon} </Text>
      <Text color={levelColor.name || 'white'}>{log.level.toUpperCase().padEnd(5)}</Text>
      {log.context && (
        <>
          <Text> [</Text>
          <Text color="cyan">{log.context}</Text>
          <Text>] </Text>
        </>
      )}
      <Text wrap="wrap">{log.message}</Text>
    </Box>
  );
}

export default function LogViewer({ onClose, termRows: propTermRows, termCols: propTermCols }: LogViewerProps) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [scrollOffset, setScrollOffset] = useState(0);
  const [currentSession, setCurrentSession] = useState<string>('current');
  const [availableSessions, setAvailableSessions] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  
  // Handle terminal resizing
  const [termRows, setTermRows] = useState(propTermRows || process.stdout.rows || 24);
  const [termCols, setTermCols] = useState(propTermCols || process.stdout.columns || 80);

  useEffect(() => {
    // Only set up resize listener if props aren't provided (standalone mode)
    if (propTermRows === undefined && propTermCols === undefined) {
      const onResize = () => {
        setTermRows(process.stdout.rows || 24);
        setTermCols(process.stdout.columns || 80);
      };
      
      process.stdout.on('resize', onResize);
      return () => process.stdout.off('resize', onResize);
    } else {
      // Update from props when they change
      setTermRows(propTermRows || process.stdout.rows || 24);
      setTermCols(propTermCols || process.stdout.columns || 80);
    }
  }, [propTermRows, propTermCols]);

  // Calculate viewport dimensions
  const headerRows = 3; // Title, session selector, separator
  const footerRows = 2; // Controls, border
  const viewportHeight = Math.max(5, termRows - headerRows - footerRows);
  const maxWidth = Math.max(40, termCols - 6); // Account for borders and padding

  // Load available sessions
  useEffect(() => {
    Logger.getAvailableSessions()
      .map(sessions => {
        setAvailableSessions(['current', ...sessions]);
        setLoading(false);
      })
      .mapErr(err => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  // Load logs for current session
  useEffect(() => {
    setLoading(true);
    
    if (currentSession === 'current') {
      // Get the current session by finding the latest session file
      Logger.getLatestSessionFile()
        .andThen(filePath => {
          // Extract session ID from file path
          const match = filePath.match(/argonaut-session-(.+)\.log$/);
          if (!match) {
            return Logger.readSessionLogs('current'); // This will fail gracefully
          }
          return Logger.readSessionLogs(match[1]);
        })
        .map(sessionLogs => {
          setLogs(sessionLogs);
          setScrollOffset(Math.max(0, sessionLogs.length - viewportHeight)); // Start at bottom
          setLoading(false);
          setError(null);
        })
        .mapErr(err => {
          setError(err.message);
          setLoading(false);
        });
      return;
    }

    Logger.readSessionLogs(currentSession)
      .map(sessionLogs => {
        setLogs(sessionLogs);
        setScrollOffset(Math.max(0, sessionLogs.length - viewportHeight)); // Start at bottom
        setLoading(false);
        setError(null);
      })
      .mapErr(err => {
        setError(err.message);
        setLoading(false);
      });
  }, [currentSession, viewportHeight]);

  // Handle keyboard input
  useInput((input, key) => {
    if (input === 'q' || key.escape) {
      onClose();
      return;
    }

    if (key.upArrow) {
      setScrollOffset(Math.max(0, scrollOffset - 1));
    } else if (key.downArrow) {
      setScrollOffset(Math.min(Math.max(0, logs.length - viewportHeight), scrollOffset + 1));
    } else if (key.pageUp) {
      setScrollOffset(Math.max(0, scrollOffset - viewportHeight));
    } else if (key.pageDown) {
      setScrollOffset(Math.min(Math.max(0, logs.length - viewportHeight), scrollOffset + viewportHeight));
    } else if (input === 'g') {
      setScrollOffset(0); // Go to top
    } else if (input === 'G') {
      setScrollOffset(Math.max(0, logs.length - viewportHeight)); // Go to bottom
    } else if (input === 'n' && availableSessions.length > 1) {
      // Next session
      const currentIndex = availableSessions.indexOf(currentSession);
      const nextIndex = (currentIndex + 1) % availableSessions.length;
      setCurrentSession(availableSessions[nextIndex]);
    } else if (input === 'p' && availableSessions.length > 1) {
      // Previous session
      const currentIndex = availableSessions.indexOf(currentSession);
      const prevIndex = currentIndex === 0 ? availableSessions.length - 1 : currentIndex - 1;
      setCurrentSession(availableSessions[prevIndex]);
    }
  });

  // Calculate visible logs
  const visibleLogs = logs.slice(scrollOffset, scrollOffset + viewportHeight);

  // Render loading state
  if (loading) {
    return (
      <Box flexDirection="column" height={termRows}>
        <Box 
          flexDirection="column" 
          borderStyle="round" 
          borderColor="yellow" 
          paddingX={1} 
          flexGrow={1}
        >
          <Box justifyContent="center">
            <Text color="yellow">📋 Loading Logs...</Text>
          </Box>
          <Box flexGrow={1} justifyContent="center" alignItems="center">
            <Text>Loading session: {currentSession}</Text>
          </Box>
        </Box>
        <Box justifyContent="center">
          <Text dimColor>Press 'q' to close</Text>
        </Box>
      </Box>
    );
  }

  // Render error state
  if (error) {
    return (
      <Box flexDirection="column" height={termRows}>
        <Box 
          flexDirection="column" 
          borderStyle="round" 
          borderColor="red" 
          paddingX={1} 
          flexGrow={1}
        >
          <Box justifyContent="center">
            <Text color="red">❌ Error Loading Logs</Text>
          </Box>
          <Box flexGrow={1} justifyContent="center" alignItems="center" flexDirection="column">
            <Text>{error}</Text>
            {availableSessions.length > 0 && (
              <Box marginTop={1}>
                <Text dimColor>Available sessions: {availableSessions.join(', ')}</Text>
              </Box>
            )}
          </Box>
        </Box>
        <Box justifyContent="center">
          <Text dimColor>Press 'q' to close • 'n'/'p' to switch sessions</Text>
        </Box>
      </Box>
    );
  }

  const scrollInfo = logs.length > 0 
    ? `${scrollOffset + 1}-${Math.min(scrollOffset + viewportHeight, logs.length)} of ${logs.length}`
    : '0 of 0';

  return (
    <Box flexDirection="column" height={termRows}>
      <Box 
        flexDirection="column" 
        borderStyle="round" 
        borderColor="cyan" 
        paddingX={1} 
        flexGrow={1}
      >
        {/* Header */}
        <Box justifyContent="space-between">
          <Text color="cyan">📋 Session Logs</Text>
          <Text color="gray">Session: {currentSession}</Text>
        </Box>
        
        <Box>
          <Text dimColor>{'─'.repeat(maxWidth)}</Text>
        </Box>

        {/* Log entries */}
        <Box flexDirection="column" flexGrow={1}>
          {visibleLogs.length === 0 ? (
            <Box justifyContent="center" alignItems="center" flexGrow={1}>
              <Text dimColor>
                {currentSession === 'current' 
                  ? 'No logs in current session yet'
                  : 'No logs found in this session'
                }
              </Text>
            </Box>
          ) : (
            visibleLogs.map((log, index) => (
              <LogLine key={scrollOffset + index} log={log} maxWidth={maxWidth} />
            ))
          )}
        </Box>
      </Box>

      {/* Footer - outside the border */}
      <Box justifyContent="space-between" paddingX={1}>
        <Text dimColor>
          ↑↓ Scroll • PgUp/PgDn Page • g/G Top/Bottom • n/p Sessions
        </Text>
        <Text dimColor>{scrollInfo}</Text>
      </Box>
      
      <Box justifyContent="center">
        <Text dimColor>Press 'q' or Esc to close</Text>
      </Box>
    </Box>
  );
}