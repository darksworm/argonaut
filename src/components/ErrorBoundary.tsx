import { Box, Text, useInput } from "ink";
import React, { useEffect, useState } from "react";
import { logReactError } from "../services/error-handler";
import { runLogViewerSession } from "../services/log-viewer";

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
  errorInfo?: React.ErrorInfo;
  showLogs: boolean;
}

interface ErrorBoundaryProps {
  children: React.ReactNode;
}

export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, showLogs: false };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log side effects only; state is set via getDerivedStateFromError to avoid nested updates
    try {
      logReactError(error, errorInfo);
    } catch {}
    // Do not call setState here to prevent update loops
  }

  render() {
    if (this.state.hasError && this.state.error) {
      return (
        <ErrorDisplay
          error={this.state.error}
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
  onToggleLogs,
}: {
  error: Error;
  showLogs: boolean;
  onToggleLogs: (showLogs: boolean) => void;
}) {
  const [termRows, setTermRows] = useState(process.stdout.rows || 24);

  useEffect(() => {
    const onResize = () => {
      setTermRows(process.stdout.rows || 24);
    };

    process.stdout.on("resize", onResize);
    return () => {
      process.stdout.off("resize", onResize);
    };
  }, []);
  // Handle log viewer when showLogs changes
  useEffect(() => {
    const openLogs = async () => {
      if (showLogs) {
        try {
          await runLogViewerSession({
            title: "Error Logs",
          });

          // Small delay and close logs (state change will trigger re-render)
          await new Promise((resolve) => setTimeout(resolve, 50));
          onToggleLogs(false);
        } catch {
          try {
            const stdinAny = process.stdin as any;
            stdinAny.setRawMode?.(true);
            stdinAny.resume?.();
          } catch {}
          onToggleLogs(false);
        }
      }
    };

    openLogs();
  }, [showLogs, onToggleLogs]);

  useInput((input, key) => {
    if (input === "l" || input === "L") {
      if (!showLogs) {
        onToggleLogs(true);
      }
    } else if (key.escape || input === "q") {
      process.exit(1);
    }
  });

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
          <Text color="red" bold>
            💥 Application Error
          </Text>
        </Box>

        <Box flexDirection="column" marginBottom={1}>
          <Text color="red">
            Something went wrong in the React application:
          </Text>
          <Text wrap="wrap">{error.message}</Text>
        </Box>

        {error.stack && (
          <Box flexDirection="column" marginBottom={1}>
            <Text color="gray" dimColor>
              Stack trace:
            </Text>
            <Text wrap="wrap" dimColor>
              {error.stack}
            </Text>
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
