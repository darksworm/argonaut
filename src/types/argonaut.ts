export interface ArgonautServerConfig {
  serverUrl: string;
  imported: boolean;
  importedAt?: string;
  lastConnected?: string;
  
  // ArgoCD connection settings
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
  
  // Argonaut settings
  saveSettings: boolean;
  autoRelogin: boolean;
}

export interface ArgonautConfig {
  version: number;
  servers: ArgonautServerConfig[];
}

export interface ServerImportStatus {
  serverUrl: string;
  isNew: boolean;
  currentConfig?: ArgonautServerConfig;
  argoConfig?: {
    contextName?: string;
    insecure?: boolean;
    grpcWeb?: boolean;
    grpcWebRootPath?: string;
    plaintext?: boolean;
    core?: boolean;
  };
}