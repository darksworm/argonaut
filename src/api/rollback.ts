import {api} from './transport';
import type {ArgoApplication} from '../types/argo';

export type CanIResponse = { allowed?: boolean; value?: string };
export type SyncWindowsResponse = { canSync?: boolean; details?: string; message?: string };
export type RevisionMetadata = { author?: string; date?: string; message?: string; tags?: string[] };

export async function getApplication(server: string, token: string, name: string, signal?: AbortSignal): Promise<ArgoApplication> {
  const data: any = await api(server, token, `/api/v1/applications/${encodeURIComponent(name)}`, { signal } as RequestInit).catch(() => null as any);
  return (data || {}) as ArgoApplication;
}

export async function getRevisionMetadata(server: string, token: string, name: string, revision: string, signal?: AbortSignal): Promise<RevisionMetadata | null> {
  const path = `/api/v1/applications/${encodeURIComponent(name)}/revisions/${encodeURIComponent(revision)}/metadata`;
  const data: any = await api(server, token, path, { signal } as RequestInit).catch(() => null as any);
  if (!data) return null;
  return {
    author: data?.author || data?.Author,
    date: data?.date || data?.Date,
    message: data?.message || data?.Message,
    tags: Array.isArray(data?.tags) ? data.tags : undefined,
  };
}

export async function getManifests(server: string, token: string, name: string, revision?: string, signal?: AbortSignal): Promise<string[]> {
  const qs = revision ? `?revision=${encodeURIComponent(revision)}` : '';
  const path = `/api/v1/applications/${encodeURIComponent(name)}/manifests${qs}`;
  const data: any = await api(server, token, path, { signal } as RequestInit).catch(() => null as any);
  // Argo can return {manifests: string[]} or raw array. Normalize.
  const arr: any[] = Array.isArray(data?.manifests) ? data.manifests : (Array.isArray(data) ? data : []);
  // Ensure strings
  return arr.map(x => typeof x === 'string' ? x : JSON.stringify(x));
}

export async function getSyncWindows(server: string, token: string, name: string, signal?: AbortSignal): Promise<SyncWindowsResponse> {
  const path = `/api/v1/applications/${encodeURIComponent(name)}/syncwindows`;
  const data: any = await api(server, token, path, { signal } as RequestInit).catch(() => null as any);
  if (!data) return { canSync: true };
  // Argo returns {status: {canSync: boolean, ...}} or top-level canSync sometimes; be defensive
  const canSync = (data?.status?.canSync ?? data?.canSync);
  const details = data?.status?.message || data?.message || data?.details;
  return { canSync: canSync !== false, details };
}

export async function canIRollback(server: string, token: string, signal?: AbortSignal): Promise<boolean> {
  const path = `/api/v1/account/can-i/applications/update/rollback`;
  const data: any = await api(server, token, path, { signal } as RequestInit).catch(() => null as any);
  if (!data) return true; // fail-soft
  const allowed = data?.allowed ?? (typeof data?.value === 'string' ? data.value.toLowerCase() === 'yes' : undefined);
  return allowed !== false;
}

export async function postRollback(server: string, token: string, name: string, body: { id: number; name: string; dryRun?: boolean; prune?: boolean }): Promise<any> {
  const path = `/api/v1/applications/${encodeURIComponent(name)}/rollback`;
  return api(server, token, path, { method: 'POST', body: JSON.stringify(body) });
}
