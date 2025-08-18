import type {Server} from '../types/server';

export async function api(server: Server, path: string, init?: RequestInit) {
  const url = server.baseUrl + path;
  // Normalize headers to a plain record for RequestInit compatibility
  const baseHeaders: Record<string, string> = {
    Authorization: `Bearer ${server.token}`,
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

  // Handle insecure TLS
  const fetchInit: RequestInit = {
    method: init?.method ?? 'GET',
    headers: baseHeaders,
    body: init?.body,
    signal: init?.signal
  };

  // For insecure connections, we need to configure the agent to ignore cert errors
  if (server.insecure && typeof globalThis !== 'undefined' && 'process' in globalThis) {
    // In Node.js environment, we can set rejectUnauthorized to false
    // This is a Node.js-specific feature and won't work in browsers
    process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
  }

  const res = await fetch(url, fetchInit);
  
  // Reset the TLS setting after the request
  if (server.insecure && typeof globalThis !== 'undefined' && 'process' in globalThis) {
    delete process.env.NODE_TLS_REJECT_UNAUTHORIZED;
  }

  if (!res.ok) throw new Error(`${init?.method ?? 'GET'} ${path} → ${res.status} ${res.statusText}`);
  return res.headers.get('content-type')?.includes('json') ? res.json() : res.text();
}
