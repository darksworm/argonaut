import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import YAML from 'yaml';
import type { ArgonautConfig, ArgonautServerConfig, ServerImportStatus } from '../types/argonaut';
import { readCLIConfig, type ArgoCLIConfig } from './cli-config';

const ARGONAUT_CONFIG_DIR = path.join(os.homedir(), '.config', 'argonaut');
const ARGONAUT_CONFIG_PATH = path.join(ARGONAUT_CONFIG_DIR, 'config');

export async function readArgonautConfig(): Promise<ArgonautConfig | null> {
  try {
    const txt = await fs.readFile(ARGONAUT_CONFIG_PATH, 'utf8');
    const config = YAML.parse(txt) as ArgonautConfig;
    
    // Validate version
    if (config.version !== 1) {
      console.warn(`Unsupported Argonaut config version: ${config.version}`);
      return null;
    }
    
    return config;
  } catch {
    return null;
  }
}

export async function writeArgonautConfig(config: ArgonautConfig): Promise<void> {
  await fs.mkdir(ARGONAUT_CONFIG_DIR, { recursive: true });
  await fs.writeFile(ARGONAUT_CONFIG_PATH, YAML.stringify(config), { mode: 0o600 });
}

export function createDefaultServerConfig(serverUrl: string): ArgonautServerConfig {
  return {
    serverUrl,
    imported: false,
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
  };
}

export async function detectNewServers(): Promise<ServerImportStatus[]> {
  const argoConfig = await readCLIConfig();
  const argonautConfig = await readArgonautConfig();
  
  if (!argoConfig) {
    return [];
  }
  
  // Get all unique server URLs from ArgoCD config
  const argoServers = new Set<string>();
  
  // Add servers from servers array
  argoConfig.servers?.forEach(server => {
    if (server.server) {
      argoServers.add(server.server);
    }
  });
  
  // Add servers from contexts
  argoConfig.contexts?.forEach(context => {
    if (context.server) {
      argoServers.add(context.server);
    }
  });
  
  // Check which servers are new
  const knownServers = new Set(
    argonautConfig?.servers?.map(s => s.serverUrl) || []
  );
  
  const importStatuses: ServerImportStatus[] = [];
  
  for (const serverUrl of argoServers) {
    const isNew = !knownServers.has(serverUrl);
    const currentConfig = argonautConfig?.servers?.find(s => s.serverUrl === serverUrl);
    
    // Extract ArgoCD config for this server
    const context = argoConfig.contexts?.find(c => c.server === serverUrl);
    const serverInfo = argoConfig.servers?.find(s => s.server === serverUrl);
    
    const argoConfigInfo = {
      contextName: context?.name || '',
      insecure: serverInfo?.insecure || false,
      grpcWeb: serverInfo?.['grpc-web'] || false,
      grpcWebRootPath: serverInfo?.['grpc-web-root-path'] || '',
      plaintext: serverInfo?.['plain-text'] || false,
      core: serverInfo?.core || false,
    };
    
    importStatuses.push({
      serverUrl,
      isNew,
      currentConfig,
      argoConfig: argoConfigInfo,
    });
  }
  
  return importStatuses;
}

export async function saveServerConfig(serverConfig: ArgonautServerConfig): Promise<void> {
  let config = await readArgonautConfig();
  
  if (!config) {
    config = {
      version: 1,
      servers: [],
    };
  }
  
  // Find existing server or add new one
  const existingIndex = config.servers.findIndex(s => s.serverUrl === serverConfig.serverUrl);
  
  if (existingIndex >= 0) {
    config.servers[existingIndex] = serverConfig;
  } else {
    config.servers.push(serverConfig);
  }
  
  await writeArgonautConfig(config);
}

export async function removeServerConfig(serverUrl: string): Promise<void> {
  const config = await readArgonautConfig();
  
  if (!config) return;
  
  config.servers = config.servers.filter(s => s.serverUrl !== serverUrl);
  
  await writeArgonautConfig(config);
}

export async function getServerConfig(serverUrl: string): Promise<ArgonautServerConfig | null> {
  const config = await readArgonautConfig();
  
  if (!config) return null;
  
  return config.servers.find(s => s.serverUrl === serverUrl) || null;
}