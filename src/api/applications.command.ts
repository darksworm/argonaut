import {api} from './transport';

export async function syncApp(server: string, token: string, name: string): Promise<void> {
  await api(server, token, `/api/v1/applications/${encodeURIComponent(name)}/sync`, {
    method: 'POST',
    body: JSON.stringify({})
  });
}
