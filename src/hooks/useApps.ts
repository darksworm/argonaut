import {useEffect, useState} from 'react';
import {listApps, watchApps} from '../api/applications.query';
import {appToItem} from '../services/app-mapper';
import type {AppItem} from '../types/domain';

export function useApps(server: string|null, token: string|null, paused: boolean = false) {
  const [apps, setApps] = useState<AppItem[]>([]);
  const [status, setStatus] = useState('Idle');

  useEffect(() => {
    if (!server || !token) return;
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
        const items = (await listApps(server, token, controller.signal)).map(appToItem);
        if (!cancelled) setApps(items);
        setStatus('Live');
        for await (const ev of watchApps(server, token, undefined, controller.signal)) {
          const {type, application} = ev || {} as any;
          // @ts-expect-error minimal runtime guard
          if (!application?.metadata?.name) continue;
          setApps(curr => {
            const map = new Map(curr.map(a => [a.name, a] as const));
            if (type === 'DELETED') map.delete(application.metadata.name);
            else map.set(application.metadata.name, appToItem(application as any));
            return Array.from(map.values());
          });
        }
      } catch (e: any) {
        if (controller.signal.aborted) {
          // Silent on abort
          return;
        }
        setStatus(`Error: ${e?.message || String(e)}`);
      }
    })();
    return () => { cancelled = true; controller.abort(); };
  }, [server, token, paused]);

  return {apps, status};
}
