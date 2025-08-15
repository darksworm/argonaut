import React, {useEffect, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import chalk from 'chalk';

interface LoadingViewProps {
  termRows: number;
  message?: string;
  server?: string | null;
  showHeader?: boolean;
  showAbort?: boolean;
  onAbort?: () => void;
}

const LoadingView: React.FC<LoadingViewProps> = ({
  termRows,
  message = 'Loading...',
  server,
  showHeader = false,
  showAbort = false,
  onAbort,
}) => {
  const [spinnerFrame, setSpinnerFrame] = useState(0);
  const spinnerFrames = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];

  useEffect(() => {
    const interval = setInterval(() => {
      setSpinnerFrame(prev => (prev + 1) % spinnerFrames.length);
    }, 100);

    return () => clearInterval(interval);
  }, []);

  useInput((input, key) => {
    if (key.escape || input === 'q') {
      process.exit(0);
      return;
    }
    if (showAbort && (input === 'a' || input === 'A')) {
      onAbort?.();
      return;
    }
  });

  const header = showHeader 
    ? `${chalk.bold('View:')} ${chalk.yellow('LOADING')} • ${chalk.bold('Context:')} ${chalk.cyan(server || '—')}`
    : null;

  return (
    <Box flexDirection="column" height={termRows - 1}>
      <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}>
        {header && <Box><Text>{header}</Text></Box>}
        <Box flexGrow={1} alignItems="center" justifyContent="center" flexDirection="column">
          <Text color="yellow" bold>
            {spinnerFrames[spinnerFrame]}  {message.toUpperCase()}
          </Text>
        </Box>
      </Box>
      
      {showAbort && (
        <Box paddingLeft={1}>
          <Text dimColor>Press 'a' to abort and return to login</Text>
        </Box>
      )}
    </Box>
  );
};

export default LoadingView;