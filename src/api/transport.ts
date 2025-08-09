import {ensureHttps} from '../config/paths';

export async function api(base: string, token: string, path: string, init?: RequestInit) {
  const url = ensureHttps(base) + path;
  // Normalize headers to a plain record for RequestInit compatibility
  const baseHeaders: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json'
  };
  if (init?.headers) {
    const h = init.headers as any;
    if (typeof (h as any)?.forEach === 'function') {
      // Headers instance
      (h as Headers).forEach((v: string, k: string) => { baseHeaders[k] = v; });
    } else {
      Object.assign(baseHeaders, h as Record<string, string>);
    }
  }
  const res = await fetch(url, {
    method: init?.method ?? 'GET',
    headers: baseHeaders,
    body: init?.body
  });
  if (!res.ok) throw new Error(`${init?.method ?? 'GET'} ${path} → ${res.status} ${res.statusText}`);
  return res.headers.get('content-type')?.includes('json') ? res.json() : res.text();
}
