import {useEffect, useState} from 'react';
import {listApps, watchApps} from '../api/applications.query';
import {appToItem} from '../services/app-mapper';
import type {AppItem} from '../types/domain';

export function useApps(
  baseUrl: string | null,
  token: string | null,
  paused: boolean = false,
  onAuthError?: (err: Error) => void
) {
  const [apps, setApps] = useState<AppItem[]>([]);
  const [status, setStatus] = useState('Idle');

  useEffect(() => {
    if (!baseUrl || !token) return;
    if (paused) {
      // When paused, ensure status reflects paused state and avoid setting up watchers
      setStatus('Paused');
      return;
    }
    const controller = new AbortController();
    let cancelled = false;
    (async () => {
      setStatus('Loading…');
      try {
        const items = (await listApps(baseUrl, token, controller.signal)).map(appToItem);
        if (!cancelled) setApps(items);
        setStatus('Live');
        for await (const ev of watchApps(baseUrl, token, undefined, controller.signal)) {
          const {type, application} = ev || ({} as any);
          if (!application?.metadata?.name) continue;
          setApps(curr => {
            const map = new Map(curr.map(a => [a.name, a] as const));
            if (application?.metadata?.name) {
              if (type === 'DELETED') {
                map.delete(application.metadata.name);
              } else {
                map.set(application.metadata.name, appToItem(application as any));
              }
            }
            return Array.from(map.values());
          });
        }
      } catch (e: any) {
        if (controller.signal.aborted) {
          // Silent on abort
          return;
        }
        const msg = e?.message || String(e);
        // If unauthorized, signal up to app to handle re-auth
        if (/\b(401|403)\b/i.test(msg) || /unauthorized/i.test(msg)) {
          onAuthError?.(e instanceof Error ? e : new Error(msg));
          setStatus('Auth required');
          return;
        }
        setStatus(`Error: ${msg}`);
      }
    })();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [baseUrl, token, paused]);

  return {apps, status};
}
