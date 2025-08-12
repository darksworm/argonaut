import React, {useEffect, useRef, useState} from 'react';
import {Box, Text, useInput} from 'ink';
import {
    getApplication as getAppApi,
    postRollback as postRollbackApi,
    getRevisionMetadata as getRevisionMetadataApi
} from '../api/rollback';
import {runRollbackDiffSession} from './DiffView';
import ConfirmationBox from './ConfirmationBox';
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
    const [watch, setWatch] = useState(true);
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
                    .map((h: any) => ({
                        id: Number(h?.id ?? 0),
                        revision: String(h?.revision || ''),
                        deployedAt: h?.deployedAt
                    }))
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
            try {
                metaAbortRef.current?.abort();
            } catch {
            }
        };
    }, [app, server, token]);

    // Fetch revision metadata for highlighted row
    useEffect(() => {
        if (subMode !== 'list') return;
        if (!server || !token) return;
        const row = rows[idx];
        if (!row || row.author) return;
        try {
            metaAbortRef.current?.abort();
        } catch {
        }
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
            try {
                ac.abort();
            } catch {
            }
        };
    }, [subMode, idx, rows, app, server, token]);

    // Key handling inside rollback overlay
    useInput((input, key) => {
        if (subMode === 'list') {
            if (key.escape || input === 'q') {
                onClose();
                return;
            }
            if (input === 'j' || key.downArrow) {
                setIdx(i => Math.min(i + 1, Math.max(0, rows.filter(r => filterRollbackRow(r, filter)).length - 1)));
                return;
            }
            if (input === 'k' || key.upArrow) {
                setIdx(i => Math.max(i - 1, 0));
                return;
            }
            if (input.toLowerCase() === 'd') {
                runRollbackDiff();
                return;
            }
            if (key.return) {
                if (idx !== 0) {
                    setSubMode('confirm');
                }
                return;
            }
            if (input.toLowerCase() === 'p') {
                setPrune(v => !v);
                return;
            }
            return;
        }
        if (subMode === 'confirm') {
            if (key.escape || input === 'q') {
                setSubMode('list');
                return;
            }
            if (input.toLowerCase() === 'p') {
                setPrune(v => !v);
                return;
            }
            if (input.toLowerCase() === 'w') {
                setWatch(v => !v);
                return;
            }
            // All other handling is done via the ConfirmationBox component
            return;
        }
    });

    async function runRollbackDiff() {
        if (!server || !token) {
            setError('Not authenticated.');
            return;
        }
        const row = rows[idx];
        if (!row) {
            setError('No selection to diff.');
            return;
        }
        try {
            const opened = await runRollbackDiffSession(server, token, app, row.revision, {forwardInput: true});
            if (!opened) setError('No differences.');
        } catch (e: any) {
            setError(`Diff failed: ${e?.message || String(e)}`);
        }
    }

    async function executeRollback(confirm: boolean) {
        if (!confirm) {
            setSubMode('list');
            setError('Rollback cancelled.');
            return;
        }
        const row = rows[idx];
        if (!server || !token || !row) {
            setError('Not ready.');
            return;
        }
        try {
            await postRollbackApi(server, token, app, {id: row.id, name: app, prune});
            // Start watching via resources view and close rollback
            if (watch) onStartWatching(app); else onClose();
        } catch (e: any) {
            setError(e?.message || String(e));
            setSubMode('confirm');
        }
    }

    // Render
    const row = rows[idx];

    return (<>
        {/* Rollback confirmation box (rendered outside the bordered container) */}
        {subMode === 'confirm' && row && (
            <Box marginTop={1}>
                <ConfirmationBox
                    title="Confirm rollback"
                    message={`Rollback ${app} from ${shortSha(fromRev)} to`}
                    target={shortSha(row.revision)}
                    options={[
                        {key: 'p', label: 'Prune', value: prune},
                        {
                            key: 'w',
                            label: 'Watch',
                            value: watch,
                        }
                    ]}
                    onConfirm={(confirmed) => executeRollback(confirmed)}
                />
            </Box>
        )}

        <Box flexDirection="column" marginTop={1} flexGrow={1} borderStyle="round" borderColor="magenta"
             paddingX={1} flexWrap="nowrap">
            <Box flexDirection="column" marginTop={1} flexGrow={1}>
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
                                    <Box key={`${r.id}-${r.revision}`}
                                         backgroundColor={active ? 'magentaBright' : undefined}>
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
                        <Text dimColor>j/k to move • d diff • p prune • Enter confirm • Esc/q cancel</Text>
                    </Box>
                </Box>
            </Box>
            <Box flexGrow={1}/>
        </Box>
    </>);
}

function filterRollbackRow(row: RollbackRow, f: string): boolean {
    const q = (f || '').toLowerCase();
    if (!q) return true;
    const fields = [String(row.id || ''), String(row.revision || ''), String(row.author || ''), String(row.date || ''), String(row.message || '')];
    return fields.some((s) => s.toLowerCase().includes(q));
}
