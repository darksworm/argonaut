import type { ArgoApplication } from "../types/argo";
import type { Server } from "../types/server";
import { api } from "./transport";

export type RevisionMetadata = {
  author?: string;
  date?: string;
  message?: string;
  tags?: string[];
};

export async function getApplication(
  server: Server,
  name: string,
  appNamespace?: string,
  signal?: AbortSignal,
): Promise<ArgoApplication> {
  const params = new URLSearchParams();
  if (appNamespace) params.set("appNamespace", appNamespace);
  const data: any = await api(
    server,
    `/api/v1/applications/${encodeURIComponent(name)}${params.toString() ? `?${params.toString()}` : ""}`,
    { signal } as RequestInit,
  ).catch(() => null as any);
  return (data || {}) as ArgoApplication;
}

export async function getRevisionMetadata(
  server: Server,
  name: string,
  revision: string,
  appNamespace?: string,
  signal?: AbortSignal,
): Promise<RevisionMetadata | null> {
  const params = new URLSearchParams();
  if (appNamespace) params.set("appNamespace", appNamespace);
  const path = `/api/v1/applications/${encodeURIComponent(name)}/revisions/${encodeURIComponent(revision)}/metadata${params.toString() ? `?${params.toString()}` : ""}`;
  const data: any = await api(server, path, { signal } as RequestInit).catch(
    () => null as any,
  );
  if (!data) return null;
  return {
    author: data?.author || data?.Author,
    date: data?.date || data?.Date,
    message: data?.message || data?.Message,
    tags: Array.isArray(data?.tags) ? data.tags : undefined,
  };
}

export async function getManifests(
  server: Server,
  name: string,
  revision?: string,
  signal?: AbortSignal,
  appNamespace?: string,
): Promise<string[]> {
  const params = new URLSearchParams();
  if (revision) params.set("revision", revision);
  if (appNamespace) params.set("appNamespace", appNamespace);
  const path = `/api/v1/applications/${encodeURIComponent(name)}/manifests${params.toString() ? `?${params.toString()}` : ""}`;
  const data: any = await api(server, path, { signal } as RequestInit).catch(
    () => null as any,
  );
  // Argo can return {manifests: string[]} or raw array. Normalize.
  const arr: any[] = Array.isArray(data?.manifests)
    ? data.manifests
    : Array.isArray(data)
      ? data
      : [];
  // Ensure strings
  return arr.map((x) => (typeof x === "string" ? x : JSON.stringify(x)));
}

export async function postRollback(
  server: Server,
  name: string,
  body: {
    id: number;
    name: string;
    dryRun?: boolean;
    prune?: boolean;
    appNamespace?: string;
  },
): Promise<any> {
  const params = new URLSearchParams();
  if (body.appNamespace) params.set("appNamespace", body.appNamespace);
  const path = `/api/v1/applications/${encodeURIComponent(name)}/rollback${params.toString() ? `?${params.toString()}` : ""}`;
  return api(server, path, { method: "POST", body: JSON.stringify(body) });
}
