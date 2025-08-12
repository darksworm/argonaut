import React, {useEffect, useRef, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import {getApplication as getAppApi, postRollback as postRollbackApi, getRevisionMetadata as getRevisionMetadataApi} from '../api/rollback';
import {runRollbackDiffSession} from './DiffView';
import {humanizeSince, shortSha, singleLine} from "../utils";

export type RollbackRow = {
  id: number;
  revision: string;
  deployedAt?: string;
  author?: string;
  date?: string;
  message?: string;
  metaError?: string;
};

interface RollbackProps {
  app: string;
  server: string | null;
  token: string | null;
  onClose: () => void;
  onStartWatching: (appName: string) => void;
}

export default function Rollback(props: RollbackProps) {
  const {app, server, token, onClose, onStartWatching} = props;

  type SubMode = 'list' | 'confirm';
  const [subMode, setSubMode] = useState<SubMode>('list');
  const [fromRev, setFromRev] = useState<string | undefined>(undefined);
  const [rows, setRows] = useState<RollbackRow[]>([]);
  const [idx, setIdx] = useState(0);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState('');
  const [prune, setPrune] = useState(false);
  const [metaLoadingKey, setMetaLoadingKey] = useState<string | null>(null);
  const metaAbortRef = useRef<AbortController | null>(null);

  // Initial fetch of app history and current revision
  useEffect(() => {
    (async () => {
      try {
        if (!server || !token) {
          setError('Not authenticated.');
          setRows([]);
          return;
        }
        const appObj = await getAppApi(server, token, app).catch(() => ({} as any));
        const from = appObj?.status?.sync?.revision || '';
        setFromRev(from || undefined);
        const hist = Array.isArray(appObj?.status?.history) ? [...(appObj.status!.history!)] : [];
        const r: RollbackRow[] = hist
          .map((h: any) => ({id: Number(h?.id ?? 0), revision: String(h?.revision || ''), deployedAt: h?.deployedAt}))
          .filter(h => h.id > 0 && h.revision)
          .sort((a, b) => b.id - a.id);
        if (!r.length) {
          setError('No previous syncs found.');
          setRows([]);
        } else {
          setError('');
          setRows(r);
        }
        setIdx(0);
        setFilter('');
        setSubMode('list');
      } catch (e: any) {
        setError(e?.message || String(e));
        setRows([]);
        setIdx(0);
        setFilter('');
        setSubMode('list');
      }
    })();
    return () => {
      try { metaAbortRef.current?.abort(); } catch {}
    };
  }, [app, server, token]);

  // Fetch revision metadata for highlighted row
  useEffect(() => {
    if (subMode !== 'list') return;
    if (!server || !token) return;
    const row = rows[idx];
    if (!row || row.author) return;
    try { metaAbortRef.current?.abort(); } catch {}
    const ac = new AbortController();
    metaAbortRef.current = ac;
    const key = `${app}:${row.id}:${row.revision}`;
    setMetaLoadingKey(key);
    (async () => {
      try {
        const meta = await getRevisionMetadataApi(server, token, app, row.revision, ac.signal);
        const upd = [...rows];
        upd[idx] = {...row, author: meta?.author, date: meta?.date, message: meta?.message};
        setRows(upd);
      } catch (e: any) {
        const upd = [...rows];
        upd[idx] = {...row, metaError: e?.message || String(e)};
        setRows(upd);
      } finally {
        setMetaLoadingKey(prev => (prev === key ? null : prev));
      }
    })();
    return () => {
      try { ac.abort(); } catch {}
    };
  }, [subMode, idx, rows, app, server, token]);

  // Key handling inside rollback overlay
  useInput((input, key) => {
    if (subMode === 'list') {
      if (key.escape || input === 'q') { onClose(); return; }
      if (input === 'j' || key.downArrow) { setIdx(i => Math.min(i + 1, Math.max(0, rows.filter(r => filterRollbackRow(r, filter)).length - 1))); return; }
      if (input === 'k' || key.upArrow) { setIdx(i => Math.max(i - 1, 0)); return; }
      if (input.toLowerCase() === 'd') { runRollbackDiff(); return; }
      if (input.toLowerCase() === 'c' || key.return) { if (idx !== 0) setSubMode('confirm'); return; }
      return;
    }
    if (subMode === 'confirm') {
      if (key.escape || input === 'q') { setSubMode('list'); return; }
      if (input.toLowerCase() === 'p') { setPrune(v => !v); return; }
      if (input.toLowerCase() === 'c' || key.return) { executeRollback(true); return; }
      return;
    }
  });

  async function runRollbackDiff() {
    if (!server || !token) { setError('Not authenticated.'); return; }
    const row = rows[idx];
    if (!row) { setError('No selection to diff.'); return; }
    try {
      const opened = await runRollbackDiffSession(server, token, app, row.revision, { forwardInput: true });
      if (!opened) setError('No differences.');
    } catch (e: any) {
      setError(`Diff failed: ${e?.message || String(e)}`);
    }
  }

  async function executeRollback(confirm: boolean) {
    if (!confirm) { setSubMode('list'); setError('Rollback cancelled.'); return; }
    const row = rows[idx];
    if (!server || !token || !row) { setError('Not ready.'); return; }
    try {
      await postRollbackApi(server, token, app, {id: row.id, name: app, prune});
      // Close rollback view and start watching via resources view
      onStartWatching(app);
    } catch (e: any) {
      setError(e?.message || String(e));
      setSubMode('confirm');
    }
  }

  // Render
  if (subMode === 'list') {
    return (
      <Box paddingX={1} flexDirection="column">
        <Text bold>
          Rollback: <Text color="magentaBright">{app}</Text>
        </Text>
        <Box marginTop={1}>
          <Text>
            Current revision: <Text color="cyan">{shortSha(fromRev)}</Text>
          </Text>
        </Box>
        {error && (
          <Box marginTop={1}>
            <Text color="red">{error}</Text>
          </Box>
        )}
        <Box marginTop={1} flexDirection="column">
          <Box>
            <Box width={6}>
              <Text bold>ID</Text>
            </Box>
            <Box width={10}>
              <Text bold>Revision</Text>
            </Box>
            <Box width={20}>
              <Text bold>Deployed</Text>
            </Box>
            <Box flexGrow={1}>
              <Text bold>Message</Text>
            </Box>
          </Box>
          {(() => {
            const filtered = rows.filter((r) => filterRollbackRow(r, filter));
            const maxRows = Math.max(1, Math.min(10, filtered.length));
            const start = Math.max(0, Math.min(idx - Math.floor(maxRows / 2), Math.max(0, filtered.length - maxRows)));
            const slice = filtered.slice(start, start + maxRows);
            return slice.map((r: RollbackRow, i: number) => {
              const actual = start + i;
              const active = actual === idx;
              return (
                <Box key={`${r.id}-${r.revision}`} backgroundColor={active ? 'magentaBright' : undefined}>
                  <Box width={6}>
                    <Text>{String(r.id)}</Text>
                  </Box>
                  <Box width={10}>
                    <Text>{shortSha(r.revision)}</Text>
                  </Box>
                  <Box width={20}>
                    <Text>{r.deployedAt ? humanizeSince(r.deployedAt) + ' ago' : '—'}</Text>
                  </Box>
                  <Box flexGrow={1}>
                    <Text wrap="truncate-end">
                      {metaLoadingKey === `${app}:${r.id}:${r.revision}` ? '(loading…)' : singleLine(r.message || r.metaError || '')}
                    </Text>
                  </Box>
                </Box>
              );
            });
          })()}
        </Box>
        <Box marginTop={1}>
          <Text dimColor>j/k to move • d diff • c confirm • Esc/q cancel</Text>
        </Box>
      </Box>
    );
  }

  if (subMode === 'confirm') {
    const row: any = rows[idx];
    return (
      <Box paddingX={2} flexDirection="column">
        <Text bold>Confirm rollback</Text>
        <Box marginTop={1}>
          <Text>
            App: <Text color="magentaBright">{app}</Text>
          </Text>
        </Box>
        <Box>
          <Text>
            From: <Text color="cyan">{shortSha(fromRev)}</Text> → To: <Text color="cyan">{row ? shortSha(row.revision) : '—'}</Text>
          </Text>
        </Box>
        <Box>
          <Text>
            History ID: <Text color="cyan">{row?.id ?? '—'}</Text>
          </Text>
        </Box>
        {error && (
          <Box marginTop={1}>
            <Text color="red">{error}</Text>
          </Box>
        )}
        <Box marginTop={1}>
          <Text>
            Prune [p]: <Text color={(prune ? 'yellow' : undefined) as any}>{prune ? 'on' : 'off'}</Text>
          </Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Press c to confirm, Esc/q to go back</Text>
        </Box>
      </Box>
    );
  }

  // Should not reach here since we only have 'list' and 'confirm' modes
  return null;
}

function filterRollbackRow(row: RollbackRow, f: string): boolean {
  const q = (f || '').toLowerCase();
  if (!q) return true;
  const fields = [String(row.id || ''), String(row.revision || ''), String(row.author || ''), String(row.date || ''), String(row.message || '')];
  return fields.some((s) => s.toLowerCase().includes(q));
}
