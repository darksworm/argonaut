import { useEffect, useState } from "react";
import { listApps, watchApps } from "../api/applications.query";
import { getDisplayMessage, requiresUserAction } from "../services/api-errors";
import { appToItem } from "../services/app-mapper";
import type { AppItem } from "../types/domain";
import type { Server } from "../types/server";

export function useApps(
  server: Server | null,
  paused: boolean = false,
  onAuthError?: (err: Error) => void,
) {
  const [apps, setApps] = useState<AppItem[]>([]);
  const [status, setStatus] = useState("Idle");

  // biome-ignore lint/correctness/useExhaustiveDependencies: onAuthError is a callback that would cause infinite re-renders
  useEffect(() => {
    if (!server) return;
    if (paused) {
      // When paused, ensure status reflects paused state and avoid setting up watchers
      setStatus("Paused");
      return;
    }
    const controller = new AbortController();
    let cancelled = false;
    (async () => {
      setStatus("Loading…");
      try {
        // Handle the Result-based API call
        const appsResult = await listApps(server, controller.signal);

        if (appsResult.isErr()) {
          if (controller.signal.aborted) {
            return; // Silent on abort
          }

          const apiError = appsResult.error;

          // Check if requires user action (re-authentication)
          if (requiresUserAction(apiError)) {
            onAuthError?.(new Error(getDisplayMessage(apiError)));
            setStatus("Auth required");
            return;
          }

          setStatus(`Error: ${getDisplayMessage(apiError)}`);
          return;
        }

        // Success - update apps and start watching
        const items = appsResult.value.map(appToItem);
        if (!cancelled) setApps(items);
        setStatus("Live");

        // Continue with watching (this still uses the old async generator approach)
        for await (const ev of watchApps(
          server,
          undefined,
          controller.signal,
        )) {
          const { type, application } = ev || ({} as any);
          if (!application?.metadata?.name) continue;
          setApps((curr) => {
            const map = new Map(curr.map((a) => [a.name, a] as const));
            if (application?.metadata?.name) {
              if (type === "DELETED") {
                map.delete(application.metadata.name);
              } else {
                map.set(
                  application.metadata.name,
                  appToItem(application as any),
                );
              }
            }
            return Array.from(map.values());
          });
        }
      } catch (e: any) {
        if (controller.signal.aborted) {
          return; // Silent on abort
        }

        // Fallback for any unexpected errors (from watchApps)
        const msg = e?.message || String(e);
        if (/\b(401|403)\b/i.test(msg) || /unauthorized/i.test(msg)) {
          onAuthError?.(e instanceof Error ? e : new Error(msg));
          setStatus("Auth required");
          return;
        }
        setStatus(`Error: ${msg}`);
      }
    })();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [server, paused]);

  return { apps, status };
}
