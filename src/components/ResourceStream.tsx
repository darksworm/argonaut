// @ts-nocheck
import React, {useEffect, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import {spawn} from 'node:child_process';
import {ensureHttps} from '../config/paths';
import '../api/transport'; // ensure TLS relax env is applied (self-signed certs)
import {colorFor} from '../utils';

// Types for streamed resources
export type Health = {status?: string; message?: string};
export type ResourceNode = {
  kind: string;
  name: string;
  namespace?: string;
  health?: Health;
  syncStatus?: string;
};
export type ApplicationTree = {
  nodes?: ResourceNode[];
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

function pad(s: string, w: number) {
  return (s ?? '').slice(0, w).padEnd(w, ' ');
}

function ResourceRow({r}: {r: ResourceNode}) {
  const sync = r.syncStatus ?? '-';
  const health = r.health?.status ?? '-';
  const syncColor = colorFor(sync);
  const healthColor = colorFor(health);
  return (
    <Box width="100%">
      <Box width={13}>
        <Text wrap="truncate">{r.kind}</Text>
      </Box>
      <Box width={1}/>
      <Box flexGrow={1} flexShrink={1} minWidth={0}>
        <Text wrap="truncate-end">{r.name}</Text>
      </Box>
      <Box width={1}/>
      <Box width={12} justifyContent="flex-end">
        <Text color={syncColor.color as any} dimColor={syncColor.dimColor as any} wrap="truncate">{sync}</Text>
      </Box>
      <Box width={1}/>
      <Box width={10} justifyContent="flex-end">
        <Text color={healthColor.color as any} dimColor={healthColor.dimColor as any} wrap="truncate">{health}</Text>
      </Box>
    </Box>
  );
}

function Table({rows}: {rows: ResourceNode[]}) {
  return (
    <Box flexDirection="column">
      <Box>
        <Box width={13}><Text bold color="yellowBright">KIND</Text></Box>
        <Box width={1}/>
        <Box flexGrow={1} flexShrink={1} minWidth={0}><Text bold color="yellowBright">NAME</Text></Box>
        <Box width={1}/>
        <Box width={12} justifyContent="flex-end"><Text bold color="yellowBright">SYNC</Text></Box>
        <Box width={1}/>
        <Box width={10} justifyContent="flex-end"><Text bold color="yellowBright">HEALTH</Text></Box>
      </Box>
      {rows.map((r, i) => <ResourceRow key={`${r.kind}//${r.name}/${i}`} r={r} />)}
    </Box>
  );
}

export type ResourceStreamProps = {
  baseUrl: string;      // e.g. https://argocd.example.com
  token: string;        // Argo CD JWT
  appName: string;      // Application name
  context?: string;     // optional kube context for k9s
  namespace?: string;   // optional namespace for k9s
  onExit?: () => void;  // called when user quits the view (press 'q')
};

export const ResourceStream: React.FC<ResourceStreamProps> = ({baseUrl, token, appName, context, namespace, onExit}) => {
  const [rows, setRows] = useState<ResourceNode[]>([]);
  const [hint, setHint] = useState('Press k to open k9s • Press q to return');

  useEffect(() => {
    let cancel = false;
    const controller = new AbortController();
    const url = `${ensureHttps(baseUrl)}/api/v1/stream/applications/${encodeURIComponent(appName)}/resource-tree`;
    (async () => {
      try {
        for await (const tree of streamJsonResults<ApplicationTree>(url, token, controller.signal)) {
          if (cancel) break;
          const next = (tree.nodes ?? []).map(n => ({
            kind: n.kind,
            name: n.name,
            namespace: n.namespace,
            syncStatus: n.syncStatus,
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
  }, [baseUrl, token, appName]);

  useInput((input) => {
    const ch = input.toLowerCase();
    if (ch === 'q') {
      onExit?.();
    }
  });

  return (
    <Box flexDirection="column">
      <Text bold>Resources for: {appName}</Text>
      <Box paddingTop={1}/>
      <Table rows={rows} />
      <Box marginTop={1}><Text dimColor>{hint}</Text></Box>
    </Box>
  );
};

