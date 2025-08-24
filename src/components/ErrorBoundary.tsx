import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { logReactError } from '../services/error-handler';
import LogViewer from './LogViewer';

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
  errorInfo?: React.ErrorInfo;
  showLogs: boolean;
}

interface ErrorBoundaryProps {
  children: React.ReactNode;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, showLogs: false };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    logReactError(error, errorInfo);
    this.setState({ error, errorInfo });
  }

  render() {
    if (this.state.hasError) {
      return (
        <ErrorDisplay 
          error={this.state.error!}
          showLogs={this.state.showLogs}
          onToggleLogs={(showLogs: boolean) => this.setState({ showLogs })}
        />
      );
    }

    return this.props.children;
  }
}

function ErrorDisplay({ 
  error, 
  showLogs, 
  onToggleLogs 
}: {
  error: Error;
  showLogs: boolean;
  onToggleLogs: (showLogs: boolean) => void;
}) {
  const [termRows, setTermRows] = useState(process.stdout.rows || 24);
  const [termCols, setTermCols] = useState(process.stdout.columns || 80);

  useEffect(() => {
    const onResize = () => {
      setTermRows(process.stdout.rows || 24);
      setTermCols(process.stdout.columns || 80);
    };
    
    process.stdout.on('resize', onResize);
    return () => process.stdout.off('resize', onResize);
  }, []);
  useInput((input, key) => {
    if (input === 'l' || input === 'L') {
      onToggleLogs(!showLogs);
    } else if (key.escape || input === 'q') {
      process.exit(1);
    }
  });

  if (showLogs) {
    return (
      <Box flexDirection="column" height={termRows}>
        <Box paddingX={1} marginBottom={1}>
          <Text color="red" bold>💥 Crash detected - showing logs</Text>
          <Text> • Press </Text>
          <Text color="cyan">L</Text>
          <Text> to toggle error details • </Text>
          <Text color="cyan">Q/Esc</Text>
          <Text> to exit</Text>
        </Box>
        
        <Box flexGrow={1}>
          <LogViewer 
            onClose={() => onToggleLogs(false)}
          />
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" height={termRows - 1}>
      <Box 
        flexDirection="column" 
        borderStyle="round" 
        borderColor="red" 
        paddingX={2}
        paddingY={1}
        flexGrow={1}
      >
        <Box justifyContent="center" marginBottom={1}>
          <Text color="red" bold>💥 Application Error</Text>
        </Box>
        
        <Box flexDirection="column" marginBottom={1}>
          <Text color="red">Something went wrong in the React application:</Text>
          <Text wrap="wrap">{error.message}</Text>
        </Box>

        {error.stack && (
          <Box flexDirection="column" marginBottom={1}>
            <Text color="gray" dimColor>Stack trace:</Text>
            <Text wrap="wrap" dimColor>{error.stack}</Text>
          </Box>
        )}

        <Box flexDirection="column">
          <Text dimColor>This error has been logged to the session logs.</Text>
        </Box>
      </Box>
      
      <Box paddingX={1} marginTop={1}>
        <Text>Press </Text>
        <Text color="cyan">L</Text>
        <Text dimColor> to view logs • </Text>
        <Text color="cyan">Q/Esc</Text>
        <Text dimColor> to exit</Text>
      </Box>
    </Box>
  );
}
