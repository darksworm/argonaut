import React, {useEffect, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import {ensureHttps} from '../config/paths';
import '../api/transport'; // ensure TLS relax env is applied (self-signed certs)
import {colorFor} from '../utils';

// Types for streamed resources
export type Health = {status?: string; message?: string};
export type ResourceNode = {
  group?: string;
  kind: string;
  name: string;
  namespace?: string;
  version?: string;
  health?: Health;
};
export type ApplicationTree = {
  nodes?: ResourceNode[];
};

export type ApplicationWatchEvent = {
  application?: {
    status?: {
      resources?: Array<{
        group?: string;
        kind?: string;
        name?: string;
        namespace?: string;
        version?: string;
        status?: string; // Synced | OutOfSync | Unknown
      }>;
    };
  };
};

// Stream NDJSON from ArgoCD streaming API; yields objects at obj.result
export async function* streamJsonResults<T>(url: string, token: string, signal?: AbortSignal): AsyncGenerator<T> {
  const res = await fetch(url, {headers: {Authorization: `Bearer ${token}`}, signal} as any);
  if (!(res as any).body) throw new Error('No response body');
  const reader = (res as any).body.getReader();
  const decoder = new TextDecoder();
  let buf = '';

  while (true) {
    const {value, done} = await reader.read();
    if (done) break;
    buf += decoder.decode(value, {stream: true});

    let nl: number;
    while ((nl = buf.indexOf('\n')) >= 0) {
      const line = buf.slice(0, nl).trim();
      buf = buf.slice(nl + 1);
      if (!line) continue;
      try {
        const obj = JSON.parse(line);
        if (obj?.result) yield obj.result as T;
      } catch {
        // tolerate partial/non-NDJSON frames; keep buffering
      }
    }
  }

  if (buf.trim()) {
    try {
      const obj = JSON.parse(buf.trim());
      if (obj?.result) yield obj.result as T;
    } catch {/* ignore */}
  }
}

const keyFor = (n: {group?: string; kind?: string; namespace?: string; name?: string; version?: string}) =>
  `${n.group || ''}/${n.kind || ''}/${n.namespace || ''}/${n.name || ''}/${n.version || ''}`;

function ResourceRow({r, syncByKey}: {r: ResourceNode; syncByKey: Record<string, string>}) {
  const status = r.health?.status ?? '-';
  const statusColor = colorFor(status);
  const syncVal = syncByKey[keyFor(r)] ?? '-';
  const syncColor = colorFor(syncVal);
  return (
    <Box width="100%">
      <Box width={13} flexShrink={0}>
        <Text wrap="truncate">{r.kind}</Text>
      </Box>
      <Box width={1} flexShrink={0}/>
      <Box flexGrow={1} flexShrink={1} minWidth={0}>
        <Text wrap="truncate-end">{r.name}</Text>
      </Box>
      <Box width={1} flexShrink={0}/>
      <Box width={12} flexShrink={0} justifyContent="flex-end">
        <Text color={syncColor.color as any} dimColor={syncColor.dimColor as any} wrap="truncate">{syncVal}</Text>
      </Box>
      <Box width={1} flexShrink={0}/>
      <Box width={12} flexShrink={0} justifyContent="flex-end">
        <Text color={statusColor.color as any} dimColor={statusColor.dimColor as any} wrap="truncate">{status}</Text>
      </Box>
    </Box>
  );
}

function Table({rows, syncByKey}: {rows: ResourceNode[]; syncByKey: Record<string, string>}) {
  return (
    <Box flexDirection="column">
      <Box>
        <Box width={13} flexShrink={0}><Text bold color="yellowBright">KIND</Text></Box>
        <Box width={1} flexShrink={0}/>
        <Box flexGrow={1} flexShrink={1} minWidth={0}><Text bold color="yellowBright">NAME</Text></Box>
        <Box width={1} flexShrink={0}/>
        <Box width={12} flexShrink={0} justifyContent="flex-end"><Text bold color="yellowBright">SYNC</Text></Box>
        <Box width={1} flexShrink={0}/>
        <Box width={12} flexShrink={0} justifyContent="flex-end"><Text bold color="yellowBright">STATUS</Text></Box>
      </Box>
      {rows.map((r, i) => <ResourceRow key={`${r.kind}//${r.name}/${i}`} r={r} syncByKey={syncByKey} />)}
    </Box>
  );
}

export type ResourceStreamProps = {
  baseUrl: string;      // e.g. https://argocd.example.com
  token: string;        // Argo CD JWT
  appName: string;      // Application name
  appNamespace?: string; // Application control plane namespace
  onExit?: () => void;  // called when user quits the view (press 'q')
};

export const ResourceStream: React.FC<ResourceStreamProps> = ({baseUrl, token, appName, appNamespace, onExit}) => {
  const [rows, setRows] = useState<ResourceNode[]>([]);
  const [hint, setHint] = useState('Press q or Esc to return');
  const [syncByKey, setSyncByKey] = useState<Record<string, string>>({});

  // Initial fetch: resource tree (non-streaming) so view has immediate data
  useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams();
    if (appNamespace) params.set('appNamespace', appNamespace);
    const url = `${ensureHttps(baseUrl)}/api/v1/applications/${encodeURIComponent(appName)}/resource-tree${params.toString() ? `?${params.toString()}` : ''}`;
    (async () => {
      try {
        const res = await fetch(url, {headers: {Authorization: `Bearer ${token}`}, signal: controller.signal} as any);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const tree: ApplicationTree = await res.json();
        const next = (tree.nodes ?? []).map(n => ({
          group: (n as any).group,
          kind: n.kind,
          name: n.name,
          namespace: n.namespace,
          version: (n as any).version,
          health: n.health
        }));
        setRows(next);
      } catch (err: any) {
        const msg = String(err?.message ?? err);
        if (!/aborted|AbortError/i.test(msg)) {
          setHint(h => h.includes('Stream error') ? h : `Load error: ${msg}`);
        }
      }
    })();
    return () => controller.abort();
  }, [baseUrl, token, appName, appNamespace]);

  // Stream: resource tree updates
  useEffect(() => {
    let cancel = false;
    const controller = new AbortController();
    const params = new URLSearchParams();
    if (appNamespace) params.set('appNamespace', appNamespace);
    const url = `${ensureHttps(baseUrl)}/api/v1/stream/applications/${encodeURIComponent(appName)}/resource-tree${params.toString() ? `?${params.toString()}` : ''}`;
    (async () => {
      try {
        for await (const tree of streamJsonResults<ApplicationTree>(url, token, controller.signal)) {
          if (cancel) break;
          const next = (tree.nodes ?? []).map(n => ({
            group: (n as any).group,
            kind: n.kind,
            name: n.name,
            namespace: n.namespace,
            version: (n as any).version,
            health: n.health
          }));
          setRows(next);
        }
      } catch (err: any) {
        const msg = String(err?.message ?? err);
        if (!/aborted|AbortError/i.test(msg)) {
          setHint(`Stream error: ${msg}`);
        }
      }
    })();
    return () => { cancel = true; controller.abort(); };
  }, [baseUrl, token, appName, appNamespace]);

  // Initial fetch: application status -> syncByKey
  useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams();
    if (appNamespace) params.set('appNamespace', appNamespace);
    const url = `${ensureHttps(baseUrl)}/api/v1/applications/${encodeURIComponent(appName)}${params.toString() ? `?${params.toString()}` : ''}`;
    (async () => {
      try {
        const res = await fetch(url, {headers: {Authorization: `Bearer ${token}`}, signal: controller.signal} as any);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const app = await res.json();
        const resources = app?.status?.resources || [];
        if (Array.isArray(resources)) {
          const m: Record<string, string> = {};
          for (const r of resources) {
            const k = keyFor(r as any);
            if (k) m[k] = (r as any).status || '-';
          }
          setSyncByKey(m);
        }
      } catch (err: any) {
        const msg = String(err?.message ?? err);
        if (!/aborted|AbortError/i.test(msg)) {
          setHint(h => h.includes('Stream error') ? h : `${h}`);
        }
      }
    })();
    return () => controller.abort();
  }, [baseUrl, token, appName, appNamespace]);

  // Stream application watch events to derive per-resource sync status
  useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams();
    params.set('name', appName);
    if (appNamespace) params.set('appNamespace', appNamespace);
    const url = `${ensureHttps(baseUrl)}/api/v1/stream/applications?${params.toString()}`;
    (async () => {
      try {
        for await (const evt of streamJsonResults<ApplicationWatchEvent>(url, token, controller.signal)) {
          const resources = evt?.application?.status?.resources || [];
          if (!resources || !Array.isArray(resources)) continue;
          const m: Record<string, string> = {};
          for (const r of resources) {
            const k = keyFor(r as any);
            if (k) m[k] = r.status || '-';
          }
          setSyncByKey(m);
        }
      } catch (err: any) {
        const msg = String(err?.message ?? err);
        if (!/aborted|AbortError/i.test(msg)) {
          // Don't override main hint if already set; append minimal info
          setHint(h => h.includes('Stream error') ? h : `${h}`);
        }
      }
    })();
    return () => controller.abort();
  }, [baseUrl, token, appName, appNamespace]);

  useInput((input, key) => {
    const ch = (input || '').toLowerCase();
    if (ch === 'q' || key?.escape) {
      onExit?.();
    }
  });

  return (
    <Box flexDirection="column">
      <Text bold>Resources for: {appName}</Text>
      <Box paddingTop={1}/>
      <Table rows={rows} syncByKey={syncByKey} />
      <Box marginTop={1}><Text dimColor>{hint}</Text></Box>
    </Box>
  );
};

