import React, { useEffect, useState } from 'react';
import { Box, Text, useInput } from 'ink';
import type { ServerImportStatus } from '../types/argonaut';
import type { ArgonautServerConfig } from '../types/argonaut';
import { detectNewServers, saveServerConfig, createDefaultServerConfig } from '../config/argonaut-config';
import { ServerConnectivity, ConnectivityResult } from './ServerConnectivity';
import { ServerForm, argoServerFields, argonautFields, isFieldDisabled } from './ServerFormFields';
import { ScrollBox } from './ScrollBox';
import LoadingView from './LoadingView';

interface ImportViewProps {
  termRows: number;
  onComplete: () => void;
}

type ImportMode = 'detecting' | 'overview' | 'configuring' | 'testing';

const ImportView: React.FC<ImportViewProps> = ({ termRows, onComplete }) => {
  const [mode, setMode] = useState<ImportMode>('detecting');
  const [servers, setServers] = useState<ServerImportStatus[]>([]);
  const [currentServerIndex, setCurrentServerIndex] = useState(0);
  const [currentForm, setCurrentForm] = useState<ArgonautServerConfig | null>(null);
  const [currentField, setCurrentField] = useState(0);
  const [inputMode, setInputMode] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const [scrollOffset, setScrollOffset] = useState(0);
  const [connectivityResult, setConnectivityResult] = useState<ConnectivityResult | null>(null);
  const [importedCount, setImportedCount] = useState(0);

  const allFields = [...argoServerFields, ...argonautFields];

  // Detect servers on mount
  useEffect(() => {
    (async () => {
      const detected = await detectNewServers();
      const newServers = detected.filter(s => s.isNew);
      
      if (newServers.length === 0) {
        // No new servers to import
        onComplete();
        return;
      }
      
      setServers(newServers);
      setMode('overview');
    })();
  }, []);

  // Auto-scroll to keep current field visible
  useEffect(() => {
    if (mode !== 'configuring') return;
    
    const availableHeight = Math.max(3, termRows - 11);
    
    if (currentField < scrollOffset) {
      setScrollOffset(currentField);
    } else if (currentField >= scrollOffset + availableHeight) {
      setScrollOffset(currentField - availableHeight + 1);
    }
  }, [currentField, termRows, mode]);

  useInput((input, key) => {
    if (mode === 'detecting') {
      if (input.toLowerCase() === 'q') {
        onComplete();
        return;
      }
      return;
    }

    if (mode === 'overview') {
      if (input.toLowerCase() === 'q' || key.escape) {
        onComplete();
        return;
      }
      if (input.toLowerCase() === 'y' || key.return) {
        startImporting();
        return;
      }
      if (input.toLowerCase() === 'n') {
        onComplete();
        return;
      }
      return;
    }

    if (mode === 'testing') {
      if (input.toLowerCase() === 'q' || key.escape) {
        setMode('configuring');
        return;
      }
      if (input.toLowerCase() === 's') {
        saveCurrentServer(true);
        return;
      }
      if (input.toLowerCase() === 'r') {
        setMode('configuring');
        return;
      }
      if (input.toLowerCase() === 'c') {
        nextServer();
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

    if (mode === 'configuring') {
      if (input.toLowerCase() === 'q' || key.escape) {
        setMode('overview');
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
        saveCurrentServer(false);
        return;
      }
    }
  });

  const startImporting = () => {
    if (servers.length === 0) {
      onComplete();
      return;
    }
    
    setCurrentServerIndex(0);
    setupServerForm(0);
    setMode('configuring');
  };

  const setupServerForm = (serverIndex: number) => {
    const server = servers[serverIndex];
    const form = createDefaultServerConfig(server.serverUrl);
    
    // Apply ArgoCD config if available
    if (server.argoConfig) {
      form.contextName = server.argoConfig.contextName || '';
      form.insecure = server.argoConfig.insecure || false;
      form.grpcWeb = server.argoConfig.grpcWeb || false;
      form.grpcWebRootPath = server.argoConfig.grpcWebRootPath || '';
      form.plaintext = server.argoConfig.plaintext || false;
      form.core = server.argoConfig.core || false;
    }
    
    setCurrentForm(form);
    setCurrentField(0);
    setScrollOffset(0);
    setConnectivityResult(null);
  };

  const saveCurrentServer = async (skipConnectivity: boolean) => {
    if (!currentForm) return;
    
    if (!skipConnectivity && (!connectivityResult || !connectivityResult.success)) {
      setMode('testing');
      return;
    }
    
    // Mark as imported and save
    const serverToSave = {
      ...currentForm,
      imported: true,
      importedAt: new Date().toISOString(),
      lastConnected: connectivityResult?.success ? new Date().toISOString() : undefined,
    };
    
    await saveServerConfig(serverToSave);
    setImportedCount(prev => prev + 1);
    
    nextServer();
  };

  const nextServer = () => {
    if (currentServerIndex + 1 >= servers.length) {
      onComplete();
      return;
    }
    
    setCurrentServerIndex(prev => prev + 1);
    setupServerForm(currentServerIndex + 1);
    setMode('configuring');
  };

  if (mode === 'detecting') {
    return (
      <LoadingView
        termRows={termRows}
        message="Detecting ArgoCD servers..."
        showHeader={false}
        showAbort={false}
      />
    );
  }

  if (mode === 'overview') {
    return (
      <Box flexDirection="column" height={termRows - 1}>
        <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="cyan" paddingX={1}>
          <Box flexDirection="column" paddingX={1} paddingY={1}>
            <Text bold>📥 Import ArgoCD Servers</Text>
            <Box marginTop={1}>
              <Text dimColor>
                Found {servers.length} new server{servers.length !== 1 ? 's' : ''} in your ArgoCD config:
              </Text>
            </Box>
            
            <Box marginTop={1} flexDirection="column">
              {servers.map((server, index) => (
                <Box key={server.serverUrl} marginY={0}>
                  <Text>• <Text color="cyan">{server.serverUrl}</Text></Text>
                </Box>
              ))}
            </Box>
            
            <Box marginTop={2}>
              <Text>
                Do you want to import these servers into Argonaut? 
                You'll be able to configure connection settings for each one.
              </Text>
            </Box>
          </Box>
        </Box>
        
        <Box paddingLeft={1}>
          <Text dimColor>
            <Text color="green" bold>y</Text> to import • <Text color="red" bold>n</Text> to skip • q to quit
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
                ({currentServerIndex + 1}/{servers.length})
              </Text>
            </Box>
            
            <ServerConnectivity
              serverConfig={currentForm}
              onResult={setConnectivityResult}
              autoTest={true}
            />
            
            {connectivityResult && !connectivityResult.success && (
              <Box marginTop={1}>
                <Text color="yellow">
                  Connection failed. You can still save this server configuration
                  and fix the connection issues later in settings.
                </Text>
              </Box>
            )}
          </Box>
        </Box>
        
        <Box paddingLeft={1}>
          <Text dimColor>
            <Text color="green" bold>s</Text> to save anyway • r to return to config • c to continue • q to quit
          </Text>
        </Box>
      </Box>
    );
  }

  if (mode === 'configuring' && currentForm) {
    return (
      <Box flexDirection="column" height={termRows - 1}>
        <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}>
          <Box flexDirection="column" paddingX={1} paddingY={1}>
            <Text bold>
              ⚙️ Configure Server ({currentServerIndex + 1}/{servers.length})
            </Text>
            <Box marginTop={1}>
              <Text dimColor>
                Server: <Text color="cyan">{currentForm.serverUrl}</Text>
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

export default ImportView;