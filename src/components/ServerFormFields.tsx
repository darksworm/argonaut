import React from 'react';
import { Box, Text } from 'ink';
import type { ArgonautServerConfig } from '../types/argonaut';

export interface FieldDefinition {
  key: keyof ArgonautServerConfig;
  label: string;
  type: 'input' | 'password' | 'toggle' | 'header' | 'spacer';
  required: boolean;
}

export const argoServerFields: FieldDefinition[] = [
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

export const argonautFields: FieldDefinition[] = [
  { key: 'saveSettings', label: 'Save Login Settings', type: 'toggle', required: false },
  { key: 'autoRelogin', label: 'Auto Login', type: 'toggle', required: false },
];

export function isFieldDisabled(fieldKey: keyof ArgonautServerConfig, form: ArgonautServerConfig): boolean {
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
}

interface ServerFormFieldProps {
  field: FieldDefinition;
  form: ArgonautServerConfig;
  isActive: boolean;
  onToggle: (key: keyof ArgonautServerConfig) => void;
  onEdit: (key: keyof ArgonautServerConfig, value: string) => void;
}

export const ServerFormField: React.FC<ServerFormFieldProps> = ({
  field,
  form,
  isActive,
  onToggle,
  onEdit,
}) => {
  const value = form[field.key];
  const disabled = isFieldDisabled(field.key, form);
  
  // Special handling for spacer type (empty row)
  if (field.type === 'spacer') {
    return <Box key={field.key as string} height={1} />;
  }
  
  // Special handling for header type
  if (field.type === 'header') {
    return (
      <Box key={field.key as string}>
        <Text bold color="green">{field.label}</Text>
      </Box>
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
};

interface ServerFormProps {
  form: ArgonautServerConfig;
  currentField: number;
  allFields: FieldDefinition[];
  onFieldChange: (key: keyof ArgonautServerConfig, value: any) => void;
  onEditField: (key: keyof ArgonautServerConfig, value: string) => void;
  showHeaders?: boolean;
}

export const ServerForm: React.FC<ServerFormProps> = ({
  form,
  currentField,
  allFields,
  onFieldChange,
  onEditField,
  showHeaders = true,
}) => {
  return (
    <>
      {showHeaders && <Text bold color="green">Argo CD Server</Text>}
      
      {allFields.map((field, index) => {
        const isArgoField = argoServerFields.includes(field as any);
        const isArgonautField = argonautFields.includes(field as any);
        
        // Show section headers
        if (showHeaders && isArgonautField && argoServerFields.includes(allFields[index - 1] as any)) {
          return (
            <React.Fragment key={`${field.key as string}-section`}>
              <Box height={1} />
              <Text bold color="green">Argonaut</Text>
              <ServerFormField
                field={field}
                form={form}
                isActive={currentField === index}
                onToggle={onFieldChange}
                onEdit={onEditField}
              />
            </React.Fragment>
          );
        }
        
        return (
          <ServerFormField
            key={field.key as string}
            field={field}
            form={form}
            isActive={currentField === index}
            onToggle={onFieldChange}
            onEdit={onEditField}
          />
        );
      })}
    </>
  );
};

export function buildLoginCommand(form: ArgonautServerConfig, maskPassword = false): string {
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
}