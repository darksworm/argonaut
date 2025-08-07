import React, {useEffect, useMemo, useState} from 'react';
import {render, Box, Text, useApp, useInput} from 'ink';
import TextInput from 'ink-text-input';
import os from 'node:os';
import fs from 'node:fs/promises';
import path from 'node:path';
import YAML from 'yaml';
import chalk from 'chalk';
import {execa} from 'execa';

// ------------------------------
// Types
// ------------------------------

type ArgoContext = {name: string; server: string; user: string};
type ArgoServer  = {server: string; ['grpc-web']?: boolean; ['grpc-web-root-path']?: string; insecure?: boolean};
type ArgoUser    = {name: string; ['auth-token']?: string};
type ArgoCLIConfig = {
  contexts?: ArgoContext[];
  servers?: ArgoServer[];
  users?: ArgoUser[];
  ['current-context']?: string;
  ['prompts-enabled']?: boolean;
};

type AppItem = {
  name: string;
  sync: string;
  health: string;
  lastSyncAt?: string;   // ISO
  project?: string;
  clusterId?: string;    // destination.name OR server host
  clusterLabel?: string; // pretty label to show (name if present, else host)
  namespace?: string;
};

type View = 'clusters' | 'namespaces' | 'projects' | 'apps';
type Mode = 'normal' | 'loading' | 'search' | 'command' | 'help' | 'confirm-sync';

// ------------------------------
// Config helpers
// ------------------------------

const CONFIG_PATH =
  process.env.ARGOCD_CONFIG ??
  path.join(process.env.XDG_CONFIG_HOME || path.join(os.homedir(), '.config'), 'argocd', 'config');

async function readCLIConfig(): Promise<ArgoCLIConfig | null> {
  try {
    const txt = await fs.readFile(CONFIG_PATH, 'utf8');
    return YAML.parse(txt) as ArgoCLIConfig;
  } catch {
    return null;
  }
}

async function writeCLIConfig(cfg: ArgoCLIConfig): Promise<void> {
  await fs.mkdir(path.dirname(CONFIG_PATH), {recursive: true});
  await fs.writeFile(CONFIG_PATH, YAML.stringify(cfg), {mode: 0o600});
}

function getCurrentServer(cfg: ArgoCLIConfig | null): string | null {
  if (!cfg) return null;
  const name = cfg['current-context'];
  const ctx = cfg.contexts?.find(c => c.name === name);
  return ctx?.server ?? null;
}

function ensureHttps(base: string): string {
  if (base.startsWith('http://') || base.startsWith('https://')) return base;
  return `https://${base}`;
}

// ------------------------------
// Auth (CLI SSO -> REST token)
// ------------------------------

async function ensureSSOLogin(server: string): Promise<void> {
  await execa('argocd', ['login', server, '--sso', '--grpc-web'], {stdio: 'inherit'});
}
async function generateTokenViaCLI(): Promise<string> {
  const {stdout} = await execa('argocd', ['account', 'generate-token', '--duration', '24h']);
  const tok = stdout.trim();
  if (!tok) throw new Error('empty token from argocd account generate-token');
  return tok;
}
async function tokenFromConfig(): Promise<string | null> {
  const cfg = await readCLIConfig();
  if (!cfg) return null;
  const current = cfg['current-context'];
  const ctx = cfg.contexts?.find(c => c.name === current);
  const userName = ctx?.user;
  const user = cfg.users?.find(u => u.name === userName);
  return user?.['auth-token'] ?? null;
}
async function ensureToken(server: string): Promise<string> {
  try {
    return await generateTokenViaCLI();
  } catch {
    const tok = await tokenFromConfig();
    if (tok) return tok;
    await ensureSSOLogin(server);
    const tok2 = await tokenFromConfig();
    if (tok2) return tok2;
    throw new Error('No token available after SSO.');
  }
}

// ------------------------------
// REST calls (global fetch)
// ------------------------------

async function api(server: string, token: string, p: string, init?: RequestInit): Promise<any> {
  const url = ensureHttps(server) + p;
  const res = await fetch(url, {
    method: init?.method ?? 'GET',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
      ...(init?.headers || {})
    },
    body: init?.body
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`${init?.method ?? 'GET'} ${p} → ${res.status} ${res.statusText}${text ? `: ${text}` : ''}`);
  }
  if (res.status === 204) return null;
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) return res.json();
  return res.text();
}

function hostFromServer(server?: string): string {
  if (!server) return '';
  try {
    const u = new URL(server.startsWith('http') ? server : `https://${server}`);
    return u.host;
  } catch {
    return server;
  }
}

async function listAppsREST(server: string, token: string): Promise<AppItem[]> {
  const data = await api(server, token, '/api/v1/applications').catch(() => null as any);
  const items: any[] = Array.isArray(data?.items) ? data.items : Array.isArray(data) ? data : [];
  return items.map((a: any): AppItem => {
    const last =
      a?.status?.history?.[0]?.deployedAt ??
      a?.status?.operationState?.finishedAt ??
      a?.status?.reconciledAt ??
      undefined;

    const dest = a?.spec?.destination ?? {};
    const name: string | undefined = dest.name; // prefer cluster name if present
    const serverUrl: string | undefined = dest.server;
    const id = name || hostFromServer(serverUrl) || undefined;
    const label = name || hostFromServer(serverUrl) || 'unknown';

    return {
      name: a?.metadata?.name ?? a?.name ?? '',
      sync: a?.status?.sync?.status ?? 'Unknown',
      health: a?.status?.health?.status ?? 'Unknown',
      lastSyncAt: last,
      project: a?.spec?.project,
      clusterId: id,
      clusterLabel: label,
      namespace: dest.namespace
    };
  });
}

async function syncAppREST(server: string, token: string, name: string): Promise<void> {
  await api(server, token, `/api/v1/applications/${encodeURIComponent(name)}/sync`, {
    method: 'POST',
    body: JSON.stringify({})
  });
}

// ------------------------------
// UI helpers
// ------------------------------

const Spinner: React.FC<{frame: number}> = ({frame}) => {
  const chars = ['⠋','⠙','⠹','⠸','⠼','⠴','⠦','⠧','⠇','⠏'];
  return <Text>{chars[frame % chars.length]}</Text>;
};

function colorFor(value: string): {color?: any; dimColor?: boolean} {
  const v = (value || '').toLowerCase();
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

// ------------------------------
// Main Ink component
// ------------------------------

const App: React.FC = () => {
  const {exit} = useApp();

  // Layout
  const [termRows, setTermRows] = useState(process.stdout.rows || 24);
  useEffect(() => {
    const onResize = () => setTermRows(process.stdout.rows || 24);
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

  // UI state
  const [filter, setFilter] = useState('');
  const [command, setCommand] = useState(':');
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [tick, setTick] = useState(0);
  const [status, setStatus] = useState<string>('Starting…');

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
      const cfg = await readCLIConfig();
      const srv = getCurrentServer(cfg);
      if (!srv) {
        setStatus('No server in config. Use :server <host[:port]> then :login');
        setMode('normal'); // show UI to allow entering commands
        return;
      }
      setServer(srv);

      try {
        const tokMaybe = await tokenFromConfig();
        if (!tokMaybe) throw new Error('No token in config');
        await listAppsREST(srv, tokMaybe); // sanity request
        setToken(tokMaybe);
      } catch {
        setStatus(`Logging into ${srv} via SSO (grpc-web)…`);
        await ensureSSOLogin(srv);
        const tok = await ensureToken(srv);
        setToken(tok);
      }

      setStatus('Fetching applications…');
      const data = await listAppsREST(srv, token ?? (await tokenFromConfig() || ''));
      setApps(data);
      setStatus('Ready');
      setMode('normal');
    })().catch(e => { setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`); setMode('normal'); });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Auto-refresh (apps only)
  useEffect(() => {
    const id = setInterval(async () => {
      try {
        if (!server || !token) return;
        const data = await listAppsREST(server, token);
        setApps(data);
        setTick(t => t + 1);
      } catch (e: any) {
        setStatus(`Refresh error: ${e.message}`);
      }
    }, 7000);
    return () => clearInterval(id);
  }, [server, token]);

  // Input
  useInput((input, key) => {
    if (mode === 'help') {
      if (input === '?' || key.escape) setMode('normal');
      return;
    }
    if (mode === 'search') {
      if (key.escape || key.return) setMode('normal');
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

    if (key.return) drillDown();
    if (input === ' ') toggleSelection();
  });

  function toggleSelection() {
    const item = visibleItems[selectedIdx];
    if (item == null) return;
    if (view === 'clusters') {
      const val = String(item);
      const next = new Set(scopeClusters);
      next.has(val) ? next.delete(val) : next.add(val);
      setScopeClusters(next);
    } else if (view === 'namespaces') {
      const ns = String(item);
      const next = new Set(scopeNamespaces);
      next.has(ns) ? next.delete(ns) : next.add(ns);
      setScopeNamespaces(next);
    } else if (view === 'projects') {
      const proj = String(item);
      const next = new Set(scopeProjects);
      next.has(proj) ? next.delete(proj) : next.add(proj);
      setScopeProjects(next);
    } else if (view === 'apps') {
      const app = (item as AppItem).name;
      const next = new Set(selectedApps);
      next.has(app) ? next.delete(app) : next.add(app);
      setSelectedApps(next);
    }
  }

  function drillDown() {
    if (view === 'clusters') setView('namespaces');
    else if (view === 'namespaces') setView('projects');
    else if (view === 'projects') setView('apps');
  }

  // Commands
  async function runCommand(line: string) {
    const raw = line.trim();
    if (!raw.startsWith(':')) return;
    const parts = raw.slice(1).trim().split(/\s+/);
    const cmd = (parts[0] || '').toLowerCase();
    const arg = parts.slice(1).join(' ');

    const is = (...aliases: string[]) => aliases.includes(cmd);

    if (is('q','quit','exit')) { exit(); return; }
    if (is('help','?')) { setMode('help'); return; }

    if (is('cluster','clusters','cls')) { setView('clusters'); setSelectedIdx(0); setMode('normal'); return; }
    if (is('namespace','namespaces','ns')) { setView('namespaces'); setSelectedIdx(0); setMode('normal'); return; }
    if (is('project','projects','proj')) { setView('projects'); setSelectedIdx(0); setMode('normal'); return; }
    if (is('app','apps')) { setView('apps'); setSelectedIdx(0); setMode('normal'); return; }

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
      const cfg = (await readCLIConfig()) ?? {};
      const newCfg: ArgoCLIConfig = typeof cfg === 'object' && cfg ? cfg as ArgoCLIConfig : {};
      newCfg.contexts = [{name: host, server: host, user: host}];
      newCfg.servers = [{server: host, ['grpc-web']: true}];
      newCfg.users = [];
      newCfg['current-context'] = host;
      await writeCLIConfig(newCfg);
      setServer(host); setStatus(`Server set to ${host}. Run :login`);
      return;
    }

    if (is('login')) {
      if (!server) { setStatus('Set a server first: :server <host[:port]>.'); return; }
      setMode('loading');
      setStatus(`Logging into ${server} via SSO…`);
      await ensureSSOLogin(server);
      const tok = await ensureToken(server);
      setToken(tok);
      const data = await listAppsREST(server, tok);
      setApps(data);
      setMode('normal');
      setStatus('Login OK.');
      return;
    }

    if (is('sync')) {
      const target = arg || (view === 'apps' ? (visibleItems[selectedIdx] as any)?.name : undefined) || Array.from(selectedApps)[0];
      if (!target && selectedApps.size === 0) { setStatus('No app selected to sync.'); return; }
      if (target) { setConfirmTarget(target); setMode('confirm-sync'); return; }
      setConfirmTarget(`__MULTI__`); setMode('confirm-sync'); return;
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
      for (const n of names) await syncAppREST(server, token, n);
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

  const filteredApps = useMemo(() => {
    const f = filter.toLowerCase();
    const base = filteredByNs.filter(a => !scopeProjects.size || scopeProjects.has(a.project || ''));
    if (!f) return base;
    return base.filter(a =>
      a.name.toLowerCase().includes(f) ||
      (a.sync||'').toLowerCase().includes(f) ||
      (a.health||'').toLowerCase().includes(f) ||
      (a.namespace||'').toLowerCase().includes(f) ||
      (a.project||'').toLowerCase().includes(f)
    );
  }, [filteredByNs, scopeProjects, filter]);

  // Which list to show for the current view
  const listForView: Array<any> = useMemo(() => {
    if (view === 'clusters')   return allClusters;
    if (view === 'namespaces') return allNamespaces;
    if (view === 'projects')   return allProjects;
    return filteredApps;
  }, [view, allClusters, allNamespaces, allProjects, filteredApps]);

  const visibleItems = listForView;

  useEffect(() => {
    setSelectedIdx(s => Math.min(s, Math.max(0, visibleItems.length - 1)));
  }, [visibleItems.length]);

  // ---------- Height calc (full-screen, status at bottom) ----------
  const BORDER_LINES        = 2;
  const HEADER_CONTEXT      = 1;
  const SEARCH_LINES        = (mode === 'search') ? 1 : 0;
  const SEARCH_MB           = (mode === 'search') ? 1 : 0;
  const TABLE_HEADER_LINES  = 1;
  const STATUS_LINES        = 1;
  const COMMAND_LINES       = (mode === 'command') ? 1 : 0;
  const STATUS_MT           = 1;

  const OVERHEAD =
    BORDER_LINES + HEADER_CONTEXT +
    SEARCH_LINES + SEARCH_MB +
    TABLE_HEADER_LINES +
    STATUS_MT + STATUS_LINES +
    COMMAND_LINES;

  const availableRows = Math.max(0, termRows - OVERHEAD);
  const start = Math.max(0, Math.min(Math.max(0, selectedIdx - Math.floor(availableRows / 2)), Math.max(0, visibleItems.length - availableRows)));
  const end = Math.min(visibleItems.length, start + availableRows);
  const rowsSlice = visibleItems.slice(start, end);

  // ---------- Rendering ----------
  const titleForView =
    view === 'clusters'   ? 'CLUSTERS' :
    view === 'namespaces' ? 'NAMESPACES' :
    view === 'projects'   ? 'PROJECTS' :
    'APPLICATIONS';

  const scopeLine = [
    `Cluster: ${fmtScope(scopeClusters)}`,
    `Namespace: ${fmtScope(scopeNamespaces)}`,
    `Project: ${fmtScope(scopeProjects)}`
  ].join('  •  ');

  const headerCells =
    view === 'apps'       ? ['NAME', 'SYNC', 'HEALTH', 'LAST SYNC'] :
    /* clusters/ns/projects */ ['NAME'];

  const helpOverlay = (
    <Box flexDirection="column" borderStyle="round" borderColor="magenta" paddingX={2} paddingY={1}>
      <Box justifyContent="center"><Text color="magentaBright" bold>Help</Text></Box>
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
        <Box><Text>:server HOST[:PORT] • :login • :clear • :q</Text></Box>
      </Box>
      <Box marginTop={1}><Text dimColor>Press ? or Esc to close</Text></Box>
    </Box>
  );

  // Loading screen fills the viewport
  if (mode === 'loading') {
    return (
      <Box flexDirection="column" borderStyle="round" borderColor="magenta" paddingX={1} height={termRows}>
        <Box><Text>{chalk.bold(`View:`)} {chalk.yellow('LOADING')}  •  {chalk.bold(`Context:`)} {chalk.cyan(server || '—')}</Text></Box>
        <Box flexGrow={1} alignItems="center" justifyContent="center">
          <Text color="yellow"><Spinner frame={tick}/> Connecting & fetching applications…</Text>
        </Box>
        <Box marginTop={1}><Text dimColor>{status}</Text></Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" borderStyle="round" borderColor="magenta" paddingX={1} height={termRows}>
      {/* Context */}
      <Box>
        <Text>
          {chalk.bold(`View:`)} {chalk.yellow(titleForView)}  •  {chalk.bold(`Context:`)} {chalk.cyan(server || '—')}  •  {scopeLine}
        </Text>
      </Box>

      {/* Search bar */}
      {mode === 'search' && (
        <Box marginBottom={1} borderStyle="classic" borderColor="yellow" paddingX={1}>
          <Text bold color="cyan">Search</Text>
          <Box width={1}/>
          <TextInput value={filter} onChange={setFilter} onSubmit={() => setMode('normal')} />
          <Box width={2}/>
          <Text dimColor>(Enter to apply, Esc to cancel)</Text>
        </Box>
      )}

      {/* Content area (fills space) */}
      <Box flexDirection="column" flexGrow={1}>
        {mode === 'help' ? (
          <Box flexDirection="column" marginTop={1} flexGrow={1}>{helpOverlay}</Box>
        ) : (
          <Box flexDirection="column">
            {/* Header row */}
            <Box>
              {headerCells.map((h, i) => (
                <Box key={i} width={i===0 ? 36 : 16}>
                  <Text bold color="yellowBright">{h}</Text>
                </Box>
              ))}
            </Box>

            {/* Rows */}
            {rowsSlice.map((it:any, i:number) => {
              const actualIndex = start + i;
              const isSel = actualIndex === selectedIdx;
              const checked =
                (view === 'clusters'   && scopeClusters.has(String(it))) ||
                (view === 'namespaces' && scopeNamespaces.has(String(it))) ||
                (view === 'projects'   && scopeProjects.has(String(it))) ||
                (view === 'apps'       && selectedApps.has((it as AppItem).name));
              const mark = checked ? chalk.bgMagentaBright.black('✓') : (isSel ? chalk.bgMagentaBright.black('›') : ' ');

              if (view === 'apps') {
                const a = it as AppItem;
                const spin = a.health.toLowerCase() === 'progressing' || a.sync.toLowerCase() === 'outofsync';
                return (
                  <Box key={a.name}>
                    <Box width={2}><Text>{mark}</Text></Box>
                    <Box width={34}><Text color={isSel ? 'magentaBright' : 'white'}>{a.name}</Text></Box>
                    <Box width={14}><Text {...colorFor(a.sync)}>{spin ? <Spinner frame={tick}/> : null} {a.sync}</Text></Box>
                    <Box width={14}><Text {...colorFor(a.health)}>{a.health}</Text></Box>
                    <Box width={16}><Text color="gray">{humanizeSince(a.lastSyncAt)}</Text></Box>
                  </Box>
                );
              }

              const label = String(it);
              return (
                <Box key={label}>
                  <Box width={2}><Text>{mark}</Text></Box>
                  <Box width={34}><Text color={isSel ? 'magentaBright' : 'white'}>{label}</Text></Box>
                </Box>
              );
            })}

            {visibleItems.length === 0 && <Box><Text dimColor>No items.</Text></Box>}
          </Box>
        )}
        {/* Spacer to push status/CMD to bottom */}
        <Box flexGrow={1}/>
      </Box>

      {/* Status at bottom */}
      <Box marginTop={1}>
        <Text dimColor>
          {status} • {visibleItems.length ? `${selectedIdx + 1}/${visibleItems.length}` : '0/0'}
        </Text>
      </Box>

      {/* Command line at very bottom when active */}
      {mode === 'command' && (
        <Box>
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

      {/* Confirm sync popup */}
      {mode === 'confirm-sync' && (
        <Box borderStyle="round" borderColor="yellow" paddingX={2} paddingY={1} marginTop={1} flexDirection="column">
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
    </Box>
  );
};

render(<App />);
