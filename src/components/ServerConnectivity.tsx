import React, { useState, useEffect } from 'react';
import { Box, Text } from 'ink';
import { exec } from 'child_process';
import { promisify } from 'util';
import type { ArgonautServerConfig } from '../types/argonaut';
import { buildLoginCommand } from './ServerFormFields';

const execAsync = promisify(exec);

export interface ConnectivityResult {
  success: boolean;
  error?: string;
  message: string;
}

interface ServerConnectivityProps {
  serverConfig: ArgonautServerConfig;
  onResult?: (result: ConnectivityResult) => void;
  autoTest?: boolean;
}

export const ServerConnectivity: React.FC<ServerConnectivityProps> = ({
  serverConfig,
  onResult,
  autoTest = false,
}) => {
  const [testing, setTesting] = useState(false);
  const [result, setResult] = useState<ConnectivityResult | null>(null);

  const testConnection = async (): Promise<ConnectivityResult> => {
    try {
      const command = buildLoginCommand(serverConfig);
      const { stdout, stderr } = await execAsync(command, { timeout: 10000 });
      
      if (stderr && !stderr.includes('Successfully logged in')) {
        return {
          success: false,
          error: stderr,
          message: `Connection failed: ${stderr}`,
        };
      }
      
      return {
        success: true,
        message: 'Connection successful',
      };
    } catch (e: any) {
      const errorMessage = e.message || String(e);
      
      // Check for common error types
      if (errorMessage.includes('timeout')) {
        return {
          success: false,
          error: errorMessage,
          message: 'Connection timeout - server may be unreachable',
        };
      }
      
      if (errorMessage.includes('certificate')) {
        return {
          success: false,
          error: errorMessage,
          message: 'TLS certificate error - try enabling "Skip TLS Test"',
        };
      }
      
      if (errorMessage.includes('unauthorized') || errorMessage.includes('authentication')) {
        return {
          success: false,
          error: errorMessage,
          message: 'Authentication failed - check credentials',
        };
      }
      
      return {
        success: false,
        error: errorMessage,
        message: `Connection failed: ${errorMessage}`,
      };
    }
  };

  const handleTest = async () => {
    setTesting(true);
    setResult(null);
    
    const testResult = await testConnection();
    setResult(testResult);
    setTesting(false);
    
    onResult?.(testResult);
  };

  useEffect(() => {
    if (autoTest && !testing && !result) {
      handleTest();
    }
  }, [autoTest]);

  if (testing) {
    return (
      <Box borderStyle="single" borderColor="yellow" paddingX={1} marginY={1}>
        <Box flexDirection="column">
          <Text color="yellow" bold>🔍 Testing Connection</Text>
          <Text dimColor>Connecting to {serverConfig.serverUrl}...</Text>
        </Box>
      </Box>
    );
  }

  if (result) {
    return (
      <Box 
        borderStyle="single" 
        borderColor={result.success ? 'green' : 'red'} 
        paddingX={1} 
        marginY={1}
      >
        <Box flexDirection="column">
          <Text color={result.success ? 'green' : 'red'} bold>
            {result.success ? '✅ Connection Test' : '❌ Connection Test'}
          </Text>
          <Text>{result.message}</Text>
          {result.error && (
            <Text dimColor wrap="wrap">
              Error details: {result.error}
            </Text>
          )}
        </Box>
      </Box>
    );
  }

  return (
    <Box borderStyle="single" borderColor="gray" paddingX={1} marginY={1}>
      <Box flexDirection="column">
        <Text bold>🔍 Connection Test</Text>
        <Text dimColor>Press 't' to test connection to {serverConfig.serverUrl}</Text>
      </Box>
    </Box>
  );
};

interface ConnectivityStatusProps {
  result: ConnectivityResult | null;
  compact?: boolean;
}

export const ConnectivityStatus: React.FC<ConnectivityStatusProps> = ({
  result,
  compact = false,
}) => {
  if (!result) {
    return compact ? (
      <Text dimColor>Untested</Text>
    ) : (
      <Text dimColor>Connection not tested</Text>
    );
  }

  const icon = result.success ? '✅' : '❌';
  const color = result.success ? 'green' : 'red';
  
  if (compact) {
    return <Text color={color}>{icon}</Text>;
  }

  return (
    <Box>
      <Text color={color}>{icon} {result.message}</Text>
    </Box>
  );
};