import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import TextInput from 'ink-text-input';
import type { ArgonautServerConfig } from '../types/argonaut';

interface PasswordPromptProps {
  termRows: number;
  serverConfig: ArgonautServerConfig;
  onSubmit: (credentials: { username?: string; password: string }) => void;
  onCancel: () => void;
}

const PasswordPrompt: React.FC<PasswordPromptProps> = ({
  termRows,
  serverConfig,
  onSubmit,
  onCancel,
}) => {
  const [username, setUsername] = useState(serverConfig.username || '');
  const [password, setPassword] = useState('');
  const [currentField, setCurrentField] = useState<'username' | 'password'>(
    serverConfig.username ? 'password' : 'username'
  );

  useInput((input, key) => {
    if (key.escape) {
      onCancel();
      return;
    }
    
    if (key.tab) {
      setCurrentField(prev => prev === 'username' ? 'password' : 'username');
      return;
    }
  });

  const handleUsernameSubmit = (value: string) => {
    setUsername(value);
    setCurrentField('password');
  };

  const handlePasswordSubmit = (value: string) => {
    setPassword(value);
    
    const credentials: { username?: string; password: string } = {
      password: value,
    };
    
    if (!serverConfig.username || username !== serverConfig.username) {
      credentials.username = username;
    }
    
    onSubmit(credentials);
  };

  const needsUsername = !serverConfig.username || currentField === 'username';

  return (
    <Box flexDirection="column" height={termRows - 1}>
      <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="yellow" paddingX={1}>
        <Box flexDirection="column" paddingX={1} paddingY={1}>
          <Text bold>🔐 Authentication Required</Text>
          <Box marginTop={1}>
            <Text>
              Server: <Text color="cyan">{serverConfig.serverUrl}</Text>
            </Text>
          </Box>
          
          {serverConfig.sso ? (
            <Box marginTop={1}>
              <Text color="yellow">
                This server uses SSO authentication. Please run:
              </Text>
              <Box marginTop={1}>
                <Text color="cyan">argocd login {serverConfig.serverUrl} --sso</Text>
              </Box>
            </Box>
          ) : serverConfig.core ? (
            <Box marginTop={1}>
              <Text color="yellow">
                This server uses core mode. Please run:
              </Text>
              <Box marginTop={1}>
                <Text color="cyan">argocd login {serverConfig.serverUrl} --core</Text>
              </Box>
            </Box>
          ) : (
            <Box flexDirection="column" marginTop={2}>
              {needsUsername && (
                <Box flexDirection="column" marginBottom={1}>
                  <Text color={currentField === 'username' ? 'cyan' : 'dimColor'}>
                    Username:
                  </Text>
                  {currentField === 'username' && (
                    <Box borderStyle="single" borderColor="cyan" paddingX={1}>
                      <TextInput
                        value={username}
                        onChange={setUsername}
                        onSubmit={handleUsernameSubmit}
                        placeholder="Enter username"
                      />
                    </Box>
                  )}
                  {currentField !== 'username' && username && (
                    <Box paddingX={1}>
                      <Text>{username}</Text>
                    </Box>
                  )}
                </Box>
              )}
              
              <Box flexDirection="column">
                <Text color={currentField === 'password' ? 'cyan' : 'dimColor'}>
                  Password:
                </Text>
                {currentField === 'password' && (
                  <Box borderStyle="single" borderColor="cyan" paddingX={1}>
                    <TextInput
                      value={password}
                      onChange={setPassword}
                      onSubmit={handlePasswordSubmit}
                      placeholder="Enter password"
                      mask="*"
                    />
                  </Box>
                )}
              </Box>
            </Box>
          )}
        </Box>
      </Box>
      
      <Box paddingLeft={1}>
        <Text dimColor>
          {serverConfig.sso || serverConfig.core ? (
            'Run the command above and restart the app • Esc to cancel'
          ) : (
            'Tab to switch fields • Enter to submit • Esc to cancel'
          )}
        </Text>
      </Box>
    </Box>
  );
};

export default PasswordPrompt;