import os from 'node:os';
import path from 'node:path';

export const CONFIG_PATH =
  process.env.ARGOCD_CONFIG ??
  path.join(process.env.XDG_CONFIG_HOME || path.join(os.homedir(), '.config'), 'argocd', 'config');

export function ensureHttps(base: string): string {
  if (base.startsWith('http://') || base.startsWith('https://')) return base;
  return `https://${base}`;
}

export function hostFromServer(server?: string): string {
  if (!server) return '';
  try {
    const u = new URL(server.startsWith('http') ? server : `https://${server}`);
    return u.host;
  } catch {
    return server;
  }
}
