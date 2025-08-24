import type { Result } from "neverthrow";
import { type ApiError, wrapApiCall } from "../services/api-errors";
import { getHttpClient } from "../services/http-client";
import type { ApplicationWatchEvent, ArgoApplication } from "../types/argo";
import type { Server } from "../types/server";
import { api } from "./transport";

export type ResourceDiff = {
  liveState?: string;
  targetState?: string;
  name?: string;
  namespace?: string;
};

export function listApps(
  server: Server,
  signal?: AbortSignal,
): Promise<Result<ArgoApplication[], ApiError>> {
  return wrapApiCall(async () => {
    const data: any = await api(server, "/api/v1/applications", {
      signal,
    } as RequestInit);
    const items: any[] = Array.isArray(data?.items)
      ? data.items
      : Array.isArray(data)
        ? data
        : [];
    return items as ArgoApplication[];
  });
}

export function getManagedResourceDiffs(
  server: Server,
  appName: string,
  signal?: AbortSignal,
): Promise<Result<ResourceDiff[], ApiError>> {
  return wrapApiCall(async () => {
    const path = `/api/v1/applications/${encodeURIComponent(appName)}/managed-resources`;
    const data: any = await api(server, path, { signal } as RequestInit);
    const items: any[] = Array.isArray(data?.items)
      ? data.items
      : Array.isArray(data)
        ? data
        : [];
    return items as ResourceDiff[];
  });
}

// Async generator: yields {type, application}
export async function* watchApps(
  server: Server,
  params?: Record<string, string | string[]>,
  signal?: AbortSignal,
): AsyncGenerator<ApplicationWatchEvent, void, unknown> {
  const qs = new URLSearchParams();
  if (params)
    Object.entries(params).forEach(([k, v]) => {
      if (Array.isArray(v)) {
        v.forEach((x) => {
          qs.append(k, x);
        });
      } else {
        qs.set(k, v);
      }
    });
  const path = `/api/v1/stream/applications${qs.size ? `?${qs.toString()}` : ""}`;

  const client = getHttpClient(server.config, server.token);
  const stream = await client.stream(path, { signal });

  const decoder = new TextDecoder();
  let buffer = "";

  try {
    for await (const chunk of stream) {
      if (signal?.aborted) return;

      // Convert chunk to Uint8Array if it's a Buffer
      const uint8Array =
        chunk instanceof Buffer ? new Uint8Array(chunk) : (chunk as Uint8Array);
      buffer += decoder.decode(uint8Array, { stream: true });

      let i = buffer.indexOf("\n");
      while (i >= 0) {
        const line = buffer.slice(0, i).trim();
        buffer = buffer.slice(i + 1);

        if (line) {
          try {
            const msg = JSON.parse(line);
            if (msg?.result) {
              yield msg.result as ApplicationWatchEvent;
            }
          } catch {
            // ignore malformed lines
          }
        }
        i = buffer.indexOf("\n");
      }
    }
  } catch (e: any) {
    // If aborted, exit silently; otherwise rethrow
    if (e?.message === "Request aborted" || signal?.aborted) return;
    throw e;
  }
}
