import {api} from './transport';

export async function syncApp(baseUrl: string, token: string, name: string, opts?: { prune?: boolean; appNamespace?: string }): Promise<void> {
  const params = new URLSearchParams();
  if (opts?.appNamespace) params.set('appNamespace', opts.appNamespace);
  
  await api(baseUrl, token, `/api/v1/applications/${encodeURIComponent(name)}/sync${params.toString() ? `?${params.toString()}` : ''}`, {
    method: 'POST',
    body: JSON.stringify({ prune: !!opts?.prune, appNamespace: opts?.appNamespace })
  });
}
