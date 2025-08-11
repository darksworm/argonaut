import {readCLIConfig} from '../config/cli-config';

export async function tokenFromConfig(): Promise<string | null> {
  const cfg = await readCLIConfig();
  if (!cfg) return null;
  const current = cfg['current-context'];
  const ctx = (cfg.contexts ?? []).find(c => c.name === (current ?? ''));
  const userName = ctx?.user;
  const user = (cfg.users ?? []).find(u => u.name === (userName ?? ''));
  return user?.['auth-token'] ?? null;
}
