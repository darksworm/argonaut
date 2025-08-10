import React, {useEffect, useMemo, useState} from 'react';
import {render, Box, Text, useApp, useInput} from 'ink';
import TextInput from 'ink-text-input';
import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import chalk from 'chalk';
import {execa} from 'execa';
import {spawn as ptySpawn, IPty} from 'node-pty';
import ArgoNautBanner from "./banner";
import packageJson from '../package.json';
import type {AppItem, View, Mode} from './types/domain';
import type {ArgoCLIConfig} from './config/cli-config';
import {readCLIConfig as readCLIConfigExt, writeCLIConfig as writeCLIConfigExt, getCurrentServer as getCurrentServerExt} from './config/cli-config';
import {ensureSSOLogin as ensureSSOLoginExt, tokenFromConfig as tokenFromConfigExt, ensureToken as ensureTokenExt} from './auth/token';
import {getApiVersion as getApiVersionApi} from './api/version';
import {syncApp} from './api/applications.command';
import {useApps} from './hooks/useApps';
import {getManagedResourceDiffs} from './api/applications.query';

// Switch to terminal alternate screen on start, and restore on exit
(function setupAlternateScreen() {
  if (typeof process === 'undefined') return;
  const out = process.stdout as any;
  const isTTY = !!out && typeof out.isTTY === 'boolean' ? out.isTTY : false;
  if (!isTTY) return;

  let cleaned = false;
  const enable = () => { try { out.write("\u001B[?1049h"); } catch {} };
  const disable = () => {
    if (cleaned) return; cleaned = true;
    try { out.write("\u001B[?1049l"); } catch {}
  };

  enable();

  process.on('exit', disable);
  process.on('SIGINT', () => { disable(); process.exit(130); });
  process.on('SIGTERM', () => { disable(); process.exit(143); });
  process.on('SIGHUP', () => { disable(); process.exit(129); });
  process.on('uncaughtException', (err) => { disable(); console.error(err); process.exit(1); });
  process.on('unhandledRejection', (reason) => { disable(); console.error(reason); process.exit(1); });
})();

// ------------------------------
// UI helpers
// ------------------------------

function colorFor(appState: string): {color?: any; dimColor?: boolean} {
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
  return Array.from(new Set(arr)).sort((a:any,b:any)=>`${a}`.localeCompare(`${b}`));
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
  sync: 12,
  health: 12,
  last: 10
} as const;

function RowBG({active, children}:{active:boolean; children:React.ReactNode}) {
  // Only handle text color when active/checked, background is handled at the row level
  return <Text color={active ? 'black' : undefined}>{children}</Text>;
}

// Utilities for diff command
function toYamlDoc(input?: string): string | null {
  if (!input) return null;
  try {
    const obj = JSON.parse(input);
    return YAML.stringify(obj, { lineWidth: 120 } as any);
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
    return () => { process.stdout.off('resize', onResize); };
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
    })().catch(e => { setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`); setMode('normal'); });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Live data via useApps hook
  const {apps: liveApps, status: appsStatus} = useApps(server, token);

  useEffect(() => {
    if (!server || !token) return;
    setApps(liveApps);
    setStatus(appsStatus);
  }, [server, token, liveApps, appsStatus]);

  // Periodic API version refresh (1 min)
  useEffect(() => {
    if (!server || !token) return;
    const id = setInterval(async () => {
      try {
        const v = await getApiVersionApi(server, token);
        setApiVersion(v);
      } catch {/* noop */}
    }, 60000);
    return () => clearInterval(id);
  }, [server, token]);

  // Input
  useInput((input, key) => {
    if (mode === 'external') return;
    if (mode === 'help') {
      if (input === '?' || key.escape) setMode('normal');
      return;
    }
    if (mode === 'search') {
      if (key.escape) { setMode('normal'); setSearchQuery(''); }
      // Enter handled by TextInput onSubmit
      return;
    }

    // In normal mode, Escape clears the active filter
    if (mode === 'normal' && key.escape && activeFilter) {
      setActiveFilter('');
      return;
    }
    if (mode === 'command') {
      if (key.escape) { setMode('normal'); setCommand(':'); }
      return; // TextInput handles typing/enter
    }
    if (mode === 'confirm-sync') {
      if (input.toLowerCase() === 'y' || key.return) { confirmSync(true); }
      if (input.toLowerCase() === 'n' || key.escape) { confirmSync(false); }
      return;
    }

    // normal
    if (input === '?') { setMode('help'); return; }
    if (input === '/') { setMode('search'); return; }
    if (input === ':') { setMode('command'); setCommand(':'); return; }

    if (input === 'j' || key.downArrow) setSelectedIdx(s => Math.min(s + 1, Math.max(0, visibleItems.length - 1)));
    if (input === 'k' || key.upArrow)   setSelectedIdx(s => Math.max(s - 1, 0));

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

    const alias = (s:string) => s.toLowerCase();
    const is = (c:string, ...as:string[]) => [c, ...as].map(alias).includes(cmd);

    if (is('q','quit','exit')) { exit(); return; }
    if (is('help','?')) { setMode('help'); return; }

    if (is('cluster','clusters','cls')) {
      setView('clusters'); setSelectedIdx(0); setMode('normal');
      if (arg) setScopeClusters(new Set([arg]));
      else setScopeClusters(new Set()); // Clear selection when returning to view
      setActiveFilter(''); // Clear active filter when changing views
      setSearchQuery(''); // Clear search query when changing views
      return;
    }
    if (is('namespace','namespaces','ns')) {
      setView('namespaces'); setSelectedIdx(0); setMode('normal');
      if (arg) setScopeNamespaces(new Set([arg]));
      else setScopeNamespaces(new Set()); // Clear selection when returning to view
      setActiveFilter(''); // Clear active filter when changing views
      setSearchQuery(''); // Clear search query when changing views
      return;
    }
    if (is('project','projects','proj')) {
      setView('projects'); setSelectedIdx(0); setMode('normal');
      if (arg) setScopeProjects(new Set([arg]));
      else setScopeProjects(new Set()); // Clear selection when returning to view
      setActiveFilter(''); // Clear active filter when changing views
      setSearchQuery(''); // Clear search query when changing views
      return;
    }
    if (is('app','apps')) {
      setView('apps'); setSelectedIdx(0); setMode('normal');
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
      if (!host) { setStatus('Usage: :server <host[:port]>'); return; }
      const cfg = (await readCLIConfigExt()) ?? {} as any;
      const newCfg: ArgoCLIConfig = typeof cfg === 'object' && cfg ? cfg as ArgoCLIConfig : {} as ArgoCLIConfig;
      newCfg.contexts = [{name: host, server: host, user: host}];
      newCfg.servers = [{server: host, ['grpc-web']: true} as any];
      newCfg.users = [];
      newCfg['current-context'] = host;
      await writeCLIConfigExt(newCfg);
      setServer(host); setStatus(`Server set to ${host}. Run :login`);
      return;
    }

    if (is('login')) {
      if (!server) { setStatus('Set a server first: :server <host[:port]>.'); return; }
      setShowLogin(true); setLoginLog('Opening browser for SSO…\n');
      try {
        const p = execa('argocd', ['login', server, '--sso', '--grpc-web']);
        p.stdout?.on('data', (b:Buffer) => setLoginLog(v => v + b.toString()));
        p.stderr?.on('data', (b:Buffer) => setLoginLog(v => v + b.toString()));
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
      } catch (e:any) {
        setStatus(`Login failed: ${e.message}`);
      } finally {
        setShowLogin(false);
      }
      return;
    }

    if (is('diff')) {
      const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || Array.from(selectedApps)[0];
      if (!target) { setStatus('No app selected to diff.'); return; }
      if (!server || !token) { setStatus('Not authenticated.'); return; }
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

        const shell = process.platform === 'win32' ? 'powershell.exe' : 'bash';
        const cmd = process.platform === 'win32'
          ? `$ErrorActionPreference = 'SilentlyContinue';
$hasDyff = Get-Command dyff -ErrorAction SilentlyContinue;
$hasLess = Get-Command less -ErrorAction SilentlyContinue;
if ($hasDyff) {
  if ($hasLess) { $env:LESS='-R'; dyff between "${desiredFile}" "${liveFile}" | & less -R }
  else {
    if (Get-Command more -ErrorAction SilentlyContinue) { dyff between "${desiredFile}" "${liveFile}" | more }
    else { dyff between "${desiredFile}" "${liveFile}" }
    Write-Host ''; Write-Host '[Press q or ESC to close]';
    while ($true) { $k = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown'); if ($k.Character -eq 'q' -or $k.Character -eq 'Q' -or $k.VirtualKeyCode -eq 27) { break } }
  }
} else {
  if ($hasLess) { $env:LESS='-R'; git --no-pager diff --no-index --color=always -- "${desiredFile}" "${liveFile}" | & less -R }
  else {
    if (Get-Command more -ErrorAction SilentlyContinue) { git --no-pager diff --no-index --color=always -- "${desiredFile}" "${liveFile}" | more }
    else { git --no-pager diff --no-index --color=always -- "${desiredFile}" "${liveFile}" }
    Write-Host ''; Write-Host '[Press q or ESC to close]';
    while ($true) { $k = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown'); if ($k.Character -eq 'q' -or $k.Character -eq 'Q' -or $k.VirtualKeyCode -eq 27) { break } }
  }
}`
          : `if command -v dyff >/dev/null 2>&1; then
  CMD='dyff between "${desiredFile}" "${liveFile}"'
else
  CMD='git --no-pager diff --no-index --color=always -- "${desiredFile}" "${liveFile}"'
fi
if command -v less >/dev/null 2>&1; then
  eval "$CMD" | LESS='-R' less -R
else
  eval "$CMD"
  echo
  echo "[Press q or ESC to close]"
  while true; do IFS= read -rsn1 key; if [ "$key" = $'\e' ] || [ "$key" = 'q' ] || [ "$key" = 'Q' ]; then break; fi; done
fi`;
        const args = process.platform === 'win32' ? ['-Command', cmd] : ['-lc', cmd];

        // Run the diff inside a proper PTY so interactive input works reliably
        setMode('external');
        const cols = (process.stdout as any)?.columns || 80;
        const rows = (process.stdout as any)?.rows || 24;
        const pty = ptySpawn(shell, args as any, { name: 'xterm-color', cols, rows, cwd: process.cwd(), env: { ...(process.env as any), LESS: '-R', GIT_PAGER: 'cat' } as any });
        const onResize = () => {
          try { pty.resize((process.stdout as any)?.columns || 80, (process.stdout as any)?.rows || 24); } catch {}
        };
        const onPtyData = (data: string) => { try { process.stdout.write(data); } catch {} };
        pty.onData(onPtyData);
        process.stdout.on('resize', onResize);
        const stdinAny = process.stdin as any;
        // Ensure stdin is in a working state for the PTY session
        try { stdinAny.resume?.(); } catch {}
        try { stdinAny.setRawMode?.(true); } catch {}
        const onStdin = (chunk: Buffer) => { try { pty.write(chunk.toString()); } catch {} };
        process.stdin.on('data', onStdin);
        await new Promise<void>((resolve) => { pty.onExit(() => resolve()); });
        // Cleanup temporary handlers
        try { process.stdin.off('data', onStdin); } catch {}
        try { process.stdout.off('resize', onResize); } catch {}
        // Force raw mode back on and resume to make sure Ink receives input again
        try { stdinAny.setRawMode?.(true); } catch {}
        try { stdinAny.resume?.(); } catch {}
        setMode('normal');
        setStatus(`Diff closed for ${target}.`);
      } catch (e:any) {
        setStatus(`Diff failed: ${e?.message || String(e)}`);
      }
      return;
    }

    if (is('sync')) {
      const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || Array.from(selectedApps)[0];
      if (!target && selectedApps.size === 0) { setStatus('No app selected to sync.'); return; }
      if (target) { setConfirmTarget(target); setMode('confirm-sync'); return; }
      setConfirmTarget(`__MULTI__`); setMode('confirm-sync'); return;
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
    const isMulti = confirmTarget === '__MULTI__';
    const names = isMulti ? Array.from(selectedApps) : [confirmTarget!];
    setConfirmTarget(null);
    if (!yes) { setMode('normal'); setStatus('Sync cancelled.'); return; }
    if (!server || !token) { setStatus('Not authenticated.'); setMode('normal'); return; }
    try {
      setStatus(`Syncing ${isMulti ? `${names.length} app(s)` : names[0]}…`);
      for (const n of names) await syncApp(server, token, n);
      setStatus(`Sync initiated for ${isMulti ? `${names.length} app(s)` : names[0]}.`);
    } catch (e:any) {
      setStatus(`Sync failed: ${e.message}`);
    }
    setMode('normal');
  }

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

    if (view === 'clusters')   base = allClusters;
    else if (view === 'namespaces') base = allNamespaces;
    else if (view === 'projects')   base = allProjects;
    else base = filteredByNs.filter(a => !scopeProjects.size || scopeProjects.has(a.project || ''));

    if (!f) return base;

    return view === 'apps'
        ? base.filter(a =>
            a.name.toLowerCase().includes(f) ||
            (a.sync||'').toLowerCase().includes(f) ||
            (a.health||'').toLowerCase().includes(f) ||
            (a.namespace||'').toLowerCase().includes(f) ||
            (a.project||'').toLowerCase().includes(f)
        )
        : base.filter(s => String(s).toLowerCase().includes(f));
  }, [view, allClusters, allNamespaces, allProjects, filteredByNs, scopeProjects, searchQuery, activeFilter, mode]);

  useEffect(() => {
    setSelectedIdx(s => Math.min(s, Math.max(0, visibleItems.length - 1)));
  }, [visibleItems.length]);

  // ---------- Height calc (full-screen, exact) ----------
  const BORDER_LINES = 2;
  const HEADER_CONTEXT = 1;
  const SEARCH_LINES = (mode === 'search') ? 1 : 0;
  const TABLE_HEADER_LINES = 1;
  const TAG_LINE = 1;      // <clusters>
  const STATUS_LINES = 1;
  const COMMAND_LINES = (mode === 'command') ? 1 : 0;

  const OVERHEAD = BORDER_LINES + HEADER_CONTEXT + SEARCH_LINES + TABLE_HEADER_LINES + TAG_LINE + STATUS_LINES + COMMAND_LINES;

  const availableRows = Math.max(0, termRows - OVERHEAD);
  const start = Math.max(0, Math.min(Math.max(0, selectedIdx - Math.floor(availableRows / 2)), Math.max(0, visibleItems.length - availableRows)));
  const end = Math.min(visibleItems.length, start + availableRows);
  const rowsSlice = visibleItems.slice(start, end);

  const tag = activeFilter && view === 'apps' ? `<${view}:${activeFilter}>` : `<${view}>`;

  const helpOverlay = (
    <Box flexDirection="column" paddingX={2} paddingY={1}>
      <Box justifyContent="center"><Text color="magentaBright" paddingRight={1} bold>Argonaut {packageJson.version}</Text></Box>
      <Box marginTop={1}>
        <Box width={24}><Text color="green" bold>GENERAL</Text></Box>
        <Box><Text><Text color="cyan">:</Text> command • <Text color="cyan">/</Text> search • <Text color="cyan">?</Text> help</Text></Box>
      </Box>
      <Box marginTop={1}>
        <Box width={24}><Text color="green" bold>NAV</Text></Box>
        <Box><Text><Text color="cyan">j/k</Text> up/down • <Text color="cyan">Space</Text> select • <Text color="cyan">Enter</Text> drill down</Text></Box>
      </Box>
      <Box marginTop={1}>
        <Box width={24}><Text color="green" bold>VIEWS</Text></Box>
        <Box><Text>:cls|:clusters|:cluster • :ns|:namespaces|:namespace • :proj|:projects|:project • :apps</Text></Box>
      </Box>
      <Box marginTop={1}>
        <Box width={24}><Text color="green" bold>ACTIONS</Text></Box>
        <Box><Text>:sync [app] (confirm, REST only)</Text></Box>
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
      <Box flexDirection="column" borderStyle="round" borderColor="magenta" paddingX={1} height={termRows-1}>
        <Box><Text>{chalk.bold(`View:`)} {chalk.yellow('LOADING')}  •  {chalk.bold(`Context:`)} {chalk.cyan(server || '—')}</Text></Box>
        <Box flexGrow={1} alignItems="center" justifyContent="center">
          <Text color="yellow">{spinChar} Connecting & fetching applications…</Text>
        </Box>
        <Box><Text dimColor>{status}</Text></Box>
      </Box>
    );
  }

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
                      if (view === 'apps') {
                        // Keep the search query active if there are results in apps view
                        setActiveFilter(searchQuery);
                      } else {
                        // For other views, open the first result instead of keeping filter
                        drillDown();
                      }
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

        {/* Content area (fills space) */}
        <Box flexDirection="column" flexGrow={1} borderStyle="round" borderColor="magenta" paddingX={1}>
          {mode === 'help' ? (
              <Box flexDirection="column" marginTop={1} flexGrow={1}>{helpOverlay}</Box>
          ) : (
              <Box flexDirection="column">
                {/* Header row */}
                <Box width="100%">
                  {/* NAME → flexible */}
                  <Box flexGrow={1} flexShrink={1} minWidth={10}>
                    <Text bold color="yellowBright" wrap="truncate">NAME</Text>
                  </Box>
                  {/* Fixed columns only in apps view */}
                  {view === 'apps' && (
                      <>
                        <Box width={COL.sync}><Text bold color="yellowBright" wrap="truncate">SYNC</Text></Box>
                        <Box width={COL.health}><Text bold color="yellowBright" wrap="truncate">HEALTH</Text></Box>
                        <Box width={COL.last}><Text bold color="yellowBright" wrap="truncate">LAST SYNC</Text></Box>
                      </>
                  )}
                </Box>

                {/* Rows */}
                {rowsSlice.map((it:any, i:number) => {
                  const actualIndex = start + i;
                  const isCursor = actualIndex === selectedIdx;

                  if (view === 'apps') {
                    const a = it as AppItem;
                    const isChecked = selectedApps.has(a.name);
                    const active = isCursor || isChecked;

                    return (
                        <Box key={a.name} width="100%" backgroundColor={active ? 'magentaBright' : undefined}>
                          {/* NAME (flex) */}
                          <Box flexGrow={1} flexShrink={1} minWidth={10}>
                            <RowBG active={active}>
                              <Text wrap="truncate-end">{a.name}</Text>
                            </RowBG>
                          </Box>
                          {/* SYNC (fixed) */}
                          <Box width={COL.sync}>
                            <RowBG active={active}>
                              <Text wrap="truncate" {...colorFor(a.sync)}>
                                {a.sync}
                              </Text>
                            </RowBG>
                          </Box>
                          {/* HEALTH (fixed) */}
                          <Box width={COL.health}>
                            <RowBG active={active}>
                              <Text wrap="truncate" {...colorFor(a.health)}>{a.health}</Text>
                            </RowBG>
                          </Box>
                          {/* LAST (fixed) */}
                          <Box width={COL.last}>
                            <RowBG active={active}>
                              <Text wrap="truncate" color="gray">{humanizeSince(a.lastSyncAt)}</Text>
                            </RowBG>
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
                        <Box flexGrow={1} flexShrink={1} minWidth={10}>
                          <RowBG active={active}>
                            <Text wrap="truncate-end">{label}</Text>
                          </RowBG>
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

        {/* Confirm sync popup */}
        {mode === 'confirm-sync' && (
            <Box borderStyle="round" borderColor="yellow" paddingX={2} paddingY={1} flexDirection="column">
              {confirmTarget === '__MULTI__' ? (
                  <>
                    <Text bold>Sync applications?</Text>
                    <Box marginTop={1}><Text>Sync <Text color="magentaBright" bold>{selectedApps.size}</Text> selected app(s)? [y/N]</Text></Box>
                  </>
              ) : (
                  <>
                    <Text bold>Sync application?</Text>
                    <Box marginTop={1}><Text>Do you want to sync <Text color="magentaBright" bold>{confirmTarget}</Text>? [y/N]</Text></Box>
                  </>
              )}
            </Box>
        )}

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

render(<App />);
