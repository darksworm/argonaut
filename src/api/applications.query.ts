import {api} from './transport';
import type {ApplicationWatchEvent, ArgoApplication} from '../types/argo';

export type ResourceDiff = {
  liveState?: string;
  targetState?: string;
  name?: string;
  namespace?: string;
};

export async function listApps(baseUrl: string, token: string, signal?: AbortSignal): Promise<ArgoApplication[]> {
  const data: any = await api(baseUrl, token, '/api/v1/applications', { signal } as RequestInit).catch(() => null as any);
  const items: any[] = Array.isArray(data?.items) ? data.items : Array.isArray(data) ? data : [];
  return items as ArgoApplication[];
}

export async function getManagedResourceDiffs(baseUrl: string, token: string, appName: string, signal?: AbortSignal): Promise<ResourceDiff[]> {
  const path = `/api/v1/applications/${encodeURIComponent(appName)}/managed-resources`;
  const data: any = await api(baseUrl, token, path, { signal } as RequestInit).catch(() => null as any);
  const items: any[] = Array.isArray(data?.items) ? data.items : Array.isArray(data) ? data : [];
  return items as ResourceDiff[];
}

// Async generator: yields {type, application}
export async function* watchApps(
  baseUrl: string,
  token: string,
  params?: Record<string, string | string[]>,
  signal?: AbortSignal
): AsyncGenerator<ApplicationWatchEvent, void, unknown> {
  const qs = new URLSearchParams();
  if (params) Object.entries(params).forEach(([k,v]) => Array.isArray(v) ? v.forEach(x=>qs.append(k,x)) : qs.set(k,v));
  const url = `${baseUrl}/api/v1/stream/applications${qs.size?`?${qs.toString()}`:''}`;
  const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` }, signal });
  if (!res.ok || !res.body) throw new Error(`watch failed: ${res.status} ${res.statusText}`);
  const reader = (res.body as any).getReader();
  const dec = new TextDecoder(); let buf = '';
  try {
    for (;;) {
      const {value, done} = await reader.read(); if (done) return;
      buf += dec.decode(value, {stream:true});
      for (let i; (i = buf.indexOf('\n')) >= 0; ) {
        const line = buf.slice(0, i).trim(); buf = buf.slice(i+1);
        if (!line) continue;
        try {
          const msg = JSON.parse(line);
          if (msg?.result) yield msg.result as ApplicationWatchEvent; // { type, application }
        } catch {
          // ignore malformed lines
        }
      }
    }
  } catch (e: any) {
    // If aborted, exit silently; otherwise rethrow
    if (e?.name === 'AbortError') return;
    // Some runtimes may wrap abort as DOMException or custom error; check signal
    if (signal?.aborted) return;
    throw e;
  }
}
