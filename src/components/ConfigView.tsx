import React, { useEffect, useState } from 'react';
import { Box, Text, useInput } from 'ink';
import type { ArgonautServerConfig } from '../types/argonaut';
import { readArgonautConfig, saveServerConfig, removeServerConfig, createDefaultServerConfig } from '../config/argonaut-config';
import { ServerConnectivity, ConnectivityResult, ConnectivityStatus } from './ServerConnectivity';
import { argoServerFields, argonautFields, isFieldDisabled } from './ServerFormFields';
import { ScrollBox } from './ScrollBox';
import LoadingView from './LoadingView';

interface ConfigViewProps {
  termRows: number;
  onClose: () => void;
}

type ConfigMode = 'loading' | 'list' | 'editing' | 'testing';

const ConfigView: React.FC<ConfigViewProps> = ({ termRows, onClose }) => {
  const [mode, setMode] = useState<ConfigMode>('loading');
  const [servers, setServers] = useState<ArgonautServerConfig[]>([]);
  const [selectedServerIndex, setSelectedServerIndex] = useState(0);
  const [currentForm, setCurrentForm] = useState<ArgonautServerConfig | null>(null);
  const [currentField, setCurrentField] = useState(0);
  const [inputMode, setInputMode] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const [scrollOffset, setScrollOffset] = useState(0);
  const [connectivityResult, setConnectivityResult] = useState<ConnectivityResult | null>(null);

  const allFields = [...argoServerFields, ...argonautFields];

  // Load servers on mount
  useEffect(() => {
    (async () => {
      const config = await readArgonautConfig();
      if (config) {
        setServers(config.servers);
      }
      setMode('list');
    })();
  }, []);

  // Auto-scroll to keep current field visible
  useEffect(() => {
    if (mode !== 'editing') return;
    
    const availableHeight = Math.max(3, termRows - 11);
    
    if (currentField < scrollOffset) {
      setScrollOffset(currentField);
    } else if (currentField >= scrollOffset + availableHeight) {
      setScrollOffset(currentField - availableHeight + 1);
    }
  }, [currentField, termRows, mode]);

  useInput((input, key) => {
    if (mode === 'loading') {
      if (input.toLowerCase() === 'q') {
        onClose();
        return;
      }
      return;
    }

    if (mode === 'list') {
      if (input.toLowerCase() === 'q' || key.escape) {
        onClose();
        return;
      }
      
      if (input === 'j' || key.downArrow) {
        setSelectedServerIndex(prev => Math.min(prev + 1, servers.length - 1));
        return;
      }
      
      if (input === 'k' || key.upArrow) {
        setSelectedServerIndex(prev => Math.max(prev - 1, 0));
        return;
      }

      if (key.return || input === 'e') {
        editServer();
        return;
      }

      if (input === 'd') {
        deleteServer();
        return;
      }

      if (input === 'n') {
        addNewServer();
        return;
      }
      
      return;
    }

    if (mode === 'testing') {
      if (input.toLowerCase() === 'q' || key.escape) {
        setMode('editing');
        return;
      }
      if (input.toLowerCase() === 's') {
        saveCurrentServer();
        return;
      }
      if (input.toLowerCase() === 'r') {
        setMode('editing');
        return;
      }
      return;
    }

    if (inputMode) {
      if (key.return) {
        const field = allFields[currentField];
        setCurrentForm(prev => prev ? { ...prev, [field.key]: inputValue } : null);
        setInputMode(false);
        setInputValue('');
        return;
      }
      if (key.escape) {
        setInputMode(false);
        setInputValue('');
        return;
      }
      if (key.backspace || key.delete) {
        setInputValue(prev => prev.slice(0, -1));
        return;
      }
      if (input) {
        setInputValue(prev => prev + input);
        return;
      }
    }

    if (mode === 'editing') {
      if (input.toLowerCase() === 'q' || key.escape) {
        setMode('list');
        return;
      }
      
      if (input === 'j' || key.downArrow) {
        setCurrentField(prev => {
          let next = prev + 1;
          while (next < allFields.length && currentForm && isFieldDisabled(allFields[next].key as any, currentForm)) {
            next++;
          }
          return Math.min(next, allFields.length - 1);
        });
        return;
      }
      
      if (input === 'k' || key.upArrow) {
        setCurrentField(prev => {
          let next = prev - 1;
          while (next >= 0 && currentForm && isFieldDisabled(allFields[next].key as any, currentForm)) {
            next--;
          }
          return Math.max(next, 0);
        });
        return;
      }

      if (key.return || input === ' ') {
        if (!currentForm) return;
        
        const field = allFields[currentField];
        const disabled = isFieldDisabled(field.key as any, currentForm);
        
        if (disabled) return;
        
        if (field.type === 'toggle') {
          setCurrentForm(prev => prev ? { ...prev, [field.key]: !prev[field.key as keyof ArgonautServerConfig] } : null);
        } else {
          setInputValue(String(currentForm[field.key as keyof ArgonautServerConfig] || ''));
          setInputMode(true);
        }
        return;
      }

      if (input.toLowerCase() === 't') {
        setMode('testing');
        return;
      }

      if (input.toLowerCase() === 's') {
        saveCurrentServer();
        return;
      }
    }
  });

  const editServer = () => {
    if (servers.length === 0) return;
    
    setCurrentForm(servers[selectedServerIndex]);
    setCurrentField(0);
    setScrollOffset(0);
    setConnectivityResult(null);
    setMode('editing');
  };

  const addNewServer = () => {
    const newServer = createDefaultServerConfig('');
    setCurrentForm(newServer);
    setCurrentField(0); // Start at serverUrl field
    setScrollOffset(0);
    setConnectivityResult(null);
    setMode('editing');
  };

  const deleteServer = async () => {
    if (servers.length === 0) return;
    
    const serverToDelete = servers[selectedServerIndex];
    await removeServerConfig(serverToDelete.serverUrl);
    
    // Reload servers
    const config = await readArgonautConfig();
    if (config) {
      setServers(config.servers);
      setSelectedServerIndex(prev => Math.min(prev, config.servers.length - 1));
    }
  };

  const saveCurrentServer = async () => {
    if (!currentForm) return;
    
    // Check if server URL is provided
    if (!currentForm.serverUrl.trim()) {
      // Could show error message here
      return;
    }
    
    const serverToSave = {
      ...currentForm,
      imported: true,
      importedAt: currentForm.importedAt || new Date().toISOString(),
      lastConnected: connectivityResult?.success ? new Date().toISOString() : currentForm.lastConnected,
    };
    
    await saveServerConfig(serverToSave);
    
    // Reload servers
    const config = await readArgonautConfig();
    if (config) {
      setServers(config.servers);
      // Update selected index to the saved server
      const savedIndex = config.servers.findIndex(s => s.serverUrl === serverToSave.serverUrl);
      if (savedIndex >= 0) {
        setSelectedServerIndex(savedIndex);
      }
    }
    
    setMode('list');
  };

  if (mode === 'loading') {
    return (
      <LoadingView
        termRows={termRows}
        message="Loading configuration..."
        showHeader={false}
        showAbort={false}
      />
    );
  }

  if (mode === 'list') {
    return (
      <Box flexDirection="column" height={termRows - 1}>
        <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="cyan" paddingX={1}>
          <Box flexDirection="column" paddingX={1} paddingY={1}>
            <Text bold>⚙️ Argonaut Configuration</Text>
            <Box marginTop={1}>
              <Text dimColor>
                {servers.length === 0 ? 'No servers configured' : `${servers.length} server${servers.length !== 1 ? 's' : ''} configured`}
              </Text>
            </Box>
            
            {servers.length > 0 && (
              <Box marginTop={1} flexDirection="column">
                {servers.map((server, index) => (
                  <Box 
                    key={server.serverUrl} 
                    backgroundColor={selectedServerIndex === index ? 'magentaBright' : undefined}
                    paddingX={1}
                    marginY={0}
                  >
                    <Box flexGrow={1}>
                      <Text>
                        <Text color="cyan">{server.serverUrl}</Text>
                        {server.contextName && <Text dimColor> ({server.contextName})</Text>}
                      </Text>
                    </Box>
                    <Box paddingLeft={2}>
                      <ConnectivityStatus result={null} compact={true} />
                    </Box>
                  </Box>
                ))}
              </Box>
            )}
            
            {servers.length === 0 && (
              <Box marginTop={2}>
                <Text dimColor>
                  No servers found. Press 'n' to add a new server or run the import flow
                  to detect servers from your ArgoCD configuration.
                </Text>
              </Box>
            )}
          </Box>
        </Box>
        
        <Box paddingLeft={1}>
          <Text dimColor>
            {servers.length > 0 && (
              <>j/k navigate • <Text color="green" bold>Enter/e</Text> edit • </>
            )}
            <Text color="green" bold>n</Text> add new • 
            {servers.length > 0 && (
              <> <Text color="red" bold>d</Text> delete • </>
            )}
            q quit
          </Text>
        </Box>
      </Box>
    );
  }

  if (mode === 'testing' && currentForm) {
    return (
      <Box flexDirection="column" height={termRows - 1}>
        <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="yellow" paddingX={1}>
          <Box flexDirection="column" paddingX={1} paddingY={1}>
            <Text bold>🔍 Testing Connection</Text>
            <Box marginTop={1}>
              <Text>
                Server: <Text color="cyan">{currentForm.serverUrl}</Text>
              </Text>
            </Box>
            
            <ServerConnectivity
              serverConfig={currentForm}
              onResult={setConnectivityResult}
              autoTest={true}
            />
          </Box>
        </Box>
        
        <Box paddingLeft={1}>
          <Text dimColor>
            <Text color="green" bold>s</Text> save • r return to edit • q back to list
          </Text>
        </Box>
      </Box>
    );
  }

  if (mode === 'editing' && currentForm) {
    return (
      <Box flexDirection="column" height={termRows - 1}>
        <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}>
          <Box flexDirection="column" paddingX={1} paddingY={1}>
            <Text bold>
              ✏️ Edit Server Configuration
            </Text>
            <Box marginTop={1}>
              <Text dimColor>
                {currentForm.serverUrl ? (
                  <>Server: <Text color="cyan">{currentForm.serverUrl}</Text></>
                ) : (
                  'New server configuration'
                )}
              </Text>
            </Box>
            
            <ScrollBox
              marginTop={1}
              flexGrow={1}
              offset={scrollOffset}
              initialHeight={Math.max(3, termRows - 11)}
            >
              {allFields.map((field, index) => {
                const isActive = currentField === index;
                const value = currentForm[field.key as keyof ArgonautServerConfig];
                const disabled = isFieldDisabled(field.key as any, currentForm);
                
                // Show section headers
                if (field === argonautFields[0]) {
                  return (
                    <React.Fragment key={`${field.key as string}-section`}>
                      <Box height={1} />
                      <Text bold color="green">Argonaut</Text>
                      <Box key={field.key as string}>
                        <Box backgroundColor={isActive && !disabled ? 'magentaBright' : undefined} paddingX={1}>
                          <Box paddingRight={2}>
                            <Text color={disabled ? 'gray' : undefined}>{field.label}:</Text>
                          </Box>
                          <Box flexGrow={1} paddingLeft={1}>
                            {field.type === 'toggle' ? (
                              <Text color={disabled ? 'gray' : (value ? 'green' : 'red')}>
                                {disabled ? '— disabled' : (value ? '✓ enabled' : '✗ disabled')}
                              </Text>
                            ) : field.type === 'password' ? (
                              <Text color={disabled ? 'gray' : undefined}>
                                {disabled ? '— disabled' : (String(value).replace(/./g, '*') || '—')}
                              </Text>
                            ) : (
                              <Text color={disabled ? 'gray' : undefined}>
                                {disabled ? '— disabled' : (String(value) || '—')}
                              </Text>
                            )}
                          </Box>
                        </Box>
                      </Box>
                    </React.Fragment>
                  );
                }
                
                // Show "Argo CD Server" header for first field
                if (field === argoServerFields[0]) {
                  return (
                    <React.Fragment key={`${field.key as string}-section`}>
                      <Text bold color="green">Argo CD Server</Text>
                      <Box key={field.key as string}>
                        <Box backgroundColor={isActive && !disabled ? 'magentaBright' : undefined} paddingX={1}>
                          <Box paddingRight={2}>
                            <Text color={disabled ? 'gray' : undefined}>{field.label}:</Text>
                          </Box>
                          <Box flexGrow={1} paddingLeft={1}>
                            {field.type === 'toggle' ? (
                              <Text color={disabled ? 'gray' : (value ? 'green' : 'red')}>
                                {disabled ? '— disabled' : (value ? '✓ enabled' : '✗ disabled')}
                              </Text>
                            ) : field.type === 'password' ? (
                              <Text color={disabled ? 'gray' : undefined}>
                                {disabled ? '— disabled' : (String(value).replace(/./g, '*') || '—')}
                              </Text>
                            ) : (
                              <Text color={disabled ? 'gray' : undefined}>
                                {disabled ? '— disabled' : (String(value) || '—')}
                              </Text>
                            )}
                          </Box>
                        </Box>
                      </Box>
                    </React.Fragment>
                  );
                }
                
                return (
                  <Box key={field.key as string}>
                    <Box backgroundColor={isActive && !disabled ? 'magentaBright' : undefined} paddingX={1}>
                      <Box paddingRight={2}>
                        <Text color={disabled ? 'gray' : undefined}>{field.label}:</Text>
                      </Box>
                      <Box flexGrow={1} paddingLeft={1}>
                        {field.type === 'toggle' ? (
                          <Text color={disabled ? 'gray' : (value ? 'green' : 'red')}>
                            {disabled ? '— disabled' : (value ? '✓ enabled' : '✗ disabled')}
                          </Text>
                        ) : field.type === 'password' ? (
                          <Text color={disabled ? 'gray' : undefined}>
                            {disabled ? '— disabled' : (String(value).replace(/./g, '*') || '—')}
                          </Text>
                        ) : (
                          <Text color={disabled ? 'gray' : undefined}>
                            {disabled ? '— disabled' : (String(value) || '—')}
                          </Text>
                        )}
                      </Box>
                    </Box>
                  </Box>
                );
              })}
            </ScrollBox>

            {inputMode && (
              <Box marginTop={1} borderStyle="single" borderColor="cyan" paddingX={1}>
                <Text color="cyan">
                  {allFields[currentField].label}: {inputValue}
                </Text>
              </Box>
            )}
          </Box>
        </Box>
        
        <Box paddingLeft={1}>
          <Text dimColor>
            j/k navigate • Enter/Space edit • <Text color="green" bold>t</Text> test • <Text color="green" bold>s</Text> save • q back
          </Text>
        </Box>
      </Box>
    );
  }

  return null;
};

export default ConfigView;