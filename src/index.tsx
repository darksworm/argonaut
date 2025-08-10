import React, {useEffect, useMemo, useState, useRef} from 'react';
import {render, Box, Text, useApp, useInput} from 'ink';
import TextInput from 'ink-text-input';
import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import chalk from 'chalk';
import stringWidth from 'string-width';
import {execa} from 'execa';
import {spawn as ptySpawn} from 'node-pty';
import ArgoNautBanner from "./banner";
import packageJson from '../package.json';
import type {AppItem, View, Mode} from './types/domain';
import type {ArgoCLIConfig} from './config/cli-config';
import {
    readCLIConfig as readCLIConfigExt,
    writeCLIConfig as writeCLIConfigExt,
    getCurrentServer as getCurrentServerExt
} from './config/cli-config';
import {
    ensureSSOLogin as ensureSSOLoginExt,
    tokenFromConfig as tokenFromConfigExt,
    ensureToken as ensureTokenExt
} from './auth/token';
import {getApiVersion as getApiVersionApi} from './api/version';
import {syncApp} from './api/applications.command';
import {canIRollback, getApplication as getAppApi, getSyncWindows as getSyncWindowsApi, getRevisionMetadata as getRevisionMetadataApi, getManifests as getManifestsApi, postRollback as postRollbackApi} from './api/rollback';
import {watchApps} from './api/applications.query';
import {useApps} from './hooks/useApps';
import {getManagedResourceDiffs} from './api/applications.query';

// Switch to terminal alternate screen on start, and restore on exit
(function setupAlternateScreen() {
    if (typeof process === 'undefined') return;
    const out = process.stdout as any;
    const isTTY = !!out && typeof out.isTTY === 'boolean' ? out.isTTY : false;
    if (!isTTY) return;

    let cleaned = false;
    const enable = () => {
        try {
            out.write("\u001B[?1049h");
        } catch {
        }
    };
    const disable = () => {
        if (cleaned) return;
        cleaned = true;
        try {
            out.write("\u001B[?1049l");
        } catch {
        }
    };

    enable();

    process.on('exit', disable);
    process.on('SIGINT', () => {
        disable();
        process.exit(130);
    });
    process.on('SIGTERM', () => {
        disable();
        process.exit(143);
    });
    process.on('SIGHUP', () => {
        disable();
        process.exit(129);
    });
    process.on('uncaughtException', (err) => {
        disable();
        console.error(err);
        process.exit(1);
    });
    process.on('unhandledRejection', (reason) => {
        disable();
        console.error(reason);
        process.exit(1);
    });
})();

// ------------------------------
// UI helpers
// ------------------------------

function colorFor(appState: string): { color?: any; dimColor?: boolean } {
    const v = (appState || '').toLowerCase();
    if (v === 'synced' || v === 'healthy') return {color: 'green'};
    if (v === 'outofsync' || v === 'degraded') return {color: 'red'};
    if (v === 'progressing' || v === 'warning' || v === 'suspicious') return {color: 'yellow'};
    if (v === 'unknown') return {dimColor: true};
    return {};
}

function humanizeSince(iso?: string): string {
    if (!iso) return '—';
    const t = new Date(iso).getTime();
    if (!Number.isFinite(t)) return '—';
    const s = Math.max(0, Math.floor((Date.now() - t) / 1000));
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m`;
    const h = Math.floor(m / 60);
    if (h < 24) return `${h}h`;
    const d = Math.floor(h / 24);
    if (d < 30) return `${d}d`;
    const mo = Math.floor(d / 30);
    if (mo < 12) return `${mo}mo`;
    const y = Math.floor(mo / 12);
    return `${y}y`;
}

function uniqueSorted<T>(arr: T[]): T[] {
    return Array.from(new Set(arr)).sort((a: any, b: any) => `${a}`.localeCompare(`${b}`));
}

function fmtScope(set: Set<string>, max = 2): string {
    if (!set.size) return '—';
    const arr = Array.from(set);
    if (arr.length <= max) return arr.join(',');
    return `${arr.slice(0, max).join(',')} (+${arr.length - max})`;
}

// Column widths — header and rows use the same numbers
const COL = {
    mark: 2,
    name: 36,
    sync: 4,
    health: 6,
} as const;

// Utilities for diff command
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

// ------------------------------
// Main Ink component
// ------------------------------

const App: React.FC = () => {
    const {exit} = useApp();

    // Layout
    const [termRows, setTermRows] = useState(process.stdout.rows || 24);
    const [termCols, setTermCols] = useState(process.stdout.columns || 80);

    useEffect(() => {
        const onResize = () => {
            setTermRows(process.stdout.rows || 24);
            setTermCols(process.stdout.columns || 80);
        };
        process.stdout.on('resize', onResize);
        return () => {
            process.stdout.off('resize', onResize);
        };
    }, []);

    // Modes & view
    const [mode, setMode] = useState<Mode>('loading');
    const [view, setView] = useState<View>('clusters');

    // Auth
    const [server, setServer] = useState<string | null>(null);
    const [token, setToken] = useState<string | null>(null);

    // Data
    const [apps, setApps] = useState<AppItem[]>([]);
    const [apiVersion, setApiVersion] = useState<string>('');

    // UI state
    const [searchQuery, setSearchQuery] = useState('');
    const [activeFilter, setActiveFilter] = useState('');
    const [command, setCommand] = useState(':');
    const [selectedIdx, setSelectedIdx] = useState(0);
    const [status, setStatus] = useState<string>('Starting…');

    // :login modal
    const [loginLog, setLoginLog] = useState<string>('');
    const [showLogin, setShowLogin] = useState(false);

    // Scopes / selections
    const [scopeClusters, setScopeClusters] = useState<Set<string>>(new Set());
    const [scopeNamespaces, setScopeNamespaces] = useState<Set<string>>(new Set());
    const [scopeProjects, setScopeProjects] = useState<Set<string>>(new Set());
    const [selectedApps, setSelectedApps] = useState<Set<string>>(new Set());
    const [confirmTarget, setConfirmTarget] = useState<string | null>(null);

    // Rollback flow state
    const [rollbackApp, setRollbackApp] = useState<string | null>(null);
    const [rollbackRows, setRollbackRows] = useState<any[]>([]);
    const [rollbackIdx, setRollbackIdx] = useState(0);
    const [rollbackFilter, setRollbackFilter] = useState('');
    const [rollbackError, setRollbackError] = useState<string>('');
    const metaAbortRef = useRef<AbortController | null>(null);
    const [rollbackFromRev, setRollbackFromRev] = useState<string | undefined>(undefined);
    const [rollbackDryRun, setRollbackDryRun] = useState(true);
    const [rollbackPrune, setRollbackPrune] = useState(false);
    const [rollbackHistoryId, setRollbackHistoryId] = useState<number | null>(null);
    const [rollbackProgressLog, setRollbackProgressLog] = useState<string[]>([]);
    const rollbackWatchAbortRef = useRef<AbortController | null>(null);
    const [rollbackEditingFilter, setRollbackEditingFilter] = useState(false);
    const [rollbackMetaLoadingKey, setRollbackMetaLoadingKey] = useState<string | null>(null);

    // Boot & auth
    useEffect(() => {
        (async () => {
            setMode('loading');
            setStatus('Loading ArgoCD config…');
            const cfg = await readCLIConfigExt();
            const srv = getCurrentServerExt(cfg);
            if (!srv) {
                setStatus('No server in config. Use :server <host[:port]> then :login');
                setMode('normal'); // show UI to allow entering commands
                return;
            }
            setServer(srv);

            try {
                const tokMaybe = await tokenFromConfigExt();
                if (!tokMaybe) throw new Error('No token in config');
                // Replace sanity request with API version check
                const version = await getApiVersionApi(srv, tokMaybe);
                setApiVersion(version);
                setToken(tokMaybe);
            } catch {
                setStatus(`Logging into ${srv} via SSO (grpc-web)…`);
                await ensureSSOLoginExt(srv);
                const tok = await ensureTokenExt(srv);
                setToken(tok);

                // Fetch API version after login
                try {
                    const version = await getApiVersionApi(srv, tok);
                    setApiVersion(version);
                } catch (e) {
                    console.error('Error fetching API version after login:', e);
                }
            }

            // Apps are handled by useApps hook
            setStatus('Ready');
            setMode('normal');
        })().catch(e => {
            setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`);
            setMode('normal');
        });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Live data via useApps hook
    const {apps: liveApps, status: appsStatus} = useApps(server, token, mode === 'external');

    useEffect(() => {
        if (!server || !token) return;
        if (mode === 'external') return; // pause syncing state while in external/diff mode
        setApps(liveApps);
        setStatus(appsStatus);
    }, [server, token, liveApps, appsStatus, mode]);

    // Periodic API version refresh (1 min)
    useEffect(() => {
        if (!server || !token) return;
        const id = setInterval(async () => {
            try {
                const v = await getApiVersionApi(server, token);
                setApiVersion(v);
            } catch {/* noop */
            }
        }, 60000);
        return () => clearInterval(id);
    }, [server, token]);

    const [confirmInput, setConfirmInput] = useState('');

    // Input
    useInput((input, key) => {
        if (mode === 'external') return;
        if (mode === 'help') {
            if (input === '?' || key.escape) setMode('normal');
            return;
        }
        if (mode === 'search') {
            if (key.escape) {
                setMode('normal');
                setSearchQuery('');
            }
            // Enter handled by TextInput onSubmit
            return;
        }

        // In normal mode, Escape clears the active filter
        if (mode === 'normal' && key.escape && activeFilter) {
            setActiveFilter('');
            return;
        }
        if (mode === 'command') {
            if (key.escape) {
                setMode('normal');
                setCommand(':');
            }
            return; // TextInput handles typing/enter
        }
        if (mode === 'confirm-sync') {
            // Esc or 'q' aborts immediately
            if (key.escape || input.toLowerCase() === 'q') {
                confirmSync(false);
                return;
            }
            // All other handling is done via TextInput onSubmit in the confirm dialog
            return;
        }
        if (mode === 'rollback') {
            if (key.escape || input === 'q') { setMode('normal'); return; }
            if (input === 'j' || key.downArrow) { setRollbackIdx(i => Math.min(i + 1, Math.max(0, rollbackRows.filter(r => filterRollbackRow(r, rollbackFilter)).length - 1))); return; }
            if (input === 'k' || key.upArrow) { setRollbackIdx(i => Math.max(i - 1, 0)); return; }
            if (input.toLowerCase() === 'd') { runRollbackDiff(); return; }
            if (input.toLowerCase() === 'c' || key.return) { setMode('rollback-confirm'); return; }
            return;
        }
        if (mode === 'rollback-confirm') {
            if (key.escape || input === 'q') { setMode('rollback'); return; }
            if (input.toLowerCase() === 'p') { setRollbackPrune(v => !v); return; }
            if (input.toLowerCase() === 'r') { setRollbackDryRun(v => !v); return; }
            if (input.toLowerCase() === 'c' || key.return) { executeRollback(true); return; }
            return;
        }
        if (mode === 'rollback-progress') {
            if (key.escape) {
                try { rollbackWatchAbortRef.current?.abort(); } catch {}
                setMode('normal');
                return;
            }
            return;
        }

        // normal
        if (input === '?') {
            setMode('help');
            return;
        }
        if (input === '/') {
            setMode('search');
            return;
        }
        if (input === ':') {
            setMode('command');
            setCommand(':');
            return;
        }

        if (input === 'j' || key.downArrow) setSelectedIdx(s => Math.min(s + 1, Math.max(0, visibleItems.length - 1)));
        if (input === 'k' || key.upArrow) setSelectedIdx(s => Math.max(s - 1, 0));

        // Esc clears current view selection
        if (key.escape) {
            if (view === 'clusters') setScopeClusters(new Set());
            else if (view === 'namespaces') setScopeNamespaces(new Set());
            else if (view === 'projects') setScopeProjects(new Set());
            else setSelectedApps(new Set());
            return;
        }

        if (key.return) drillDown();
        if (input === ' ') toggleSelection();
    });

    function toggleSelection() {
        const item = visibleItems[selectedIdx];
        if (item == null) return;
        if (view === 'clusters') {
            const val = String(item);
            // Only allow single selection - create a new Set with just this item or empty if already selected
            const next = scopeClusters.has(val) ? new Set() : new Set([val]);
            setScopeClusters(next);
            // Clear lower-level selections when cluster selection changes
            setScopeNamespaces(new Set());
            setScopeProjects(new Set());
            setSelectedApps(new Set());
        } else if (view === 'namespaces') {
            const ns = String(item);
            // Only allow single selection - create a new Set with just this item or empty if already selected
            const next = scopeNamespaces.has(ns) ? new Set() : new Set([ns]);
            setScopeNamespaces(next);
            // Clear lower-level selections when namespace selection changes
            setScopeProjects(new Set());
            setSelectedApps(new Set());
        } else if (view === 'projects') {
            const proj = String(item);
            // Only allow single selection - create a new Set with just this item or empty if already selected
            const next = scopeProjects.has(proj) ? new Set() : new Set([proj]);
            setScopeProjects(next);
            // Clear lower-level selections when project selection changes
            setSelectedApps(new Set());
        } else if (view === 'apps') {
            const app = (item as AppItem).name;
            const next = new Set(selectedApps);
            next.has(app) ? next.delete(app) : next.add(app);
            setSelectedApps(next);
        }
    }

    function drillDown() {
        const item = visibleItems[selectedIdx];
        if (item == null) return;
        if (view === 'clusters') {
            const val = String(item);
            // Only allow single selection - create a new Set with just this item
            const next = new Set([val]);
            setScopeClusters(next);
            // Clear lower-level selections when navigating from clusters
            setScopeNamespaces(new Set());
            setScopeProjects(new Set());
            setSelectedApps(new Set());
            // Clear search query and active filter when changing views
            setActiveFilter('');
            setSearchQuery('');
            setView('namespaces');
            setSelectedIdx(0);
            return;
        }
        if (view === 'namespaces') {
            const ns = String(item);
            // Only allow single selection - create a new Set with just this item
            const next = new Set([ns]);
            setScopeNamespaces(next);
            // Clear lower-level selections when navigating from namespaces
            setScopeProjects(new Set());
            setSelectedApps(new Set());
            // Clear search query and active filter when changing views
            setActiveFilter('');
            setSearchQuery('');
            setView('projects');
            setSelectedIdx(0);
            return;
        }
        if (view === 'projects') {
            const proj = String(item);
            // Only allow single selection - create a new Set with just this item
            const next = new Set([proj]);
            setScopeProjects(next);
            // Clear lower-level selections when navigating from projects
            setSelectedApps(new Set());
            // Clear search query and active filter when changing views
            setActiveFilter('');
            setSearchQuery('');
            setView('apps');
            setSelectedIdx(0);
            return;
        }
    }

    // Commands
    async function runCommand(line: string) {
        const raw = line.trim();
        if (!raw.startsWith(':')) return;
        const parts = raw.slice(1).trim().split(/\s+/);
        const cmd = (parts[0] || '').toLowerCase();
        const arg = parts.slice(1).join(' ');

        const alias = (s: string) => s.toLowerCase();
        const is = (c: string, ...as: string[]) => [c, ...as].map(alias).includes(cmd);

        if (is('q', 'quit', 'exit')) {
            exit();
            return;
        }
        if (is('help', '?')) {
            setMode('help');
            return;
        }

        if (is('cluster', 'clusters', 'cls')) {
            setView('clusters');
            setSelectedIdx(0);
            setMode('normal');
            if (arg) setScopeClusters(new Set([arg]));
            else setScopeClusters(new Set()); // Clear selection when returning to view
            setActiveFilter(''); // Clear active filter when changing views
            setSearchQuery(''); // Clear search query when changing views
            return;
        }
        if (is('namespace', 'namespaces', 'ns')) {
            setView('namespaces');
            setSelectedIdx(0);
            setMode('normal');
            if (arg) setScopeNamespaces(new Set([arg]));
            else setScopeNamespaces(new Set()); // Clear selection when returning to view
            setActiveFilter(''); // Clear active filter when changing views
            setSearchQuery(''); // Clear search query when changing views
            return;
        }
        if (is('project', 'projects', 'proj')) {
            setView('projects');
            setSelectedIdx(0);
            setMode('normal');
            if (arg) setScopeProjects(new Set([arg]));
            else setScopeProjects(new Set()); // Clear selection when returning to view
            setActiveFilter(''); // Clear active filter when changing views
            setSearchQuery(''); // Clear search query when changing views
            return;
        }
        if (is('app', 'apps')) {
            setView('apps');
            setSelectedIdx(0);
            setMode('normal');
            if (arg) setSelectedApps(new Set([arg]));
            else setSelectedApps(new Set()); // Clear selection when returning to view
            setActiveFilter(''); // Clear active filter when changing views
            return;
        }

        if (is('clear')) {
            if (view === 'clusters') setScopeClusters(new Set());
            else if (view === 'namespaces') setScopeNamespaces(new Set());
            else if (view === 'projects') setScopeProjects(new Set());
            else if (view === 'apps') setSelectedApps(new Set());
            setStatus('Selection cleared.');
            return;
        }

        if (is('server')) {
            const host = arg;
            if (!host) {
                setStatus('Usage: :server <host[:port]>');
                return;
            }
            const cfg = (await readCLIConfigExt()) ?? {} as any;
            const newCfg: ArgoCLIConfig = typeof cfg === 'object' && cfg ? cfg as ArgoCLIConfig : {} as ArgoCLIConfig;
            newCfg.contexts = [{name: host, server: host, user: host}];
            newCfg.servers = [{server: host, ['grpc-web']: true} as any];
            newCfg.users = [];
            newCfg['current-context'] = host;
            await writeCLIConfigExt(newCfg);
            setServer(host);
            setStatus(`Server set to ${host}. Run :login`);
            return;
        }

        if (is('login')) {
            if (!server) {
                setStatus('Set a server first: :server <host[:port]>.');
                return;
            }
            setShowLogin(true);
            setLoginLog('Opening browser for SSO…\n');
            try {
                const p = execa('argocd', ['login', server, '--sso', '--grpc-web']);
                p.stdout?.on('data', (b: Buffer) => setLoginLog(v => v + b.toString()));
                p.stderr?.on('data', (b: Buffer) => setLoginLog(v => v + b.toString()));
                await p;
                const tok = await ensureTokenExt(server);
                setToken(tok);

                // Fetch API version after login
                try {
                    const version = await getApiVersionApi(server, tok);
                    setApiVersion(version);
                } catch (e) {
                    console.error('Error fetching API version after login command:', e);
                }

                setStatus('Login OK.');
            } catch (e: any) {
                setStatus(`Login failed: ${e.message}`);
            } finally {
                setShowLogin(false);
            }
            return;
        }

        if (is('diff')) {
            const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || Array.from(selectedApps)[0];
            if (!target) {
                setStatus('No app selected to diff.');
                return;
            }
            if (!server || !token) {
                setStatus('Not authenticated.');
                return;
            }

            try {
                setMode('normal');
                setStatus(`Preparing diff for ${target}…`);

                const diffs = await getManagedResourceDiffs(server, token, target).catch(() => [] as any[]);
                const desiredDocs: string[] = [];
                const liveDocs: string[] = [];
                for (const d of diffs as any[]) {
                    const tgt = toYamlDoc(d?.targetState);
                    const live = toYamlDoc(d?.liveState);
                    if (tgt) desiredDocs.push(tgt);
                    if (live) liveDocs.push(live);
                }
                const desiredFile = await writeTmp(desiredDocs, `${target}-desired`);
                const liveFile = await writeTmp(liveDocs, `${target}-live`);

                try {
                    await execa('git', ['--no-pager', 'diff', '--no-index', '--quiet', '--', desiredFile, liveFile]);
                    setStatus('No differences.');
                    return;
                } catch { /* has diffs: continue */
                }

                const shell = 'bash';
                const cols = (process.stdout as any)?.columns || 80;
                const pager = process.platform === 'darwin' ? "less -r -+X -K" : "less -R -+X -K";

                const cmd = `
set -e
if command -v delta >/dev/null 2>&1; then
  # Use delta directly; force paging, keep colors; ignore nonzero exit meaning "has diffs"
  DELTA_PAGER='${pager}' delta --paging=always --line-numbers --side-by-side --width=${cols} "${liveFile}" "${desiredFile}"|| true
else
  # Fallback: git diff with colors piped to less (mac uses -r, linux uses -R)
  PAGER='${pager}'
  if ! command -v less >/dev/null 2>&1; then
    PAGER='sh -c "cat; printf \\"\\n[Press Enter to close] \\"; read -r _"'
  fi
  git --no-pager diff --no-index --color=always -- "${desiredFile}" "${liveFile}" | eval "$PAGER" || true
fi
`;      // --- PTY session (no alt-screen toggling here) ---
                setMode('external');

                const args = process.platform === 'win32'
                    ? ['-NoProfile', '-NonInteractive', '-Command', cmd]
                    : ['-lc', cmd];

                const rows = (process.stdout as any)?.rows || 24;

                const pty = ptySpawn(shell, args as any, {
                    name: 'xterm-256color',
                    cols, rows,
                    cwd: process.cwd(),
                    env: {...(process.env as any), COLORTERM: 'truecolor'} as any
                });

                const onResize = () => {
                    try {
                        pty.resize((process.stdout as any)?.columns || 80, (process.stdout as any)?.rows || 24);
                    } catch {
                    }
                };
                const onPtyData = (data: string) => {
                    try {
                        process.stdout.write(data);
                    } catch {
                    }
                };
                pty.onData(onPtyData);
                process.stdout.on('resize', onResize);

                const stdinAny = process.stdin as any;
                try {
                    stdinAny.resume?.();
                    stdinAny.setRawMode?.(true);
                } catch {
                }
                const onStdin = (chunk: Buffer) => {
                    try {
                        pty.write(chunk.toString('utf8'));
                    } catch {
                    }
                };
                process.stdin.on('data', onStdin);

                await new Promise<void>((resolve) => {
                    pty.onExit(() => resolve());
                });

                // cleanup
                try {
                    process.stdin.off('data', onStdin);
                    process.stdout.off('resize', onResize);
                } catch {
                }
                try {
                    stdinAny.setRawMode?.(true);
                    stdinAny.resume?.();
                } catch {
                }

                setMode('normal');
                setStatus(`Diff closed for ${target}.`);
            } catch (e: any) {
                try {
                    const stdinAny = process.stdin as any;
                    stdinAny.setRawMode?.(true);
                    stdinAny.resume?.();
                } catch {
                }
                setMode('normal');
                setStatus(`Diff failed: ${e?.message || String(e)}`);
            }
            return;
        }

        if (is('rollback')) {
            const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || Array.from(selectedApps)[0];
            if (!target) { setStatus('No app selected to rollback.'); return; }
            if (!server || !token) { setStatus('Not authenticated.'); return; }
            await openRollbackFlow(target);
            return;
        }

        if (is('sync')) {
            // Prefer explicit arg; otherwise if multiple apps are selected, confirm multi-sync.
            if (arg) {
                setConfirmTarget(arg);
                setConfirmInput('');
                setMode('confirm-sync');
                return;
            }
            if (selectedApps.size > 1) {
                setConfirmTarget(`__MULTI__`);
                setConfirmInput('');
                setMode('confirm-sync');
                return;
            }
            // Fallback to current cursor app (apps view) or the single selected app (if exactly one is selected)
            const target = (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined)
                || (selectedApps.size === 1 ? Array.from(selectedApps)[0] : undefined);
            if (!target) {
                setStatus('No app selected to sync.');
                return;
            }
            setConfirmTarget(target);
            setConfirmInput('');
            setMode('confirm-sync');
            return;
        }

        if (is('all')) {
            // Clear all selections
            setScopeClusters(new Set());
            setScopeNamespaces(new Set());
            setScopeProjects(new Set());
            setSelectedApps(new Set());
            // Clear filters
            setActiveFilter('');
            setSearchQuery('');
            setStatus('All filtering cleared.');
            return;
        }

        setStatus(`Unknown command: ${cmd}`);
    }

    async function confirmSync(yes: boolean) {
        setMode('normal');
        const isMulti = confirmTarget === '__MULTI__';
        const names = isMulti ? Array.from(selectedApps) : [confirmTarget!];
        setConfirmTarget(null);
        if (!yes) {
            setStatus('Sync cancelled.');
            return;
        }
        if (!server || !token) {
            setStatus('Not authenticated.');
            return;
        }
        try {
            setStatus(`Syncing ${isMulti ? `${names.length} app(s)` : names[0]}…`);
            for (const n of names) syncApp(server, token, n);
            setStatus(`Sync initiated for ${isMulti ? `${names.length} app(s)` : names[0]}.`);
            // After syncing multiple apps, clear the selection
            if (isMulti) {
                setSelectedApps(new Set());
            }
        } catch (e: any) {
            setStatus(`Sync failed: ${e.message}`);
        }
    }

    // ---------- Rollback helpers ----------
    async function openRollbackFlow(appName: string) {
        try {
            setStatus(`Opening rollback for ${appName}…`);
            const app = await getAppApi(server!, token!, appName).catch(()=>({} as any));
            const fromRev = app?.status?.sync?.revision || '';
            setRollbackFromRev(fromRev || undefined);
            const hist = Array.isArray(app?.status?.history) ? [...(app.status!.history!)] : [];
            const rows = hist
                .map(h => ({ id: Number(h?.id ?? 0), revision: String(h?.revision || ''), deployedAt: h?.deployedAt }))
                .filter(h => h.id > 0 && h.revision)
                .sort((a,b)=> b.id - a.id);
            if (rows.length === 0) {
                setStatus('No previous syncs found.');
                setRollbackError('No previous syncs found.');
                setRollbackApp(appName);
                setRollbackRows([]);
                setRollbackIdx(0);
                setRollbackFilter('');
                setMode('rollback');
                return;
            }
            setRollbackError('');
            setRollbackApp(appName);
            setRollbackRows(rows);
            setRollbackIdx(0);
            setRollbackFilter('');
            setMode('rollback');
        } catch (e: any) {
            setRollbackError(e?.message || String(e));
            setRollbackApp(appName);
            setRollbackRows([]);
            setRollbackIdx(0);
            setRollbackFilter('');
            setMode('rollback');
        }
    }

    function shortSha(s?: string) { return (s || '').slice(0,7); }
    function singleLine(input?: string): string {
        const s = String(input || '');
        // Replace newlines/tabs with spaces and collapse multiple spaces
        return s.replace(/[\r\n\t]+/g, ' ').replace(/\s{2,}/g, ' ').trim();
    }
    function filterRollbackRow(row: any, f: string): boolean {
        const q = (f || '').toLowerCase();
        if (!q) return true;
        const fields = [String(row.id||''), String(row.revision||''), String(row.author||''), String(row.date||''), String(row.message||'')];
        return fields.some(s => s.toLowerCase().includes(q));
    }

    async function runRollbackDiff() {
        if (!server || !token || !rollbackApp) { setStatus('Not authenticated.'); return; }
        const row: any = rollbackRows[rollbackIdx];
        if (!row) { setStatus('No selection to diff.'); return; }
        try {
            setStatus(`Preparing diff for ${rollbackApp} (${shortSha(row.revision)})…`);
            const current = await getManifestsApi(server, token, rollbackApp).catch(()=>[]);
            const target = await getManifestsApi(server, token, rollbackApp, row.revision).catch(()=>[]);
            const currentDocs = current.map(toYamlDoc).filter(Boolean) as string[];
            const targetDocs = target.map(toYamlDoc).filter(Boolean) as string[];
            const currentFile = await writeTmp(currentDocs, `${rollbackApp}-current`);
            const targetFile = await writeTmp(targetDocs, `${rollbackApp}-target-${row.id}`);

            // Try quiet diff first, bail if no diffs
            try { await execa('git', ['--no-pager','diff','--no-index','--quiet','--', currentFile, targetFile]); setStatus('No differences.'); return; } catch {}

            const shell = 'bash';
            const cols = (process.stdout as any)?.columns || 80;
            const pager = process.platform === 'darwin' ? "less -r -+X -K" : "less -R -+X -K";
            const cmd = `
set -e
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
            setMode('external');
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
            setMode('rollback');
            setStatus('Diff closed.');
        } catch (e:any) {
            try { const stdinAny = process.stdin as any; stdinAny.setRawMode?.(true); stdinAny.resume?.(); } catch {}
            setMode('rollback');
            setStatus(`Diff failed: ${e?.message || String(e)}`);
        }
    }

    async function executeRollback(confirm: boolean) {
        if (!confirm) { setMode('rollback'); setStatus('Rollback cancelled.'); return; }
        const row: any = rollbackRows[rollbackIdx];
        if (!server || !token || !rollbackApp || !row) { setStatus('Not ready.'); return; }
        try {
            setStatus(`Rollback ${rollbackApp} to ${shortSha(row.revision)}${rollbackDryRun?' (dry-run)':''}…`);
            setRollbackHistoryId(row.id);
            await postRollbackApi(server, token, rollbackApp, { id: row.id, name: rollbackApp, dryRun: rollbackDryRun, prune: rollbackPrune });
            if (rollbackDryRun) {
                setMode('rollback-confirm');
                setRollbackError('Dry run completed.');
                return;
            }
            // Start streaming progress
            setMode('rollback-progress');
            setRollbackProgressLog([]);
            try { rollbackWatchAbortRef.current?.abort(); } catch {}
            const ac = new AbortController(); rollbackWatchAbortRef.current = ac;
            (async () => {
                try {
                    for await (const evt of watchApps(server!, token!, undefined, ac.signal)) {
                        const app = evt?.application; const name = app?.metadata?.name;
                        if (name !== rollbackApp) continue;
                        const phase = app?.status?.operationState?.phase || '';
                        const msg = app?.status?.operationState?.message || '';
                        const h = app?.status?.health?.status || '';
                        const s = app?.status?.sync?.status || '';
                        setRollbackProgressLog(log => [...log, `[${new Date().toISOString()}] ${phase||s} ${h} ${msg}`].slice(-200));
                        if ((h === 'Healthy' && s === 'Synced') || phase === 'Failed' || phase === 'Error') {
                            try { ac.abort(); } catch {}
                            break;
                        }
                    }
                } catch (e:any) {
                    if (!ac.signal.aborted) setRollbackProgressLog(log => [...log, `Stream error: ${e?.message||String(e)}`]);
                } finally {
                    // Close overlay on success
                    setMode('normal');
                    setStatus(`Rollback to ${shortSha(row.revision)} ${rollbackApp ? 'completed' : ''}`);
                }
            })();
        } catch (e:any) {
            setRollbackError(e?.message || String(e));
            setMode('rollback-confirm');
        }
    }

    // ---------- Rollback overlays UI ----------
    const rollbackOverlay = (mode === 'rollback') && (
        <Box paddingX={1} flexDirection="column">
            <Text bold>Rollback: <Text color="magentaBright">{rollbackApp}</Text></Text>
            <Box marginTop={1}>
                <Text>Current revision: <Text color="cyan">{shortSha(rollbackFromRev)}</Text></Text>
            </Box>
            {rollbackError && (
                <Box marginTop={1}><Text color="red">{rollbackError}</Text></Box>
            )}
            <Box marginTop={1} flexDirection="column">
                <Box>
                    <Box width={6}><Text bold>ID</Text></Box>
                    <Box width={10}><Text bold>Revision</Text></Box>
                    <Box width={20}><Text bold>Deployed</Text></Box>
                    <Box flexGrow={1}><Text bold>Message</Text></Box>
                </Box>
                {(() => {
                    const rows = rollbackRows.filter(r => filterRollbackRow(r, rollbackFilter));
                    const maxRows = Math.max(1, Math.min(10, rows.length));
                    const start = Math.max(0, Math.min(rollbackIdx - Math.floor(maxRows/2), Math.max(0, rows.length - maxRows)));
                    const slice = rows.slice(start, start + maxRows);
                    return slice.map((r:any, i:number) => {
                        const actual = start + i;
                        const active = actual === rollbackIdx;
                        return (
                            <Box key={`${r.id}-${r.revision}`} backgroundColor={active ? 'magentaBright' : undefined}>
                                <Box width={6}><Text>{String(r.id)}</Text></Box>
                                <Box width={10}><Text>{shortSha(r.revision)}</Text></Box>
                                <Box width={20}><Text>{r.deployedAt ? humanizeSince(r.deployedAt) + ' ago' : '—'}</Text></Box>
                                <Box flexGrow={1}><Text wrap="truncate-end">{(rollbackMetaLoadingKey === `${rollbackApp}:${r.id}:${r.revision}`) ? '(loading…)' : singleLine(r.message || r.metaError || '')}</Text></Box>
                            </Box>
                        );
                    });
                })()}
            </Box>
            <Box marginTop={1}><Text dimColor>j/k to move • d diff • c confirm • Esc/q cancel</Text></Box>
        </Box>
    );

    const rollbackConfirmOverlay = (mode === 'rollback-confirm') && (() => {
        const row: any = rollbackRows[rollbackIdx];
        return (
            <Box paddingX={2} flexDirection="column">
                <Text bold>Confirm rollback</Text>
                <Box marginTop={1}><Text>App: <Text color="magentaBright">{rollbackApp}</Text></Text></Box>
                <Box><Text>From: <Text color="cyan">{shortSha(rollbackFromRev)}</Text> → To: <Text color="cyan">{row ? shortSha(row.revision) : '—'}</Text></Text></Box>
                <Box><Text>History ID: <Text color="cyan">{row?.id ?? '—'}</Text></Text></Box>
                {rollbackError && (
                    <Box marginTop={1}><Text color="red">{rollbackError}</Text></Box>
                )}
                <Box marginTop={1}><Text>Dry-run [r]: <Text color={rollbackDryRun ? 'green' : 'yellow'}>{rollbackDryRun ? 'on' : 'off'}</Text> • Prune [p]: <Text color={rollbackPrune ? 'yellow' : undefined}>{rollbackPrune ? 'on' : 'off'}</Text></Text></Box>
                <Box marginTop={1}><Text dimColor>Press c to confirm, Esc/q to go back</Text></Box>
            </Box>
        );
    })();

    const rollbackProgressOverlay = (mode === 'rollback-progress') && (
        <Box paddingX={2} flexDirection="column">
            <Text bold>Rollback in progress: <Text color="magentaBright">{rollbackApp}</Text></Text>
            <Box marginTop={1} flexDirection="column">
                {rollbackProgressLog.slice(-Math.max(5, Math.min(20, rollbackProgressLog.length))).map((l, i) => (
                    <Text key={i} dimColor>{l}</Text>
                ))}
            </Box>
            <Box marginTop={1}><Text dimColor>Esc to close</Text></Box>
        </Box>
    );

    // ---------- Derive scopes from apps ----------
    const allClusters = useMemo(() => {
        const vals = apps.map(a => a.clusterLabel || '').filter(Boolean);
        return uniqueSorted(vals);
    }, [apps]);

    const filteredByClusters = useMemo(() => {
        if (!scopeClusters.size) return apps;
        const allowed = new Set(scopeClusters);
        return apps.filter(a => allowed.has(a.clusterLabel || ''));
    }, [apps, scopeClusters]);

    const allNamespaces = useMemo(() => {
        const nss = filteredByClusters.map(a => a.namespace || '').filter(Boolean);
        return uniqueSorted(nss);
    }, [filteredByClusters]);

    const filteredByNs = useMemo(() => {
        if (!scopeNamespaces.size) return filteredByClusters;
        const allowed = new Set(scopeNamespaces);
        return filteredByClusters.filter(a => allowed.has(a.namespace || ''));
    }, [filteredByClusters, scopeNamespaces]);

    const allProjects = useMemo(() => {
        const projs = filteredByNs.map(a => a.project || '').filter(Boolean);
        return uniqueSorted(projs);
    }, [filteredByNs]);

    const visibleItems = useMemo(() => {
        // Use activeFilter when in normal mode, otherwise use searchQuery
        const f = (mode === 'search' ? searchQuery : activeFilter).toLowerCase();
        let base: any[];

        if (view === 'clusters') base = allClusters;
        else if (view === 'namespaces') base = allNamespaces;
        else if (view === 'projects') base = allProjects;
        else base = filteredByNs.filter(a => !scopeProjects.size || scopeProjects.has(a.project || ''));

        if (!f) return base;

        return view === 'apps'
            ? base.filter(a =>
                a.name.toLowerCase().includes(f) ||
                (a.sync || '').toLowerCase().includes(f) ||
                (a.health || '').toLowerCase().includes(f) ||
                (a.namespace || '').toLowerCase().includes(f) ||
                (a.project || '').toLowerCase().includes(f)
            )
            : base.filter(s => String(s).toLowerCase().includes(f));
    }, [view, allClusters, allNamespaces, allProjects, filteredByNs, scopeProjects, searchQuery, activeFilter, mode]);

    useEffect(() => {
        setSelectedIdx(s => Math.min(s, Math.max(0, visibleItems.length - 1)));
    }, [visibleItems.length]);

    // Fetch revision metadata for highlighted rollback row (debounced via AbortController)
    useEffect(() => {
        if (mode !== 'rollback') return;
        if (!server || !token || !rollbackApp) return;
        const row: any = rollbackRows[rollbackIdx];
        if (!row || row.author) return;
        try { metaAbortRef.current?.abort(); } catch {}
        const ac = new AbortController(); metaAbortRef.current = ac;
        const key = `${rollbackApp}:${row.id}:${row.revision}`;
        setRollbackMetaLoadingKey(key);
        (async () => {
            try {
                const meta = await getRevisionMetadataApi(server, token, rollbackApp, row.revision, ac.signal);
                const upd = [...rollbackRows];
                upd[rollbackIdx] = { ...row, author: meta?.author, date: meta?.date, message: meta?.message };
                setRollbackRows(upd);
            } catch (e: any) {
                const upd = [...rollbackRows];
                upd[rollbackIdx] = { ...row, metaError: e?.message || String(e) };
                setRollbackRows(upd);
            } finally {
                setRollbackMetaLoadingKey(prev => prev === key ? null : prev);
            }
        })();
        return () => { try { ac.abort(); } catch {} };
    }, [mode, rollbackIdx, rollbackRows, rollbackApp, server, token]);

    // ---------- Height calc (full-screen, exact) ----------
    const BORDER_LINES = 2;
    // Reserve enough lines for the top banner/logo (ASCII logo is 6 lines in wide mode)
    const HEADER_CONTEXT = 6;
    const SEARCH_LINES = (mode === 'search') ? 1 : 0;
    const TABLE_HEADER_LINES = 1;
    const TAG_LINE = 1;      // <clusters>
    const STATUS_LINES = 1;
    const COMMAND_LINES = (mode === 'command') ? 1 : 0;

    const OVERHEAD = BORDER_LINES + HEADER_CONTEXT + SEARCH_LINES + TABLE_HEADER_LINES + TAG_LINE + STATUS_LINES + COMMAND_LINES;

    const availableRows = Math.max(0, termRows - OVERHEAD);
    // When the command or search bar is open, show one less app row to avoid pushing the last row into the border area
    const barOpenExtra = (mode === 'search' || mode === 'command') ? 1 : 0;
    const listRows = Math.max(0, availableRows - barOpenExtra);
    const start = Math.max(0, Math.min(Math.max(0, selectedIdx - Math.floor(listRows / 2)), Math.max(0, visibleItems.length - listRows)));
    const end = Math.min(visibleItems.length, start + listRows);
    const rowsSlice = visibleItems.slice(start, end);

    const tag = activeFilter && view === 'apps' ? `<${view}:${activeFilter}>` : `<${view}>`;

    const helpOverlay = (
        <Box flexDirection="column" paddingX={2} paddingY={1}>
            <Box justifyContent="center"><Text color="magentaBright" paddingRight={1}
                                               bold>Argonaut {packageJson.version}</Text></Box>
            <Box marginTop={1}>
                <Box width={24}><Text color="green" bold>GENERAL</Text></Box>
                <Box><Text><Text color="cyan">:</Text> command • <Text color="cyan">/</Text> search • <Text
                    color="cyan">?</Text> help</Text></Box>
            </Box>
            <Box marginTop={1}>
                <Box width={24}><Text color="green" bold>NAV</Text></Box>
                <Box><Text><Text color="cyan">j/k</Text> up/down • <Text color="cyan">Space</Text> select • <Text
                    color="cyan">Enter</Text> drill down</Text></Box>
            </Box>
            <Box marginTop={1}>
                <Box width={24}><Text color="green" bold>VIEWS</Text></Box>
                <Box><Text>:cls|:clusters|:cluster • :ns|:namespaces|:namespace • :proj|:projects|:project •
                    :apps</Text></Box>
            </Box>
            <Box marginTop={1}>
                <Box width={24}><Text color="green" bold>ACTIONS</Text></Box>
                <Box><Text>:sync [app] • :rollback [app]</Text></Box>
            </Box>
            <Box marginTop={1}>
                <Box width={24}><Text color="green" bold>MISC</Text></Box>
                <Box><Text>:server HOST[:PORT] • :login • :clear • :all • :q</Text></Box>
            </Box>
            <Box marginTop={1}><Text dimColor>Press ? or Esc to close</Text></Box>
        </Box>
    );

    // Loading screen fills the viewport
    if (mode === 'loading') {
        const spinChar = '⠋';
        return (
            <Box flexDirection="column" borderStyle="round" borderColor="magenta" paddingX={1} height={termRows - 1}>
                <Box><Text>{chalk.bold(`View:`)} {chalk.yellow('LOADING')} • {chalk.bold(`Context:`)} {chalk.cyan(server || '—')}</Text></Box>
                <Box flexGrow={1} alignItems="center" justifyContent="center">
                    <Text color="yellow">{spinChar} Connecting & fetching applications…</Text>
                </Box>
                <Box><Text dimColor>{status}</Text></Box>
            </Box>
        );
    }

    // While in external diff mode, pause rendering the React app entirely
    if (mode === 'external') {
        return null;
    }

    // Full-screen rollback overlays: occupy whole screen and hide the apps UI, but keep header
    if (mode === 'rollback' || mode === 'rollback-confirm' || mode === 'rollback-progress') {
        return (
            <Box flexDirection="column" paddingX={1} height={termRows - 1}>
                <ArgoNautBanner
                    server={server}
                    clusterScope={fmtScope(scopeClusters)}
                    namespaceScope={fmtScope(scopeNamespaces)}
                    projectScope={fmtScope(scopeProjects)}
                    termCols={termCols}
                    termRows={availableRows}
                    apiVersion={apiVersion}
                    argonautVersion={packageJson.version}
                />
                <Box flexDirection="column" marginTop={1} flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1} flexWrap="nowrap">
                    <Box flexDirection="column" marginTop={1} flexGrow={1}>
                        {rollbackOverlay || rollbackConfirmOverlay || rollbackProgressOverlay}
                    </Box>
                    <Box flexGrow={1}/>
                </Box>
            </Box>
        );
    }

    const GUTTER = 1;
    const MIN_NAME = 12;
    const Sep = () => <Box width={GUTTER} />;

    // Single-cell icons (text variant) and ASCII fallback
    const ASCII_ICONS = {
        check: 'V',
        warn: '!',
        quest: '?',
        delta: '^',
    } as const;

    const SYNC_LABEL: Record<string,string> = { Synced:'Synced', OutOfSync:'OutOfSync', Unknown:'Unknown', Degraded:'Degraded' };
    const HEALTH_LABEL: Record<string,string> = { Healthy:'Healthy', Missing:'Missing', Degraded:'Degraded', Progressing:'Progressing', Unknown:'Unknown' };

    // width-aware right pad (right align inside fixed cells)
    const rightPadTo = (s: string, width: number) => {
        const w = stringWidth(s);
        return w >= width ? s : ' '.repeat(width - w) + s;
    };

    const SYNC_WIDE = 11; // width when showing icon + label
    const HEALTH_WIDE = 14; // width when showing icon + label
    const overhead = 6; // borders/padding fudge

    // Compute if we can show labels based on wide widths
    const fixedNoLastWide = SYNC_WIDE + GUTTER + HEALTH_WIDE;

    const canShowLabels = (cols:number) =>
        cols >= MIN_NAME + fixedNoLastWide + (2*GUTTER) + overhead + 15;

    // Effective column widths depending on whether labels are shown
    const SYNC_COL = canShowLabels(termCols) ? SYNC_WIDE : COL.sync;
    const HEALTH_COL = canShowLabels(termCols) ? HEALTH_WIDE : COL.health;

    const SYNC_ICON_ASCII: Record<string,string> = {
        Synced: ASCII_ICONS.check,
        OutOfSync: ASCII_ICONS.delta,
        Unknown: ASCII_ICONS.quest,
        Degraded: ASCII_ICONS.warn,
    };

    const HEALTH_ICON_ASCII: Record<string,string> = {
        Healthy: ASCII_ICONS.check,
        Missing: ASCII_ICONS.quest,
        Degraded: ASCII_ICONS.warn,
        Progressing: '.',
        Unknown: ASCII_ICONS.quest,
    };

    const getSyncIcon = (s: string) => SYNC_ICON_ASCII[s];
    const getHealthIcon = (h: string) => HEALTH_ICON_ASCII[h] ?? ASCII_ICONS.quest;

    return (
        <Box flexDirection="column" paddingX={1} height={termRows - 1}>
            <ArgoNautBanner
                server={server}
                clusterScope={fmtScope(scopeClusters)}
                namespaceScope={fmtScope(scopeNamespaces)}
                projectScope={fmtScope(scopeProjects)}
                termCols={termCols}
                termRows={availableRows}
                apiVersion={apiVersion}
                argonautVersion={packageJson.version}
            />

            {/* Search bar */}
            {mode === 'search' && (
                <Box borderStyle="round" borderColor="yellow" paddingX={1}>
                    <Text bold color="cyan">Search</Text>
                    <Box width={1}/>
                    <TextInput
                        value={searchQuery}
                        onChange={setSearchQuery}
                        onSubmit={() => {
                            setSelectedIdx(0);
                            setMode('normal');
                            if (visibleItems.length > 0) {
                                if (view === 'apps') setActiveFilter(searchQuery);
                                else drillDown();
                            }
                        }}
                    />
                    <Box width={2}/>
                    <Text dimColor>(Enter {view === 'apps' ? 'keeps filter' : 'opens first result'}, Esc cancels)</Text>
                </Box>
            )}

            {mode === 'command' && (
                <Box borderStyle="round" borderColor="yellow" paddingX={1}>
                    <Text bold color="cyan">CMD</Text>
                    <Box width={1}/>
                    <TextInput
                        value={command}
                        onChange={setCommand}
                        onSubmit={(val) => { setMode('normal'); runCommand(val); setCommand(':'); }}
                    />
                    <Box width={2}/>
                    <Text dimColor>(Enter to run, Esc to cancel)</Text>
                </Box>
            )}

            {(mode === 'rollback' || mode === 'rollback-confirm' || mode === 'rollback-progress') && (
                <Box flexDirection="column" marginTop={1} flexGrow={1}>
                    {rollbackOverlay || rollbackConfirmOverlay || rollbackProgressOverlay}
                </Box>
            )}

            {/* Confirm sync popup */}
            {mode === 'confirm-sync' && (
                <Box borderStyle="round" borderColor="yellow" paddingX={2} paddingY={1} flexDirection="column">
                    {confirmTarget === '__MULTI__' ? (
                        <>
                            <Text bold>Sync applications?</Text>
                            <Box>
                                <Text>Do you want to sync <Text color="magentaBright" bold>({selectedApps.size})</Text> applications? (y/n): </Text>
                                <TextInput
                                    value={confirmInput}
                                    onChange={(val) => {
                                        const filtered = (val || '').replace(/[^a-zA-Z]/g, '').toLowerCase();
                                        // Allow only prefixes of y/yes or n/no
                                        if (/^(y(es?)?|n(o?)?)?$/.test(filtered)) {
                                            // Limit to max length of the longest allowed word
                                            setConfirmInput(filtered.slice(0, 3));
                                        }
                                        // else ignore invalid characters (do not update state)
                                    }}
                                    onSubmit={(val) => {
                                        const t = (val || '').trim().toLowerCase();
                                        // reset input each submit
                                        setConfirmInput('');
                                        if (t === 'y' || t === 'yes') {
                                            confirmSync(true);
                                            return;
                                        }
                                        if (t === 'n' || t === 'no') {
                                            confirmSync(false);
                                            return;
                                        }
                                        if (t === '') {
                                            // Do nothing on empty submit
                                            return;
                                        }
                                        // Ignore any other input, stay in confirm mode
                                    }}
                                />
                            </Box>
                        </>
                    ) : (
                        <>
                            <Text bold>Sync application?</Text>
                            <Box marginTop={1}>
                                <Text>Do you want to sync <Text color="magentaBright" bold>{confirmTarget}</Text>? (y/n): </Text>
                                <TextInput
                                    value={confirmInput}
                                    onChange={(val) => {
                                        const filtered = (val || '').replace(/[^a-zA-Z]/g, '').toLowerCase();
                                        // Allow only prefixes of y/yes or n/no
                                        if (/^(y(es?)?|n(o?)?)?$/.test(filtered)) {
                                            // Limit to max length of the longest allowed word
                                            setConfirmInput(filtered.slice(0, 3));
                                        }
                                        // else ignore invalid characters (do not update state)
                                    }}
                                    onSubmit={(val) => {
                                        const t = (val || '').trim().toLowerCase();
                                        // reset input each submit
                                        setConfirmInput('');
                                        if (t === 'y' || t === 'yes') {
                                            confirmSync(true);
                                            return;
                                        }
                                        if (t === 'n' || t === 'no') {
                                            confirmSync(false);
                                            return;
                                        }
                                        if (t === '') {
                                            // Do nothing on empty submit
                                            return;
                                        }
                                        // Ignore any other input, stay in confirm mode
                                    }}
                                />
                            </Box>
                        </>
                    )}
                </Box>
            )}

            {/* Content area (fills space) */}
            <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1} flexWrap="nowrap">
                {mode === 'help' ? (
                    <Box flexDirection="column" marginTop={1} flexGrow={1}>{helpOverlay}</Box>
                ) : (
                    <Box flexDirection="column">
                        {/* Header row */}
                        {(() => {
                            return (
                                <Box width="100%">
                                    {/* NAME → flexible */}
                                    <Box flexGrow={1} flexShrink={1} minWidth={0}>
                                        <Text bold color="yellowBright" wrap="truncate">NAME</Text>
                                    </Box>

                                    {view === 'apps' && (
                                        <>
                                            <Sep/>
                                            <Box width={SYNC_COL} justifyContent="flex-end">
                                                <Text bold color="yellowBright" wrap="truncate">{"SYNC"}</Text>
                                            </Box>
                                            <Sep/>
                                            <Box width={HEALTH_COL} justifyContent="flex-end">
                                                <Text bold color="yellowBright" wrap="truncate">{'HEALTH'}</Text>
                                            </Box>
                                        </>
                                    )}
                                </Box>
                            );
                        })()}

                        {/* Rows */}
                        {rowsSlice.map((it:any, i:number) => {
                            const actualIndex = start + i;
                            const isCursor = actualIndex === selectedIdx;

                            if (view === 'apps') {
                                const a = it as AppItem;
                                const isChecked = selectedApps.has(a.name);
                                const active = isCursor || isChecked;

                                return (
                                    <Box key={a.name} width="100%" backgroundColor={active ? 'magentaBright' : undefined} flexWrap="nowrap" justifyContent={"flex-start"}>
                                        {/* NAME (flex) */}
                                        <Box flexGrow={1} flexShrink={1} minWidth={0}>
                                            <Text wrap="truncate-end">{a.name}</Text>
                                        </Box>

                                        {/* SYNC (fixed, right-aligned) */}
                                        <Sep/>
                                        <Box width={SYNC_COL} flexShrink={0} justifyContent="flex-end">
                                            <Text {...colorFor(a.sync)}>
                                                {rightPadTo(
                                                    canShowLabels(termCols)
                                                        ? `${getSyncIcon(a.sync)} ${SYNC_LABEL[a.sync] ?? ''}`
                                                        : `${getSyncIcon(a.sync)}`,
                                                    SYNC_COL
                                                )}
                                            </Text>
                                        </Box>

                                        {/* HEALTH (fixed, right-aligned) */}
                                        <Sep/>
                                        <Box width={HEALTH_COL} flexShrink={0} justifyContent="flex-end">
                                            <Text {...colorFor(a.health)}>
                                                {rightPadTo(
                                                    canShowLabels(termCols)
                                                        ? `${getHealthIcon(a.health)} ${HEALTH_LABEL[a.health] ?? ''}`
                                                        : `${getHealthIcon(a.health)}`,
                                                    HEALTH_COL
                                                )}
                                            </Text>
                                        </Box>
                                    </Box>
                                );
                            }

                            // clusters / namespaces / projects → single flex column
                            const label = String(it);
                            const isChecked =
                                (view === 'clusters'   && scopeClusters.has(label)) ||
                                (view === 'namespaces' && scopeNamespaces.has(label)) ||
                                (view === 'projects'   && scopeProjects.has(label));
                            const active = isCursor || isChecked;

                            return (
                                <Box key={label} width="100%" backgroundColor={active ? 'magentaBright' : undefined}>
                                    <Box flexGrow={1} flexShrink={1} minWidth={0}>
                                        <Text wrap="truncate-end">{label}</Text>
                                    </Box>
                                </Box>
                            );
                        })}

                        {visibleItems.length === 0 && (
                            <Box paddingY={1} paddingX={2}>
                                <Text dimColor>No items.</Text>
                            </Box>
                        )}
                    </Box>
                )}
                <Box flexGrow={1}/>
            </Box>

            {/* Bottom tag and status on opposite sides */}
            <Box justifyContent="space-between">
                <Box><Text dimColor>{tag}</Text></Box>
                <Box>
                    <Text dimColor>
                        {status} • {visibleItems.length ? `${selectedIdx + 1}/${visibleItems.length}` : '0/0'}
                    </Text>
                </Box>
            </Box>

            {/* :login popup */}
            {showLogin && (
                <Box borderStyle="round" borderColor="yellow" paddingX={2} paddingY={1} flexDirection="column">
                    <Text bold>Logging in…</Text>
                    <Box marginTop={1}><Text dimColor>{loginLog || 'Waiting…'}</Text></Box>
                    <Box marginTop={1}><Text dimColor>Close when complete.</Text></Box>
                </Box>
            )}
        </Box>
    );
};

render(<App/>);
