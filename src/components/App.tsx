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
import {getCurrentServer, readCLIConfig} from '../config/cli-config';
import {readArgonautConfig, detectNewServers} from '../config/argonaut-config';
import {tokenFromConfig} from '../auth/token';
import {getApiVersion as getApiVersionApi} from '../api/version';
import {getUserInfo, login} from '../api/session';
import {syncApp} from '../api/applications.command';
import {useApps} from '../hooks/useApps';
import Rollback from './Rollback';
import ImportView from './ImportView';
import ConfigView from './ConfigView';
import LoadingView from './LoadingView';
import PasswordPrompt from './PasswordPrompt';
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
    
    // Config mode state
    const [configMode, setConfigMode] = useState<'config' | null>(null);
    
    // Server selection state
    const [availableServers, setAvailableServers] = useState<any[]>([]);
    const [selectedServerIndex, setSelectedServerIndex] = useState(0);
    const [showServerSelection, setShowServerSelection] = useState(false);
    
    // Password prompt state
    const [showPasswordPrompt, setShowPasswordPrompt] = useState(false);
    const [currentServerForAuth, setCurrentServerForAuth] = useState<any>(null);

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
            setStatus('Initializing…');
            
            console.log('[DEBUG] Starting app initialization...');
            
            // Check if we have an Argonaut config
            const argonautConfig = await readArgonautConfig();
            console.log('[DEBUG] Argonaut config:', argonautConfig ? 'exists' : 'not found');
            
            if (!argonautConfig) {
                // No Argonaut config exists - check if ArgoCD config exists for import
                setStatus('Checking for ArgoCD configuration…');
                const argoConfig = await readCLIConfig();
                console.log('[DEBUG] ArgoCD config:', argoConfig ? 'exists' : 'not found');
                
                if (!argoConfig) {
                    // No ArgoCD config either - show import view with instructions
                    setStatus('No ArgoCD configuration found. Please run "argocd login" first.');
                    setMode('auth-required');
                    return;
                }
                
                // ArgoCD config exists - show import flow
                console.log('[DEBUG] Showing import flow...');
                setStatus('ArgoCD configuration detected. Starting import…');
                setMode('auth-required');
                return;
            }
            
            // We have Argonaut config - check for new servers to import
            setStatus('Checking for new ArgoCD servers…');
            const newServers = await detectNewServers();
            const hasNewServers = newServers.some(s => s.isNew);
            console.log('[DEBUG] New servers check:', hasNewServers ? 'found new servers' : 'no new servers');
            
            if (hasNewServers) {
                setStatus(`Found ${newServers.filter(s => s.isNew).length} new server(s) to import.`);
                setMode('auth-required');
                return;
            }
            
            // Continue with normal auth flow using Argonaut config
            setStatus('Loading Argonaut server config…');
            const importedServers = argonautConfig.servers.filter(s => s.imported);
            console.log('[DEBUG] Imported servers:', importedServers.length);
            
            if (importedServers.length === 0) {
                setToken(null);
                setStatus('No imported servers found. Please import servers from ArgoCD config.');
                setMode('auth-required');
                return;
            }
            
            // If multiple servers, show selection screen
            if (importedServers.length > 1) {
                console.log('[DEBUG] Multiple servers found, showing selection...');
                setAvailableServers(importedServers);
                setSelectedServerIndex(0);
                setShowServerSelection(true);
                setStatus('Multiple servers available. Select one to connect.');
                setMode('normal');
                return;
            }
            
            // Single server - proceed with authentication
            const serverConfig = importedServers[0];
            setServer(serverConfig.serverUrl);
            console.log('[DEBUG] Trying to authenticate with server:', serverConfig.serverUrl);

            try {
                console.log('[DEBUG] Getting token from config...');
                const tokMaybe = await tokenFromConfig();
                if (!tokMaybe) throw new Error('No token in config');
                console.log('[DEBUG] Got token, verifying with userinfo...');
                // Verify token by calling userinfo
                await getUserInfo(serverConfig.serverUrl, tokMaybe);
                console.log('[DEBUG] User info verified, getting API version...');
                const version = await getApiVersionApi(serverConfig.serverUrl, tokMaybe);
                setApiVersion(version);
                setToken(tokMaybe);
                setStatus('Ready');
                console.log('[DEBUG] Authentication successful!');
                
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
            } catch (e) {
                console.log('[DEBUG] Authentication failed:', e);
                setToken(null);
                setStatus('Authentication expired. Please run "argocd login" or configure servers in settings.');
                setMode('auth-required');
                return;
            }

            setMode('normal');
        })().catch(e => {
            console.log('[DEBUG] Boot error:', e);
            setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`);
            setMode('normal');
        });
    }, []);

    // Live data via useApps hook
    const {apps: liveApps, status: appsStatus} = useApps(server, token, mode === 'external', (err) => {
        // On auth error from background data flow, clear token and show auth-required message
        setToken(null);
        setStatus('Authentication required. Please configure your login settings.');
        setMode('auth-required');
    });

    useEffect(() => {
        if (!server || !token) return;
        if (mode === 'external' || mode === 'auth-required') return; // pause syncing state while in external/diff mode or auth required
        setApps(liveApps);
        setStatus(appsStatus);
    }, [server, token, liveApps, appsStatus, mode]);

    useEffect(() => {
        if(!server || !token) return;
        getApiVersionApi(server, token).then(setApiVersion)
    }, [server, token]);

    const [confirmSyncPrune, setConfirmSyncPrune] = useState(false);
    const [confirmSyncWatch, setConfirmSyncWatch] = useState(true);

    // Input
    useInput((input, key) => {
        if (mode === 'external') return;
        if (mode === 'auth-required') return;
        if (mode === 'resources') return; // handled by ResourceStream
        if (configMode === 'config') return; // handled by ConfigView
        if (showPasswordPrompt) return; // handled by PasswordPrompt
        if (mode === 'loading') {
            if (input.toLowerCase() === 'q') {
                exit();
                return;
            }
            // All other input ignored in loading mode
            return;
        }
        
        // Server selection mode
        if (showServerSelection) {
            if (input.toLowerCase() === 'q') {
                exit();
                return;
            }
            if (input === 'j' || key.downArrow) {
                setSelectedServerIndex(prev => Math.min(prev + 1, availableServers.length - 1));
                return;
            }
            if (input === 'k' || key.upArrow) {
                setSelectedServerIndex(prev => Math.max(prev - 1, 0));
                return;
            }
            if (key.return) {
                connectToSelectedServer();
                return;
            }
            if (input === ':') {
                setMode('command');
                setCommand(':');
                return;
            }
            return;
        }
        if (mode === 'help') {
            if (input === '?' || key.escape) setMode('normal');
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
            if (server) {
                (async () => {
                    try {
                        if (!token) throw new Error('No token');
                        await getUserInfo(server, token);
                    } catch {
                        setToken(null);
                        setStatus('Authentication required. Please configure your login settings.');
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


        if (is('config', 'settings')) {
            setConfigMode('config');
            return;
        }

        if (is('login')) {
            setStatus('Authentication required. Please configure your login settings.');
            setMode('auth-required');
            return;
        }

        if (is('resources', 'resource', 'res')) {
            const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || (selectedApps.size === 1 ? Array.from(selectedApps)[0] : undefined);
            if (!target) {
                setStatus('No app selected to open resources view.');
                return;
            }
            if (!server || !token) {
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
            for (const n of names) {
                const app = apps.find(a => a.name === n);
                syncApp(server, token, n, { prune: confirmSyncPrune, appNamespace: app?.appNamespace });
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

    async function connectToSelectedServer() {
        const serverConfig = availableServers[selectedServerIndex];
        if (!serverConfig) return;
        
        setShowServerSelection(false);
        setMode('loading');
        setServer(serverConfig.serverUrl);
        setStatus('Connecting to selected server…');
        
        try {
            console.log('[DEBUG] Connecting to selected server:', serverConfig.serverUrl);
            const tokMaybe = await tokenFromConfig();
            if (!tokMaybe) throw new Error('No token in config');
            
            // Verify token by calling userinfo
            await getUserInfo(serverConfig.serverUrl, tokMaybe);
            const version = await getApiVersionApi(serverConfig.serverUrl, tokMaybe);
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
            
            setMode('normal');
        } catch (e) {
            console.log('[DEBUG] Authentication failed:', e);
            setToken(null);
            
            // If authentication failed, check if we need password input
            if (serverConfig.sso || serverConfig.core) {
                setStatus('Please authenticate using the ArgoCD CLI first.');
                setShowServerSelection(true);
                setMode('normal');
            } else {
                // Show password prompt for username/password auth
                setCurrentServerForAuth(serverConfig);
                setShowPasswordPrompt(true);
                setShowServerSelection(false);
                setMode('normal');
            }
        }
    }

    async function handlePasswordSubmit(credentials: { username?: string; password: string }) {
        if (!currentServerForAuth) return;
        
        setShowPasswordPrompt(false);
        setMode('loading');
        setStatus('Authenticating...');
        
        try {
            const username = credentials.username || currentServerForAuth.username;
            if (!username) {
                throw new Error('Username is required');
            }
            
            console.log('[DEBUG] Attempting login with username/password...');
            const loginResult = await login(currentServerForAuth.serverUrl, username, credentials.password);
            
            // Verify the token works
            await getUserInfo(currentServerForAuth.serverUrl, loginResult.token);
            const version = await getApiVersionApi(currentServerForAuth.serverUrl, loginResult.token);
            
            setServer(currentServerForAuth.serverUrl);
            setToken(loginResult.token);
            setApiVersion(version);
            setStatus('Ready');
            setCurrentServerForAuth(null);
            
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
            
            setMode('normal');
        } catch (e) {
            console.log('[DEBUG] Password authentication failed:', e);
            setToken(null);
            setStatus(`Authentication failed: ${e instanceof Error ? e.message : String(e)}`);
            setShowPasswordPrompt(true);
            setMode('normal');
        }
    }

    function handlePasswordCancel() {
        setShowPasswordPrompt(false);
        setCurrentServerForAuth(null);
        setShowServerSelection(true);
        setStatus('Authentication cancelled. Select another server.');
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
        return (
            <LoadingView
                termRows={termRows}
                message="Connecting…"
                server={server}
                showHeader={true}
                showAbort={true}
                onAbort={() => setConfigMode('config')}
            />
        );
    }

    // While in external diff mode, pause rendering the React app entirely
    if (mode === 'external') {
        return null;
    }

    // Configuration view
    if (configMode === 'config') {
        return (
            <ConfigView
                termRows={termRows}
                onClose={() => setConfigMode(null)}
            />
        );
    }

    // Password prompt view
    if (showPasswordPrompt && currentServerForAuth) {
        return (
            <PasswordPrompt
                termRows={termRows}
                serverConfig={currentServerForAuth}
                onSubmit={handlePasswordSubmit}
                onCancel={handlePasswordCancel}
            />
        );
    }

    // Import/authentication required full-screen view
    if (mode === 'auth-required') {
        return (
            <ImportView
                termRows={termRows}
                onComplete={() => {
                    // Force a re-initialization of the app after import
                    setMode('loading');
                    setStatus('Restarting…');
                    // Force re-initialization by clearing relevant state
                    setToken(null);
                    setServer(null);
                    setApps([]);
                    // The useEffect will re-run due to dependency changes
                }}
            />
        );
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
                <Rollback
                    app={rollbackAppName}
                    server={server}
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
                ) : mode === 'resources' && server && token && syncViewApp ? (
                    <Box flexDirection="column" flexGrow={1}>
                        <ResourceStream baseUrl={server} token={token} appName={syncViewApp}
                                        appNamespace={apps.find(a => a.name === syncViewApp)?.appNamespace}
                                        onExit={() => { setMode('normal'); setResourcesApp(null); }}/>
                    </Box>
                ) : showServerSelection ? (
                    <Box flexDirection="column" paddingX={1} paddingY={1}>
                        <Text bold color="cyan">📡 Select Server</Text>
                        <Box marginTop={1}>
                            <Text dimColor>Choose a server to connect to:</Text>
                        </Box>
                        
                        <Box marginTop={1} flexDirection="column">
                            {availableServers.map((server, index) => (
                                <Box 
                                    key={server.serverUrl} 
                                    backgroundColor={selectedServerIndex === index ? 'magentaBright' : undefined}
                                    paddingX={1}
                                    marginY={0}
                                >
                                    <Box flexGrow={1}>
                                        <Text>
                                            <Text color="cyan">{server.serverUrl}</Text>
                                            {server.contextName && <Text dimColor> ({server.contextName})</Text>}
                                        </Text>
                                    </Box>
                                    <Box paddingLeft={2}>
                                        <Text dimColor>
                                            {server.lastConnected ? 
                                                `Last: ${new Date(server.lastConnected).toLocaleDateString()}` : 
                                                'Never connected'
                                            }
                                        </Text>
                                    </Box>
                                </Box>
                            ))}
                        </Box>
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
                <Box>
                    <Text dimColor>
                        {showServerSelection ? (
                            'j/k navigate • Enter to connect • : for commands • q to quit'
                        ) : (
                            tag
                        )}
                    </Text>
                </Box>
                <Box>
                    <Text dimColor>
                        {showServerSelection ? (
                            `${selectedServerIndex + 1}/${availableServers.length} servers`
                        ) : (
                            <>
                                {status} • {visibleItems.length ? `${selectedIdx + 1}/${visibleItems.length}` : '0/0'}
                                {isVersionOutdated && <Text color="yellow"> • Update available!</Text>}
                            </>
                        )}
                    </Text>
                </Box>
            </Box>

        </Box>
    );
};
