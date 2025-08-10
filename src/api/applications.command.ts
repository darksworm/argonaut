import {api} from './transport';

export async function syncApp(server: string, token: string, name: string, opts?: { prune?: boolean }): Promise<void> {
  await api(server, token, `/api/v1/applications/${encodeURIComponent(name)}/sync`, {
    method: 'POST',
    body: JSON.stringify({ prune: !!opts?.prune })
  });
}
