// @ts-nocheck
import React, {useEffect, useMemo, useState} from 'react';
import {Box, Text, useApp, useInput} from 'ink';
import TextInput from 'ink-text-input';
import chalk from 'chalk';
import stringWidth from 'string-width';
import {execa} from 'execa'; // still used for diff sessions
import {runAppDiffSession} from './DiffView';
import ArgoNautBanner from "./banner";
import packageJson from '../../package.json';
import type {AppItem, Mode, View} from '../types/domain';
import type {ArgoCLIConfig} from '../config/cli-config';
import {getCurrentServer, readCLIConfig} from '../config/cli-config';
import {tokenFromConfig} from '../auth/token';
import {getApiVersion as getApiVersionApi} from '../api/version';
import {getUserInfo} from '../api/session';
import {syncApp} from '../api/applications.command';
import {useApps} from '../hooks/useApps';
import Rollback from './Rollback';
import Help from './Help';
import {colorFor, fmtScope,  uniqueSorted} from "../utils";

const COL = {
    mark: 2,
    name: 36,
    sync: 4,
    health: 6,
} as const;

export const App: React.FC = () => {
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

    // Scopes / selections
    const [scopeClusters, setScopeClusters] = useState<Set<string>>(new Set());
    const [scopeNamespaces, setScopeNamespaces] = useState<Set<string>>(new Set());
    const [scopeProjects, setScopeProjects] = useState<Set<string>>(new Set());
    const [selectedApps, setSelectedApps] = useState<Set<string>>(new Set());
    const [confirmTarget, setConfirmTarget] = useState<string | null>(null);

    // Rollback overlay controller (app name to open)
    const [rollbackAppName, setRollbackAppName] = useState<string | null>(null);

    // Boot & auth
    useEffect(() => {
        (async () => {
            setMode('loading');
            setStatus('Loading ArgoCD config…');
            const cfg = await readCLIConfig();

            const currentSrv = getCurrentServer(cfg);
            if (!currentSrv) {
                setStatus('No current context configured in ArgoCD config. Please set `current-context` and login using argocd.');
                setMode('normal');
                return;
            }
            setServer(currentSrv);

            try {
                const tokMaybe = await tokenFromConfig();
                if (!tokMaybe) throw new Error('No token in config');
                // Verify token by calling userinfo
                await getUserInfo(currentSrv, tokMaybe);
                const version = await getApiVersionApi(currentSrv, tokMaybe);
                setApiVersion(version);
                setToken(tokMaybe);
                setStatus('Ready');
            } catch {
                // Token invalid or missing → prompt for login command
                setStatus('Authentication required. Provide a login command.');
                setToken(null);
                setStatus('Authentication required: please use argocd login to authenticate before running argonaut');
            }

            setMode('normal');
        })().catch(e => {
            setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`);
            setMode('normal');
        });
    }, []);

    // Live data via useApps hook
    const {apps: liveApps, status: appsStatus} = useApps(server, token, mode === 'external', (err) => {
        // On auth error from background data flow, clear token and show auth-required message
        setToken(null);
        setStatus('Authentication required: please use argocd login to authenticate before running argonaut');
    });

    useEffect(() => {
        if (!server || !token) return;
        if (mode === 'external') return; // pause syncing state while in external/diff mode
        setApps(liveApps);
        setStatus(appsStatus);
    }, [server, token, liveApps, appsStatus, mode]);

    useEffect(() => {
        if(!server || !token) return;
        getApiVersionApi(server, token).then(setApiVersion)
    }, [server, token]);

    const [confirmInput, setConfirmInput] = useState('');
    const [confirmSyncPrune, setConfirmSyncPrune] = useState(false);

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
            // Toggle prune
            if (input.toLowerCase() === 'p') {
                setConfirmSyncPrune(v => !v);
                return;
            }
            // All other handling is done via TextInput onSubmit in the confirm dialog
            return;
        }
        if (mode === 'rollback') {
            // While rollback overlay is active, let the component handle all input
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

    function clearLowerLevelSelections(view: View) {
        const emptySet = new Set();
        switch (view) {
            case 'clusters':
                setScopeNamespaces(emptySet);
            case 'namespaces':
                setScopeProjects(emptySet);
            case 'projects':
                setSelectedApps(emptySet);
        }
    }

    function toggleSelection() {
        const item = visibleItems[selectedIdx];
        if (item == null) return;
        
        const val = String(item);
        clearLowerLevelSelections(view);
        
        if (view === 'clusters') {
            const next = scopeClusters.has(val) ? new Set() : new Set([val]);
            setScopeClusters(next);
            // When a cluster is selected, verify token via userinfo
            if (server) {
                (async () => {
                    try {
                        if (!token) throw new Error('No token');
                        await getUserInfo(server, token);
                    } catch {
                        setToken(null);
                        setStatus('Authentication required: please use argocd login to authenticate before running argonaut');
                    }
                })();
            }
        } else if (view === 'namespaces') {
            const next = scopeNamespaces.has(val) ? new Set() : new Set([val]);
            setScopeNamespaces(next);
        } else if (view === 'projects') {
            const next = scopeProjects.has(val) ? new Set() : new Set([val]);
            setScopeProjects(next);
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

        setSelectedIdx(0);
        setActiveFilter('');
        setSearchQuery('');
        clearLowerLevelSelections(view);

        const val = String(item);
        const next = new Set([val]);

        switch (view) {
            case 'clusters':
                setScopeClusters(next);
                setView('namespaces');
                return;
            case 'namespaces':
                setScopeNamespaces(next);
                setView('projects');
                return;
            case 'projects':
                setScopeProjects(next);
                setView('apps');
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


        if (is('login')) {
            setStatus('Authentication required: please use argocd login to authenticate before running argonaut');
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

                const opened = await runAppDiffSession(server, token, target, {
                    forwardInput: true,
                    onEnterExternal: () => setMode('external'),
                    onExitExternal: () => {
                    },
                });
                setMode('normal');
                setStatus(opened ? `Diff closed for ${target}.` : 'No differences.');
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
            if (!target) {
                setStatus('No app selected to rollback.');
                return;
            }
            if (!server || !token) {
                setStatus('Not authenticated.');
                return;
            }
            await openRollbackFlow(target);
            return;
        }

        if (is('sync')) {
            // Prefer explicit arg; otherwise if multiple apps are selected, confirm multi-sync.
            if (arg) {
                setConfirmTarget(arg);
                setConfirmInput('');
                setConfirmSyncPrune(false);
                setMode('confirm-sync');
                return;
            }
            if (selectedApps.size > 1) {
                setConfirmTarget(`__MULTI__`);
                setConfirmInput('');
                setConfirmSyncPrune(false);
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
            setConfirmSyncPrune(false);
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
            for (const n of names) syncApp(server, token, n, { prune: confirmSyncPrune });
            setStatus(`Sync initiated for ${isMulti ? `${names.length} app(s)` : names[0]}.`);
            // After syncing multiple apps, clear the selection
            if (isMulti) {
                setSelectedApps(new Set());
            }
        } catch (e: any) {
            setStatus(`Sync failed: ${e.message}`);
        }
    }

    async function openRollbackFlow(appName: string) {
        setStatus(`Opening rollback for ${appName}…`);
        setRollbackAppName(appName);
        setMode('rollback');
    }

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

    // Loading screen fills the viewport
    if (mode === 'loading') {
        const spinChar = '⠋';
        // @ts-ignore
        const loadingHeader: string = `${chalk.bold('View:')} ${chalk.yellow('LOADING')} • ${chalk.bold('Context:')} ${chalk.cyan(server || '—')}`;
        return (
            <Box flexDirection="column" borderStyle="round" borderColor="magenta" paddingX={1} height={termRows - 1}>
                <Box><Text>{loadingHeader}</Text></Box>
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

    // Full-screen rollback overlay: occupy whole screen and hide the apps UI, but keep header
    if (mode === 'rollback' && rollbackAppName) {
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
                <Box flexDirection="column" marginTop={1} flexGrow={1} borderStyle="round" borderColor="magenta"
                     paddingX={1} flexWrap="nowrap">
                    <Box flexDirection="column" marginTop={1} flexGrow={1}>
                        <Rollback
                            app={rollbackAppName}
                            server={server}
                            token={token}
                            onClose={() => {
                                setMode('normal');
                                setRollbackAppName(null);
                            }}
                        />
                    </Box>
                    <Box flexGrow={1}/>
                </Box>
            </Box>
        );
    }

    const GUTTER = 1;
    const MIN_NAME = 12;
    const Sep = () => <Box width={GUTTER}/>;

    // Single-cell icons (text variant) and ASCII fallback
    const ASCII_ICONS = {
        check: 'V',
        warn: '!',
        quest: '?',
        delta: '^',
    } as const;

    const SYNC_LABEL: Record<string, string> = {
        Synced: 'Synced',
        OutOfSync: 'OutOfSync',
        Unknown: 'Unknown',
        Degraded: 'Degraded'
    };
    const HEALTH_LABEL: Record<string, string> = {
        Healthy: 'Healthy',
        Missing: 'Missing',
        Degraded: 'Degraded',
        Progressing: 'Progressing',
        Unknown: 'Unknown'
    };

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

    const canShowLabels = (cols: number) =>
        cols >= MIN_NAME + fixedNoLastWide + (2 * GUTTER) + overhead + 15;

    // Effective column widths depending on whether labels are shown
    const SYNC_COL = canShowLabels(termCols) ? SYNC_WIDE : COL.sync;
    const HEALTH_COL = canShowLabels(termCols) ? HEALTH_WIDE : COL.health;

    const SYNC_ICON_ASCII: Record<string, string> = {
        Synced: ASCII_ICONS.check,
        OutOfSync: ASCII_ICONS.delta,
        Unknown: ASCII_ICONS.quest,
        Degraded: ASCII_ICONS.warn,
    };

    const HEALTH_ICON_ASCII: Record<string, string> = {
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
                        onSubmit={(val) => {
                            setMode('normal');
                            runCommand(val);
                            setCommand(':');
                        }}
                    />
                    <Box width={2}/>
                    <Text dimColor>(Enter to run, Esc to cancel)</Text>
                </Box>
            )}


            {/* Confirm sync popup */}
            {mode === 'confirm-sync' && (
                <Box borderStyle="round" borderColor="yellow" paddingX={2} paddingY={1} flexDirection="column">
                    {confirmTarget === '__MULTI__' ? (
                        <>
                            <Text bold>Sync applications?</Text>
                            <Box>
                                <Text>Do you want to sync <Text color="magentaBright"
                                                                bold>({selectedApps.size})</Text> applications? (y/n):
                                </Text>
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
                            <Box marginTop={1}>
                                <Text>
                                    Prune [p]: <Text color={(confirmSyncPrune ? 'yellow' : undefined) as any}>{confirmSyncPrune ? 'on' : 'off'}</Text>
                                </Text>
                            </Box>
                        </>
                    ) : (
                        <>
                            <Text bold>Sync application?</Text>
                            <Box marginTop={1}>
                                <Text>Do you want to sync <Text color="magentaBright" bold>{confirmTarget}</Text>?
                                    (y/n): </Text>
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
                            <Box marginTop={1}>
                                <Text>
                                    Prune [p]: <Text color={(confirmSyncPrune ? 'yellow' : undefined) as any}>{confirmSyncPrune ? 'on' : 'off'}</Text>
                                </Text>
                            </Box>
                        </>
                    )}
                </Box>
            )}

            {/* Content area (fills space) */}
            <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}
                 flexWrap="nowrap">
                {mode === 'help' ? (
                    <Box flexDirection="column" marginTop={1} flexGrow={1}><Help version={packageJson.version}/></Box>
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
                        {rowsSlice.map((it: any, i: number) => {
                            const actualIndex = start + i;
                            const isCursor = actualIndex === selectedIdx;

                            if (view === 'apps') {
                                const a = it as AppItem;
                                const isChecked = selectedApps.has(a.name);
                                const active = isCursor || isChecked;

                                return (
                                    <Box key={a.name} width="100%"
                                         backgroundColor={active ? 'magentaBright' : undefined} flexWrap="nowrap"
                                         justifyContent={"flex-start"}>
                                        {/* NAME (flex) */}
                                        <Box flexGrow={1} flexShrink={1} minWidth={0}>
                                            <Text wrap="truncate-end">{a.name}</Text>
                                        </Box>

                                        {/* SYNC (fixed, right-aligned) */}
                                        <Sep/>
                                        <Box width={SYNC_COL} flexShrink={0} justifyContent="flex-end">
                                            <Text color={(colorFor(a.sync).color as any)} dimColor={colorFor(a.sync).dimColor as any}>
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
                                            <Text color={(colorFor(a.health).color as any)} dimColor={colorFor(a.health).dimColor as any}>
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
                                (view === 'clusters' && scopeClusters.has(label)) ||
                                (view === 'namespaces' && scopeNamespaces.has(label)) ||
                                (view === 'projects' && scopeProjects.has(label));
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

        </Box>
    );
};
