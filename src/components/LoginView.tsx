import React, {useEffect, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import {readCLIConfig} from '../config/cli-config';
import ConfirmationBox from './ConfirmationBox';
import LoadingView from './LoadingView';
import {ScrollBox} from './ScrollBox';
import {exec} from 'child_process';
import {promisify} from 'util';

const execAsync = promisify(exec);

interface LoginViewProps {
  server: string | null;
  termRows: number;
  onLoginSuccess: () => void;
}

interface LoginForm {
  serverUrl: string;
  contextName: string;
  username: string;
  password: string;
  sso: boolean;
  ssoPort: string;
  ssoLaunchBrowser: boolean;
  skipTestTls: boolean;
  insecure: boolean;
  grpcWeb: boolean;
  grpcWebRootPath: string;
  plaintext: boolean;
  core: boolean;
  saveSettings: boolean;
  autoRelogin: boolean;
}

type SubMode = 'form' | 'confirm' | 'loading' | 'error';

const LoginView: React.FC<LoginViewProps> = ({
  server,
  termRows,
  onLoginSuccess,
}) => {
  const [subMode, setSubMode] = useState<SubMode>('form');
  const [form, setForm] = useState<LoginForm>({
    serverUrl: server || '',
    contextName: '',
    username: '',
    password: '',
    sso: false,
    ssoPort: '8085',
    ssoLaunchBrowser: true,
    skipTestTls: false,
    insecure: false,
    grpcWeb: false,
    grpcWebRootPath: '',
    plaintext: false,
    core: false,
    saveSettings: true,
    autoRelogin: true,
  });
  const [currentField, setCurrentField] = useState(0);
  const [error, setError] = useState('');
  const [inputMode, setInputMode] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const [availableServers, setAvailableServers] = useState<string[]>([]);
  const [currentServerIndex, setCurrentServerIndex] = useState(0);
  
  // Vim-style navigation state for gg
  const [lastGPressed, setLastGPressed] = useState<number>(0);
  
  // Scrolling state
  const [scrollOffset, setScrollOffset] = useState(0);

  const argoServerFields = [
    { key: 'argo-header', label: 'Argo CD Server', type: 'header', required: false },
    { key: 'serverUrl', label: 'Server URL', type: 'input', required: true },
    { key: 'contextName', label: 'Context Name', type: 'input', required: false },
    { key: 'username', label: 'Username', type: 'input', required: false },
    { key: 'password', label: 'Password', type: 'password', required: false },
    { key: 'sso', label: 'SSO Login', type: 'toggle', required: false },
    { key: 'ssoPort', label: 'SSO Port', type: 'input', required: false },
    { key: 'ssoLaunchBrowser', label: 'Auto Launch Browser', type: 'toggle', required: false },
    { key: 'skipTestTls', label: 'Skip TLS Test', type: 'toggle', required: false },
    { key: 'insecure', label: 'Insecure', type: 'toggle', required: false },
    { key: 'grpcWeb', label: 'gRPC Web', type: 'toggle', required: false },
    { key: 'grpcWebRootPath', label: 'gRPC Web Root Path', type: 'input', required: false },
    { key: 'plaintext', label: 'Plain Text', type: 'toggle', required: false },
    { key: 'core', label: 'Core Mode', type: 'toggle', required: false },
  ];

  const argonautFields = [
    { key: 'argonaut-spacer', label: '', type: 'spacer', required: false },
    { key: 'argonaut-header', label: 'Argonaut', type: 'header', required: false },
    { key: 'saveSettings', label: 'Save Login Settings', type: 'toggle', required: false },
    { key: 'autoRelogin', label: 'Auto Login', type: 'toggle', required: false },
  ];

  const allFields = [...argoServerFields, ...argonautFields];

  const isFieldDisabled = (fieldKey: string): boolean => {
    // Header fields and spacers are not selectable/interactive
    if (fieldKey === 'argo-header' || fieldKey === 'argonaut-header' || fieldKey === 'argonaut-spacer') {
      return true;
    }
    
    // SSO mode disables username/password fields
    if (form.sso && (fieldKey === 'username' || fieldKey === 'password')) {
      return true;
    }
    
    // Core mode disables SSO and username/password fields
    if (form.core && (fieldKey === 'sso' || fieldKey === 'username' || fieldKey === 'password' || 
        fieldKey === 'ssoPort' || fieldKey === 'ssoLaunchBrowser')) {
      return true;
    }
    
    // SSO-specific options are disabled when SSO is off
    if (!form.sso && (fieldKey === 'ssoPort' || fieldKey === 'ssoLaunchBrowser')) {
      return true;
    }
    
    // gRPC Web Root Path requires gRPC Web to be enabled
    if (!form.grpcWeb && fieldKey === 'grpcWebRootPath') {
      return true;
    }
    
    // Auto Re-login requires Save Settings to be enabled
    if (!form.saveSettings && fieldKey === 'autoRelogin') {
      return true;
    }
    
    return false;
  };

  // Function to load configuration for a specific server
  const loadServerConfig = async (serverUrl: string) => {
    try {
      const config = await readCLIConfig();
      if (!config) return;

      // Find context that uses this server
      const context = config.contexts?.find(c => c.server === serverUrl);
      // Find server configuration
      const serverInfo = config.servers?.find(s => s.server === serverUrl);

      setForm(prev => ({
        ...prev,
        serverUrl: serverUrl,
        contextName: context?.name || '',
        insecure: serverInfo?.insecure || false,
        grpcWeb: serverInfo?.['grpc-web'] || false,
        grpcWebRootPath: serverInfo?.['grpc-web-root-path'] || '',
        plaintext: serverInfo?.['plain-text'] || false,
        core: serverInfo?.core || false,
        // Reset auth fields when switching servers
        username: '',
        password: '',
        sso: false,
        ssoPort: '8085',
        ssoLaunchBrowser: true,
        skipTestTls: false,
      }));
    } catch (e) {
      console.error('Failed to load server config:', e);
    }
  };

  // Load existing config on mount
  useEffect(() => {
    (async () => {
      try {
        const config = await readCLIConfig();
        if (config) {
          // Extract all available servers
          const servers = config.servers?.map(s => s.server) || [];
          const contexts = config.contexts?.map(c => c.server) || [];
          const allServers = Array.from(new Set([...servers, ...contexts])).filter(Boolean);
          setAvailableServers(allServers);

          const currentContext = config['current-context'];
          const context = config.contexts?.find(c => c.name === currentContext);

          if (context && allServers.length > 0) {
            const serverIndex = allServers.indexOf(context.server);
            setCurrentServerIndex(serverIndex >= 0 ? serverIndex : 0);
            
            // Load configuration for the current server
            await loadServerConfig(context.server);
          }
        }
      } catch (e) {
        console.error('Failed to load config:', e);
      }
    })();
  }, []);

  // Ensure current field is not disabled when form state changes
  useEffect(() => {
    if (isFieldDisabled(allFields[currentField]?.key)) {
      // Find the first non-disabled field
      const firstEnabledField = allFields.findIndex(field => !isFieldDisabled(field.key));
      if (firstEnabledField !== -1) {
        setCurrentField(firstEnabledField);
      }
    }
  }, [form.sso, form.core, form.grpcWeb, form.saveSettings, currentField]);

  // Auto-scroll to keep current field visible
  useEffect(() => {
    const confirmationBoxHeight = subMode === 'confirm' ? 6 : 0;
    const availableHeight = Math.max(3, termRows - 11 - confirmationBoxHeight);

    // If current field is above visible area, scroll up
    if (currentField < scrollOffset) {
      setScrollOffset(currentField);
    }
    // If current field is below visible area, scroll down
    else if (currentField >= scrollOffset + availableHeight) {
      setScrollOffset(currentField - availableHeight + 1 );
    }
  }, [currentField, termRows, subMode]);

  useInput((input, key) => {
    if (subMode === 'loading') {
      if (key.escape || input === 'q') {
        process.exit(0);
        return;
      }
      return;
    }

    if (inputMode) {
      if (key.return) {
        const field = allFields[currentField];
        setForm(prev => ({ ...prev, [field.key]: inputValue }));
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

    if (subMode === 'form') {
      if (key.escape || input === 'q') {
        process.exit(0);
        return;
      }
      if (input === 'j' || key.downArrow) {
        setCurrentField(prev => {
          let next = prev + 1;
          while (next < allFields.length && isFieldDisabled(allFields[next].key)) {
            next++;
          }
          return Math.min(next, allFields.length - 1);
        });
        return;
      }
      if (input === 'k' || key.upArrow) {
        setCurrentField(prev => {
          let next = prev - 1;
          while (next >= 0 && isFieldDisabled(allFields[next].key)) {
            next--;
          }
          return Math.max(next, 0);
        });
        return;
      }
      if (key.return || input === ' ') {
        const field = allFields[currentField];
        const disabled = isFieldDisabled(field.key);
        
        if (disabled) {
          return; // Ignore input for disabled fields
        }
        
        if (field.type === 'toggle') {
          setForm(prev => ({ ...prev, [field.key]: !prev[field.key as keyof LoginForm] }));
        } else {
          setInputValue(String(form[field.key as keyof LoginForm] || ''));
          setInputMode(true);
        }
        return;
      }
      if (input === 'l' || input === 'L') {
        if (!form.serverUrl.trim()) {
          setError('Server URL is required');
          return;
        }
        setSubMode('confirm');
        return;
      }
      if ((input === 's' || input === 'S') && availableServers.length > 1) {
        const nextIndex = (currentServerIndex + 1) % availableServers.length;
        setCurrentServerIndex(nextIndex);
        loadServerConfig(availableServers[nextIndex]);
        return;
      }
      
      // Vim-style navigation: gg to go to top, G to go to bottom
      if (input === 'g') {
        const now = Date.now();
        if (now - lastGPressed < 500) { // 500ms window for double g
          // Go to first enabled field
          const firstEnabledField = allFields.findIndex(field => !isFieldDisabled(field.key));
          if (firstEnabledField !== -1) {
            setCurrentField(firstEnabledField);
          }
        }
        setLastGPressed(now);
        return;
      }
      if (input === 'G') {
        // Go to last enabled field
        const enabledFields = allFields.map((field, index) => ({ field, index }))
          .filter(({ field }) => !isFieldDisabled(field.key));
        if (enabledFields.length > 0) {
          setCurrentField(enabledFields[enabledFields.length - 1].index);
        }
        return;
      }
    }

    if (subMode === 'confirm') {
      if (key.escape || input === 'q') {
        setSubMode('form');
        return;
      }
      // ConfirmationBox handles y/n
    }

    if (subMode === 'error') {
      // Any key returns to form
      setSubMode('form');
      setError('');
      return;
    }
  });

  const buildLoginCommand = (maskPassword = false): string => {
    const args: string[] = ['argocd', 'login', form.serverUrl];
    
    if (form.contextName.trim()) {
      args.push('--name', form.contextName.trim());
    }
    
    // Core mode excludes SSO and username/password authentication
    if (form.core) {
      args.push('--core');
    } else {
      // SSO authentication
      if (form.sso) {
        args.push('--sso');
        if (form.ssoPort.trim() && form.ssoPort !== '8085') {
          args.push('--sso-port', form.ssoPort.trim());
        }
        if (!form.ssoLaunchBrowser) {
          args.push('--sso-launch-browser=false');
        }
      } else {
        // Username/password authentication (only if not SSO)
        if (form.username.trim()) {
          args.push('--username', form.username.trim());
        }
        if (form.password.trim()) {
          args.push('--password', maskPassword ? '***' : form.password.trim());
        }
      }
    }
    
    // Connection options (available for all modes)
    if (form.skipTestTls) {
      args.push('--skip-test-tls');
    }
    if (form.insecure) {
      args.push('--insecure');
    }
    if (form.grpcWeb) {
      args.push('--grpc-web');
      if (form.grpcWebRootPath.trim()) {
        args.push('--grpc-web-root-path', form.grpcWebRootPath.trim());
      }
    }
    if (form.plaintext) {
      args.push('--plaintext');
    }

    return args.join(' ');
  };

  const executeLogin = async (confirm: boolean) => {
    setSubMode('loading');
    setError('');

    try {
      const command = buildLoginCommand();
      const { stdout, stderr } = await execAsync(command);
      
      if (stderr && !stderr.includes('Successfully logged in')) {
        throw new Error(stderr);
      }
      
      onLoginSuccess();
    } catch (e: any) {
      setError(e.message || String(e));
      setSubMode('error');
    }
  };

  const renderFormField = (field: typeof allFields[0], index: number) => {
    const isActive = currentField === index;
    const value = form[field.key as keyof LoginForm];
    const disabled = isFieldDisabled(field.key);
    
    // Special handling for spacer type (empty row)
    if (field.type === 'spacer') {
      return <Box key={field.key} height={1} />;
    }
    
    // Special handling for header type
    if (field.type === 'header') {
      return (
        <Box key={field.key}>
          <Text bold color="green">{field.label}</Text>
        </Box>
      );
    }
    
    return (
      <Box key={field.key}>
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
  };

  if (subMode === 'loading') {
    return (
      <LoadingView
        termRows={termRows}
        message="Loading..."
        showHeader={false}
        showAbort={true}
        onAbort={() => setSubMode('form')}
      />
    );
  }

  if (subMode === 'error') {
    return (
      <Box flexDirection="column" borderStyle="round" borderColor="red" paddingX={1} height={termRows - 1}>
        <Box flexGrow={1} alignItems="center" justifyContent="center" flexDirection="column">
          <Text color="red" bold>❌ Login Failed</Text>
          <Box height={1} />
          <Text color="red">{error}</Text>
          <Box height={1} />
          <Text dimColor>Press any key to return to login form</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" height={termRows - 1}>
      {subMode === 'confirm' && (
        <Box marginTop={1}>
          <ConfirmationBox
            title="Confirm login"
            message="Execute login command:"
            target={buildLoginCommand(true)}
            options={[]}
            onConfirm={(confirmed) => {
              if (confirmed) {
                executeLogin(true);
              } else {
                setSubMode('form');
              }
            }}
          />
        </Box>
      )}

      <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}>
        <Box flexDirection="column" flexGrow={1} paddingX={1} paddingY={1}>
        <Box flexDirection="row" justifyContent="space-between" alignItems="center">
          <Text bold>
            🔐 Argo CD Login
          </Text>
          {availableServers.length > 1 && (
            <Text dimColor>
              Server {currentServerIndex + 1}/{availableServers.length} • Press 's' to switch
            </Text>
          )}
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Configure your Argo CD server connection and authentication</Text>
        </Box>
        
        <ScrollBox
          marginTop={1}
          flexGrow={1}
          offset={scrollOffset}
          initialHeight={Math.max(3, termRows - 11 - (subMode === 'confirm' ? 6 : 0))}
        >
          {allFields.map((field, index) => 
            renderFormField(field, index)
          )}
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
        <Text wrap={"middle"} dimColor>
          j/k to navigate • Enter/Space to edit • <Text color="green" bold>l to login</Text> • {availableServers.length > 1 ? 's to switch server • ' : ''}q to quit
        </Text>
      </Box>
    </Box>
  );
};

export default LoginView;