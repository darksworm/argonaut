import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import {CONFIG_PATH, ensureHttps} from './paths';

export type ArgoContext = {name: string; server: string; user: string};
export type ArgoServer  = {server: string; ['grpc-web']?: boolean; ['grpc-web-root-path']?: string; insecure?: boolean; ['plain-text']?: boolean};
export type ArgoUser    = {name: string; ['auth-token']?: string};
export type ArgoCLIConfig = {
  contexts?: ArgoContext[];
  servers?: ArgoServer[];
  users?: ArgoUser[];
  ['current-context']?: string;
  ['prompts-enabled']?: boolean;
};

export async function readCLIConfig(): Promise<ArgoCLIConfig | null> {
  try {
    const txt = await fs.readFile(CONFIG_PATH, 'utf8');
    return YAML.parse(txt) as ArgoCLIConfig;
  } catch {
    return null;
  }
}

export async function writeCLIConfig(cfg: ArgoCLIConfig): Promise<void> {
  await fs.mkdir(path.dirname(CONFIG_PATH), {recursive: true});
  await fs.writeFile(CONFIG_PATH, YAML.stringify(cfg), {mode: 0o600});
}

export function getCurrentServer(cfg: ArgoCLIConfig | null): string | null {
  if (!cfg) return null;
  const name = cfg['current-context'];
  const ctx = (cfg.contexts ?? []).find(c => c.name === (name ?? ''));
  return ctx?.server ?? null;
}

export function getCurrentServerConfig(cfg: ArgoCLIConfig | null): ArgoServer | null {
  if (!cfg) return null;
  const serverUrl = getCurrentServer(cfg);
  if (!serverUrl) return null;
  return (cfg.servers ?? []).find(s => s.server === serverUrl) ?? null;
}

export function getCurrentServerUrl(cfg: ArgoCLIConfig | null): string | null {
  if (!cfg) return null;
  const serverUrl = getCurrentServer(cfg);
  if (!serverUrl) return null;
  const serverConfig = getCurrentServerConfig(cfg);
  return ensureHttps(serverUrl, serverConfig?.['plain-text']);
}
