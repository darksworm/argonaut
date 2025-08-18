import React, {useEffect, useMemo, useState} from 'react';
import {Box, Text, useApp, useInput} from 'ink';
import TextInput from 'ink-text-input';
import chalk from 'chalk';
import stringWidth from 'string-width';
import {runAppDiffSession} from './DiffView';
import {runLicenseSession} from './LicenseView';
import ArgoNautBanner from "./Banner";
import packageJson from '../../package.json';
import type {AppItem, Mode, View} from '../types/domain';
import OfficeSupplyManager, {rulerLineMode} from './OfficeSupplyManager';
import {getCurrentServerUrl, readCLIConfig} from '../config/cli-config';
import {hostFromUrl} from '../config/paths';
import {tokenFromConfig} from '../auth/token';
import {getApiVersion as getApiVersionApi} from '../api/version';
import {getUserInfo} from '../api/session';
import {syncApp} from '../api/applications.command';
import {useApps} from '../hooks/useApps';
import Rollback from './Rollback';
import AuthRequiredView from './AuthRequiredView';
import Help from './Help';
import {ResourceStream} from './ResourceStream';
import ConfirmationBox from './ConfirmationBox';
import {checkVersion} from '../utils/version-check';
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
    const [baseUrl, setBaseUrl] = useState<string | null>(null);
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
    const [isVersionOutdated, setIsVersionOutdated] = useState<boolean>(false);
    const [latestVersion, setLatestVersion] = useState<string | undefined>(undefined);

    // Scopes / selections
    const [scopeClusters, setScopeClusters] = useState<Set<string>>(new Set());
    const [scopeNamespaces, setScopeNamespaces] = useState<Set<string>>(new Set());
    const [scopeProjects, setScopeProjects] = useState<Set<string>>(new Set());
    const [selectedApps, setSelectedApps] = useState<Set<string>>(new Set());
    const [confirmTarget, setConfirmTarget] = useState<string | null>(null);

    // Rollback overlay controller (app name to open)
    const [rollbackAppName, setRollbackAppName] = useState<string | null>(null);
    // Single-app sync view (resource stream)
    const [syncViewApp, setResourcesApp] = useState<string | null>(null);
    
    // Vim-style navigation state for gg
    const [lastGPressed, setLastGPressed] = useState<number>(0);

    // Boot & auth
    useEffect(() => {
        (async () => {
            setMode('loading');
            setStatus('Loading ArgoCD config…');
            const cfg = await readCLIConfig();

            const currentUrl = getCurrentServerUrl(cfg);
            if (!currentUrl) {
                // If config can't be loaded or has no current context, require authentication
                setToken(null);
                setStatus('No ArgoCD context configured. Please run `argocd login` to configure and authenticate.');
                setMode('auth-required');
                return;
            }
            setBaseUrl(currentUrl);

            try {
                const tokMaybe = await tokenFromConfig();
                if (!tokMaybe) throw new Error('No token in config');
                // Verify token by calling userinfo
                await getUserInfo(currentUrl, tokMaybe);
                const version = await getApiVersionApi(currentUrl, tokMaybe);
                setApiVersion(version);
                setToken(tokMaybe);
                setStatus('Ready');
                
                // Check for version updates
                checkVersion(packageJson.version).then(result => {
                    setIsVersionOutdated(result.isOutdated);
                    if (result.latestVersion) {
                        setLatestVersion(result.latestVersion);
                    }
                    if (result.error && !result.latestVersion) {
                        setStatus(prevStatus => prevStatus === 'Ready' ? 'Ready • Could not check for updates' : prevStatus);
                    }
                }).catch(() => {
                    // Silently ignore version check errors
                });
            } catch {
                setToken(null);
                setStatus('please use argocd login to authenticate before running argonaut');
                setMode('auth-required');
                return;
            }

            setMode('normal');
        })().catch(e => {
            setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`);
            setMode('normal');
        });
    }, []);

    // Live data via useApps hook
    const {apps: liveApps, status: appsStatus} = useApps(baseUrl, token, mode === 'external', (err) => {
        // On auth error from background data flow, clear token and show auth-required message
        setToken(null);
        setStatus('please use argocd login to authenticate before running argonaut');
        setMode('auth-required');
    });

    useEffect(() => {
        if (!baseUrl || !token) return;
        if (mode === 'external' || mode === 'auth-required') return; // pause syncing state while in external/diff mode or auth required
        setApps(liveApps);
        setStatus(appsStatus);
    }, [baseUrl, token, liveApps, appsStatus, mode]);

    useEffect(() => {
        if(!baseUrl || !token) return;
        getApiVersionApi(baseUrl, token).then(setApiVersion)
    }, [baseUrl, token]);

    const [confirmSyncPrune, setConfirmSyncPrune] = useState(false);
    const [confirmSyncWatch, setConfirmSyncWatch] = useState(true);

    // Input
    useInput((input, key) => {
        if (mode === 'external') return;
        if (mode === 'resources') return; // handled by ResourceStream
        if (mode === 'auth-required') {
            if (input.toLowerCase() === 'q') {
                exit();
                return;
            }
            // All other input ignored in auth-required mode
            return;
        }
        if (mode === 'help') {
            if (input === '?' || key.escape) setMode('normal');
            return;
        }
        if (mode === 'rulerline') {
            return;
        }
        if (mode === 'search') {
            if (key.escape) {
                setMode('normal');
                setSearchQuery('');
                return;
            }
            // Allow navigating the filtered list while typing
            if (key.downArrow) {
                setSelectedIdx(s => Math.min(s + 1, Math.max(0, visibleItems.length - 1)));
                return;
            }
            if (key.upArrow) {
                setSelectedIdx(s => Math.max(s - 1, 0));
                return;
            }
            // Enter is handled by TextInput onSubmit; other typing goes to TextInput
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
            // Toggle watch (only when not multi)
            if (input.toLowerCase() === 'w') {
                if (confirmTarget !== '__MULTI__') {
                    setConfirmSyncWatch(v => !v);
                }
                return;
            }
            // All other handling is done via the ConfirmationBox component
            return;
        }
        if (mode === 'rollback') {
            // While rollback overlay is active, let the component handle all input
            return;
        }

        // normal
        if (input.toLowerCase() === 'q') {
            exit();
            return;
        }
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
        
        // Vim-style navigation: gg to go to top, G to go to bottom
        if (input === 'g') {
            const now = Date.now();
            if (now - lastGPressed < 500) { // 500ms window for double g
                setSelectedIdx(0); // Go to top
            }
            setLastGPressed(now);
            return;
        }
        if (input === 'G') {
            setSelectedIdx(Math.max(0, visibleItems.length - 1)); // Go to bottom
            return;
        }

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
        const emptyStringSet = new Set<string>();
        switch (view) {
            case 'clusters':
                setScopeNamespaces(emptyStringSet);
            case 'namespaces':
                setScopeProjects(emptyStringSet);
            case 'projects':
                setSelectedApps(emptyStringSet);
        }
    }

    function toggleSelection() {
        const item = visibleItems[selectedIdx];
        if (item == null) return;
        
        const val = String(item);
        clearLowerLevelSelections(view);
        
        if (view === 'clusters') {
            const next = scopeClusters.has(val) ? new Set<string>() : new Set([val]);
            setScopeClusters(next);
            // When a cluster is selected, verify token via userinfo
            if (baseUrl) {
                (async () => {
                    try {
                        if (!token) throw new Error('No token');
                        await getUserInfo(baseUrl, token);
                    } catch {
                        setToken(null);
                        setStatus('please use argocd login to authenticate before running argonaut');
                        setMode('auth-required');
                    }
                })();
            }
        } else if (view === 'namespaces') {
            const next = scopeNamespaces.has(val) ? new Set<string>() : new Set([val]);
            setScopeNamespaces(next);
        } else if (view === 'projects') {
            const next = scopeProjects.has(val) ? new Set<string>() : new Set([val]);
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
        if (is('license', 'licenses')) {
            try {
                setMode('normal');
                setStatus('Opening licenses…');

                await runLicenseSession({
                    forwardInput: true,
                    onEnterExternal: () => setMode('external'),
                    onExitExternal: () => {
                    },
                });
                setMode('normal');
                setStatus('License viewer closed.');
            } catch (e: any) {
                try {
                    const stdinAny = process.stdin as any;
                    stdinAny.setRawMode?.(true);
                    stdinAny.resume?.();
                } catch {
                }
                setMode('normal');
                setStatus(`License viewer failed: ${e?.message || String(e)}`);
            }
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
            setStatus('please use argocd login to authenticate before running argonaut');
            setMode('auth-required');
            return;
        }

        if (is('resources', 'resource', 'res')) {
            const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || (selectedApps.size === 1 ? Array.from(selectedApps)[0] : undefined);
            if (!target) {
                setStatus('No app selected to open resources view.');
                return;
            }
            if (!baseUrl || !token) {
                setStatus('Not authenticated.');
                return;
            }
            setResourcesApp(target);
            setMode('resources');
            return;
        }

        if (is('diff')) {
            const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || Array.from(selectedApps)[0];
            if (!target) {
                setStatus('No app selected to diff.');
                return;
            }
            if (!baseUrl || !token) {
                setStatus('Not authenticated.');
                return;
            }

            try {
                setMode('normal');
                setStatus(`Preparing diff for ${target}…`);

                const opened = await runAppDiffSession(baseUrl, token, target, {
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
            if (!baseUrl || !token) {
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
                setConfirmSyncPrune(false);
                setConfirmSyncWatch(true);
                setMode('confirm-sync');
                return;
            }
            if (selectedApps.size > 1) {
                setConfirmTarget(`__MULTI__`);
                setConfirmSyncPrune(false);
                setConfirmSyncWatch(false); // disabled for multi
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
            setConfirmSyncPrune(false);
            setConfirmSyncWatch(true);
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

        if (cmd === rulerLineMode) {
            setMode('rulerline');
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
        if (!baseUrl || !token) {
            setStatus('Not authenticated.');
            return;
        }
        try {
            setStatus(`Syncing ${isMulti ? `${names.length} app(s)` : names[0]}…`);
            for (const n of names) {
                const app = apps.find(a => a.name === n);
                syncApp(baseUrl, token, n, { prune: confirmSyncPrune, appNamespace: app?.appNamespace });
            }
            setStatus(`Sync initiated for ${isMulti ? `${names.length} app(s)` : names[0]}.`);
            // Show resource stream only for single-app syncs and when watch is enabled
            if (!isMulti && confirmSyncWatch) {
                setResourcesApp(names[0]);
                setMode('resources');
            } else {
                // After syncing multiple apps, clear the selection
                if (isMulti) setSelectedApps(new Set());
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
        const loadingHeader: string = `${chalk.bold('View:')} ${chalk.yellow('LOADING')} • ${chalk.bold('Context:')} ${chalk.cyan(hostFromUrl(baseUrl) || '—')}`;
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

    // Office supply management full-screen view
    if (mode === 'rulerline') {
        return <OfficeSupplyManager onExit={() => setMode('normal')} />;
    }

    // Authentication required full-screen view
    if (mode === 'auth-required') {
        return (
            <AuthRequiredView
                server={baseUrl ? hostFromUrl(baseUrl) : null}
                apiVersion={apiVersion}
                termCols={termCols}
                termRows={termRows}
                clusterScope={fmtScope(scopeClusters)}
                namespaceScope={fmtScope(scopeNamespaces)}
                projectScope={fmtScope(scopeProjects)}
                argonautVersion={packageJson.version}
                message={status}
            />
        );
    }

    // Full-screen rollback overlay: occupy whole screen and hide the apps UI, but keep header
    if (mode === 'rollback' && rollbackAppName) {
        return (
            <Box flexDirection="column" paddingX={1} height={termRows - 1}>
                <ArgoNautBanner
                    server={baseUrl ? hostFromUrl(baseUrl) : null}
                    clusterScope={fmtScope(scopeClusters)}
                    namespaceScope={fmtScope(scopeNamespaces)}
                    projectScope={fmtScope(scopeProjects)}
                    termCols={termCols}
                    termRows={availableRows}
                    apiVersion={apiVersion}
                    argonautVersion={packageJson.version}
                />
                <Rollback
                    app={rollbackAppName}
                    baseUrl={baseUrl}
                    token={token}
                    appNamespace={apps.find(a => a.name === rollbackAppName)?.appNamespace}
                    onClose={() => {
                        setMode('normal');
                        setRollbackAppName(null);
                    }}
                    onStartWatching={(appName) => {
                        setResourcesApp(appName);
                        setMode('resources');
                        setRollbackAppName(null);
                    }}
                />
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
                server={baseUrl ? hostFromUrl(baseUrl) : null}
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
                <ConfirmationBox
                    title={confirmTarget === '__MULTI__' ? 'Sync applications?' : 'Sync application?'}
                    message={confirmTarget === '__MULTI__' ? 'Do you want to sync' : 'Do you want to sync'}
                    target={confirmTarget === '__MULTI__' ? String(selectedApps.size) : confirmTarget!}
                    isMulti={confirmTarget === '__MULTI__'}
                    options={[
                        {
                            key: 'p',
                            label: 'Prune',
                            value: confirmSyncPrune
                        },
                        { 
                            key: 'w', 
                            label: 'Watch', 
                            value: confirmSyncWatch, 
                            disabled: confirmTarget === '__MULTI__'
                        }
                    ]}
                    onConfirm={confirmSync}
                />
            )}

            {/* Content area (fills space) */}
            <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}
                 flexWrap="nowrap">
                {mode === 'help' ? (
                    <Box flexDirection="column" marginTop={1} flexGrow={1}><Help version={packageJson.version} isOutdated={isVersionOutdated} latestVersion={latestVersion}/></Box>
                ) : mode === 'resources' && baseUrl && token && syncViewApp ? (
                    <Box flexDirection="column" flexGrow={1}>
                        <ResourceStream baseUrl={baseUrl} token={token} appName={syncViewApp}
                                        appNamespace={apps.find(a => a.name === syncViewApp)?.appNamespace}
                                        onExit={() => { setMode('normal'); setResourcesApp(null); }}/>
                    </Box>
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
                        {isVersionOutdated && <Text color="yellow"> • Update available!</Text>}
                    </Text>
                </Box>
            </Box>

        </Box>
    );
};
