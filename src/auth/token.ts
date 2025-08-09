import {execa} from 'execa';
import {readCLIConfig} from '../config/cli-config';

export async function ensureSSOLogin(server: string): Promise<void> {
  await execa('argocd', ['login', server, '--sso', '--grpc-web'], {stdio: 'inherit'} as any);
}

export async function tokenFromConfig(): Promise<string | null> {
  const cfg = await readCLIConfig();
  if (!cfg) return null;
  const current = cfg['current-context'];
  const ctx = (cfg.contexts ?? []).find(c => c.name === (current ?? ''));
  const userName = ctx?.user;
  const user = (cfg.users ?? []).find(u => u.name === (userName ?? ''));
  return user?.['auth-token'] ?? null;
}

export async function ensureToken(server: string): Promise<string> {
  const tok = await tokenFromConfig();
  if (tok) return tok;
  await ensureSSOLogin(server);
  const tok2 = await tokenFromConfig();
  if (tok2) return tok2;
  throw new Error('No token available after SSO.');
}
