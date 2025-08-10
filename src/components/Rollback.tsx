import React, {useEffect, useMemo, useRef, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import {execa} from 'execa';
import {spawn as ptySpawn} from 'node-pty';
import {getApplication as getAppApi, getManifests as getManifestsApi, postRollback as postRollbackApi, getRevisionMetadata as getRevisionMetadataApi} from '../api/rollback';
import {watchApps} from '../api/applications.query';

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
  // helper fns from parent to avoid duplicating logic
  humanizeSince: (iso?: string) => string;
  singleLine: (s?: string) => string;
  shortSha: (s?: string) => string;
}

export default function Rollback(props: RollbackProps) {
  const {app, server, token, onClose, humanizeSince, singleLine, shortSha} = props;

  type SubMode = 'list' | 'confirm' | 'progress';
  const [subMode, setSubMode] = useState<SubMode>('list');
  const [fromRev, setFromRev] = useState<string | undefined>(undefined);
  const [rows, setRows] = useState<RollbackRow[]>([]);
  const [idx, setIdx] = useState(0);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState('');
  const [prune, setPrune] = useState(false);
  const [progressLog, setProgressLog] = useState<string[]>([]);
  const [metaLoadingKey, setMetaLoadingKey] = useState<string | null>(null);
  const metaAbortRef = useRef<AbortController | null>(null);
  const rollbackWatchAbortRef = useRef<AbortController | null>(null);

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
      try { rollbackWatchAbortRef.current?.abort(); } catch {}
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
      if (input.toLowerCase() === 'c' || key.return) { setSubMode('confirm'); return; }
      return;
    }
    if (subMode === 'confirm') {
      if (key.escape || input === 'q') { setSubMode('list'); return; }
      if (input.toLowerCase() === 'p') { setPrune(v => !v); return; }
      if (input.toLowerCase() === 'c' || key.return) { executeRollback(true); return; }
      return;
    }
    if (subMode === 'progress') {
      if (key.escape) {
        try { rollbackWatchAbortRef.current?.abort(); } catch {}
        onClose();
        return;
      }
      return;
    }
  });

  async function runRollbackDiff() {
    if (!server || !token) { setError('Not authenticated.'); return; }
    const row = rows[idx];
    if (!row) { setError('No selection to diff.'); return; }
    try {
      const current = await getManifestsApi(server, token, app).catch(() => []);
      const target = await getManifestsApi(server, token, app, row.revision).catch(() => []);
      const currentDocs = current.map(toYamlDoc).filter(Boolean) as string[];
      const targetDocs = target.map(toYamlDoc).filter(Boolean) as string[];
      const currentFile = await writeTmp(currentDocs, `${app}-current`);
      const targetFile = await writeTmp(targetDocs, `${app}-target-${row.id}`);

      // Try quiet diff first, bail if no diffs
      try { await execa('git', ['--no-pager','diff','--no-index','--quiet','--', currentFile, targetFile]); setError('No differences.'); return; } catch {}

      const shell = 'bash';
      const cols = (process.stdout as any)?.columns || 80;
      const pager = process.platform === 'darwin' ? "less -r -+X -K" : "less -R -+X -K";
      const cmd = `
:set -e
if command -v delta >/dev/null 2>&1; then
  DELTA_PAGER='${pager}' delta --paging=always --line-numbers --side-by-side --width=${cols} "${currentFile}" "${targetFile}" || true
else
  PAGER='${pager}'
  if ! command -v less >/dev/null 2>&1; then
    PAGER='sh -c "cat; printf \"\\n[Press Enter to close] \"; read -r _"'
  fi
  git --no-pager diff --no-index --color=always -- "${currentFile}" "${targetFile}" | eval "$PAGER" || true
fi
`;
      const args = process.platform === 'win32' ? ['-NoProfile','-NonInteractive','-Command', cmd] : ['-lc', cmd];
      const pty = ptySpawn(shell, args as any, { name:'xterm-256color', cols:(process.stdout as any)?.columns||80, rows:(process.stdout as any)?.rows||24, cwd:process.cwd(), env:{...(process.env as any), COLORTERM:'truecolor'} as any });
      const onResize = () => { try { pty.resize((process.stdout as any)?.columns||80, (process.stdout as any)?.rows||24); } catch {} };
      const onPtyData = (data: string) => { try { process.stdout.write(data); } catch {} };
      pty.onData(onPtyData);
      process.stdout.on('resize', onResize);
      const stdinAny = process.stdin as any;
      try { stdinAny.resume?.(); stdinAny.setRawMode?.(false);} catch {}
      await new Promise<void>(resolve => { pty.onExit(() => resolve()); });
      try { process.stdout.off('resize', onResize); } catch {}
      try { stdinAny.setRawMode?.(true); stdinAny.resume?.(); } catch {}
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
      // Start streaming progress
      setSubMode('progress');
      setProgressLog([]);
      try { rollbackWatchAbortRef.current?.abort(); } catch {}
      const ac = new AbortController();
      rollbackWatchAbortRef.current = ac;
      (async () => {
        try {
          for await (const evt of watchApps(server!, token!, undefined, ac.signal)) {
            const appl: any = evt?.application; const name = appl?.metadata?.name;
            if (name !== app) continue;
            const phase = appl?.status?.operationState?.phase || '';
            const msg = appl?.status?.operationState?.message || '';
            const h = appl?.status?.health?.status || '';
            const s = appl?.status?.sync?.status || '';
            setProgressLog(log => [...log, `[${new Date().toISOString()}] ${phase||s} ${h} ${msg}`].slice(-200));
            if ((h === 'Healthy' && s === 'Synced') || phase === 'Failed' || phase === 'Error') {
              try { ac.abort(); } catch {}
              break;
            }
          }
        } catch (e: any) {
          if (!ac.signal.aborted) setProgressLog(log => [...log, `Stream error: ${e?.message||String(e)}`]);
        } finally {
          onClose();
        }
      })();
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

  // progress
  return (
    <Box paddingX={2} flexDirection="column">
      <Text bold>
        Rollback in progress: <Text color="magentaBright">{app}</Text>
      </Text>
      <Box marginTop={1} flexDirection="column">
        {progressLog.slice(-Math.max(5, Math.min(20, progressLog.length))).map((l, i) => (
          <Text key={i} dimColor>
            {l}
          </Text>
        ))}
      </Box>
      <Box marginTop={1}>
        <Text dimColor>Esc to close</Text>
      </Box>
    </Box>
  );
}

function filterRollbackRow(row: RollbackRow, f: string): boolean {
  const q = (f || '').toLowerCase();
  if (!q) return true;
  const fields = [String(row.id || ''), String(row.revision || ''), String(row.author || ''), String(row.date || ''), String(row.message || '')];
  return fields.some((s) => s.toLowerCase().includes(q));
}

function toYamlDoc(input?: string): string | null {
  if (!input) return null;
  try {
    const obj = JSON.parse(input);
    return YAML.stringify(obj, {lineWidth: 120} as any);
  } catch {
    // assume already YAML
    return input;
  }
}

async function writeTmp(docs: string[], label: string): Promise<string> {
  const file = path.join(os.tmpdir(), `${label}-${Date.now()}.yaml`);
  const content = docs.filter(Boolean).join("\n---\n");
  await fs.writeFile(file, content, 'utf8');
  return file;
}
