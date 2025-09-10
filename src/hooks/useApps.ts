import { useEffect, useState } from "react";
import {
  type ArgoApiEvent,
  ArgoApiService,
} from "../services/argo-api-service";
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

    const apiService = new ArgoApiService();

    // Event handler for ArgoApiService events
    const handleApiEvent = (event: ArgoApiEvent) => {
      switch (event.type) {
        case "apps-loaded":
          setApps(event.apps);
          break;
        case "app-updated":
          setApps((curr) => {
            const map = new Map(curr.map((a) => [a.name, a] as const));
            map.set(event.app.name, event.app);
            return Array.from(map.values());
          });
          break;
        case "app-deleted":
          setApps((curr) => {
            const map = new Map(curr.map((a) => [a.name, a] as const));
            map.delete(event.appName);
            return Array.from(map.values());
          });
          break;
        case "auth-error":
          onAuthError?.(event.error);
          break;
        case "api-error":
          // Status will be updated by status-change event
          break;
        case "status-change":
          setStatus(event.status);
          break;
      }
    };

    // Start watching applications
    const watchPromise = apiService.watchApplications(server, handleApiEvent);

    let cleanup: (() => void) | undefined;
    watchPromise.then((cleanupFn) => {
      cleanup = cleanupFn;
    });

    return () => {
      cleanup?.();
      apiService.cleanup();
    };
  }, [server, paused]);

  return { apps, status };
}
