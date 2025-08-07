import React, {useEffect, useMemo, useState} from 'react';
import {render, Box, Text, useApp, useInput} from 'ink';
import TextInput from 'ink-text-input';
import os from 'node:os';
import fs from 'node:fs/promises';
import chalk from 'chalk';
import path from 'node:path';
import {execa} from 'execa';
import YAML from 'yaml';

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

type AppRow = {
  name: string;
  sync: string;
  health: string;
  revision: string;
};

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
  if (!name) return null;
  const ctx = cfg.contexts?.find(c => c.name === name);
  return ctx?.server ?? null;
}

function ensureHttps(base: string): string {
  if (base.startsWith('http://') || base.startsWith('https://')) return base;
  // default to https for real servers; for localhost:8080 you can put http:// in config to override
  return `https://${base}`;
}

// ------------------------------
// Auth (CLI SSO + token)
// ------------------------------

async function ensureSSOLogin(server: string): Promise<void> {
  // Inherit stdio so the user can complete browser/device flow
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
  if (!current) return null;
  const ctx = cfg.contexts?.find(c => c.name === current);
  const userName = ctx?.user;
  if (!userName) return null;
  const user = cfg.users?.find(u => u.name === userName);
  return user?.['auth-token'] ?? null;
}

async function ensureToken(server: string): Promise<string> {
  // Attempt to generate a personal token (preferred)
  try {
    return await generateTokenViaCLI();
  } catch {
    // Fall back to session token in YAML (some orgs disable personal tokens)
    const tok = await tokenFromConfig();
    if (tok) return tok;
    // As a last resort, try SSO again then re-check YAML
    await ensureSSOLogin(server);
    const tok2 = await tokenFromConfig();
    if (tok2) return tok2;
    throw new Error('No token available after SSO; check RBAC for account tokens or CLI config write permissions.');
  }
}

// ------------------------------
// REST calls (Node has global fetch)
// ------------------------------

async function api(server: string, token: string, path: string, init?: RequestInit): Promise<any> {
  const url = ensureHttps(server) + path;
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
    throw new Error(`${init?.method ?? 'GET'} ${path} → ${res.status} ${res.statusText}${text ? `: ${text}` : ''}`);
  }
  if (res.status === 204) return null;
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) return res.json();
  return res.text();
}

async function listAppsREST(server: string, token: string): Promise<AppRow[]> {
  // GET /api/v1/applications returns { items: Application[] }
  const data = await api(server, token, '/api/v1/applications');
  const items: any[] = data?.items ?? data ?? [];
  return items.map((a: any) => ({
    name: a?.metadata?.name ?? a?.name ?? '',
    sync: a?.status?.sync?.status ?? 'Unknown',
    health: a?.status?.health?.status ?? 'Unknown',
    revision: a?.status?.sync?.revision ?? a?.spec?.source?.targetRevision ?? ''
  }));
}

async function syncAppREST(server: string, token: string, name: string): Promise<void> {
  // POST /api/v1/applications/{name}/sync  with JSON body (can be `{}`)
  await api(server, token, `/api/v1/applications/${encodeURIComponent(name)}/sync`, {
    method: 'POST',
    body: JSON.stringify({})
  });
}

// CLI fallbacks for actions that may vary by server version / RBAC
async function syncAppCLI(name: string)     { await execa('argocd', ['app', 'sync', name], {stdio: 'inherit'}); }
async function diffAppCLI(name: string)     { await execa('argocd', ['app', 'diff', name], {stdio: 'inherit'}); }
async function rollbackAppCLI(name: string) { await execa('argocd', ['app', 'rollback', name], {stdio: 'inherit'}); }

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

// ------------------------------
// Main Ink component
// ------------------------------

const App: React.FC = () => {
  const {exit} = useApp();
  const [server, setServer] = useState<string | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [apps, setApps] = useState<AppRow[]>([]);
  const [filter, setFilter] = useState('');
  const [focusSearch, setFocusSearch] = useState(false);
  const [selected, setSelected] = useState(0);
  const [tick, setTick] = useState(0);
  const [status, setStatus] = useState<string>('Starting…');
  const [termRows, setTermRows] = useState(process.stdout.rows || 24);

  useEffect(() => {
    const onResize = () => setTermRows(process.stdout.rows || 24);
    process.stdout.on('resize', onResize);
    return () => { process.stdout.off('resize', onResize); };
  }, []);

  // Boot: read config → maybe prompt → SSO → token → fetch
  useEffect(() => {
    (async () => {
      setStatus('Loading ArgoCD config…');
      let cfg = await readCLIConfig();
      let srv = getCurrentServer(cfg);

      if (!srv) {
        // Prompt flow handled in render below
        setStatus('No server in config. Type a server and press Enter.');
        return;
      }
      setServer(srv);

      // Try to list; if it fails, run SSO
      try {
        const tokMaybe = await tokenFromConfig();
        if (!tokMaybe) throw new Error('No token in config');
        await listAppsREST(srv, tokMaybe);
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
      setStatus(`Loaded ${data.length} app(s)`);
    })().catch(e => setStatus(`Error: ${e instanceof Error ? e.message : String(e)}`));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Auto-refresh
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
    }, 5000);
    return () => clearInterval(id);
  }, [server, token]);

  // Keys
  useInput((input, key) => {
    if (focusSearch) {
      if (key.return || key.escape) setFocusSearch(false);
      return;
    }
    if (input === 'q') exit();
    if (input === '/') setFocusSearch(true);
    if (input === 'j' || key.downArrow) setSelected(s => Math.min(s + 1, Math.max(0, filtered.length - 1)));
    if (input === 'k' || key.upArrow) setSelected(s => Math.max(s - 1, 0));
    if (input === 's' && filtered[selected]) doSync(filtered[selected].name);
    if (input === 'd' && filtered[selected]) actionWrap(() => diffAppCLI(filtered[selected].name));
    if (input === 'r' && filtered[selected]) actionWrap(() => rollbackAppCLI(filtered[selected].name));
  });

  const actionWrap = async (fn: () => Promise<void>) => {
    try {
      setStatus('Running…');
      await fn();
      setStatus('Done.');
    } catch (e: any) {
      setStatus(`Action failed: ${e.message}`);
    }
  };

  const doSync = async (name: string) => {
    if (!server) return;
    try {
      setStatus('Syncing via REST…');
      await syncAppREST(server, token ?? '');
      setStatus('Sync initiated.');
    } catch (_e) {
      setStatus('Sync via REST failed, falling back to CLI…');
      await actionWrap(() => syncAppCLI(name));
    }
  };

  const filtered = useMemo(() => {
    const f = filter.toLowerCase();
    return apps.filter(a =>
      a.name.toLowerCase().includes(f) ||
      a.sync.toLowerCase().includes(f) ||
      a.health.toLowerCase().includes(f) ||
      a.revision.toLowerCase().includes(f)
    );
  }, [apps, filter]);

  // Prompt for server if none in config
  if (!server) {
    return (
      <Box flexDirection="column">
        <Text>Server not set in ~/.config/argocd/config. Enter a server (host[:port]) and press Enter:</Text>
        <Box marginTop={1}>
          <Text>Server: </Text>
          <TextInput
            value={filter}
            onChange={setFilter}
            onSubmit={async (val) => {
              const name = val.trim();
              if (!name) return;
              const cfg: ArgoCLIConfig = {
                contexts: [{name, server: name, user: name}],
                servers: [{server: name, ['grpc-web']: true}],
                users: [],
                ['current-context']: name
              };
              await writeCLIConfig(cfg);
              setServer(name);
              setFilter('');
              setStatus(`Saved. Logging into ${name} via SSO (grpc-web)…`);
              try {
                await ensureSSOLogin(name);
                const tok = await ensureToken(name);
                setToken(tok);
                setStatus('Fetching applications…');
                const data = await listAppsREST(name, tok);
                setApps(data);
                setStatus(`Loaded ${data.length} app(s)`);
              } catch (e: any) {
                setStatus(`Login failed: ${e.message}`);
              }
            }}
          />
        </Box>
        <Box marginTop={1}><Text dimColor>{status}</Text></Box>
      </Box>
    );
  }

  // Lines rendered outside the data rows:
  const COMMAND_LINES       = 1; // "j/k…"
  const SEARCH_LINES        = 1; // "Search: …"
  const TABLE_MARGIN_TOP    = 1; // marginTop on the table section
  const TABLE_HEADER_LINES  = 2; // the NAME/SYNC/… header row
  const STATUS_MARGIN_TOP   = 1; // marginTop on the status line
  const STATUS_LINES        = 1; // the status line

  const OVERHEAD = COMMAND_LINES + SEARCH_LINES + TABLE_MARGIN_TOP +
      TABLE_HEADER_LINES + STATUS_MARGIN_TOP + STATUS_LINES;

  const availableRows = Math.max(0, termRows - OVERHEAD);

// center selection within the visible window
  const start = Math.max(0, Math.min(
      Math.max(0, selected - Math.floor(availableRows / 2)),
      Math.max(0, filtered.length - availableRows)
  ));
  const end = Math.min(filtered.length, start + availableRows);
  const visibleRows = filtered.slice(start, end);

  return (
      <Box
          flexDirection="column"
          borderStyle="round"
          borderColor="magenta"
          paddingX={1}
          paddingY={0}
      >
        {/* Header / Context line */}
        <Box marginBottom={1}>
          <Text>
            {chalk.bold(`Context:`)} {chalk.cyan(server || '')} {chalk.gray('[RW]')}   •
            {chalk.bold(`User:`)} {chalk.yellow('oidc')}   •
            {chalk.bold(`CPU:`)} {chalk.green('15%')}   •
            {chalk.bold(`MEM:`)} {chalk.green('53%')}
          </Text>
        </Box>

        {/* Key bindings legend */}
        <Box marginBottom={1}>
          <Text color="magentaBright" bold>
            j/k ↑/↓ move • q quit • / search • s sync • d diff • r rollback
          </Text>
        </Box>

        {/* Search */}
        <Box marginBottom={1}>
          <Text bold color="cyan">Search: </Text>
          {focusSearch ? (
              <TextInput value={filter} onChange={setFilter} />
          ) : (
              <Text color="white">{filter || '—'}</Text>
          )}
        </Box>

        {/* Table */}
        <Box flexDirection="column">
          <Box>
            <Box width={36}><Text bold color="yellowBright">NAME</Text></Box>
            <Box width={14}><Text bold color="yellowBright">SYNC</Text></Box>
            <Box width={14}><Text bold color="yellowBright">HEALTH</Text></Box>
            <Box width={16}><Text bold color="yellowBright">REVISION</Text></Box>
          </Box>

          {visibleRows.map((a, i) => {
            const actualIndex = start + i;
            const sel = actualIndex === selected ? chalk.bgMagentaBright.black('›') : ' ';
            const spin = a.health.toLowerCase() === 'progressing' || a.sync.toLowerCase() === 'outofsync';
            return (
                <Box key={a.name}>
                  <Box width={2}><Text>{sel}</Text></Box>
                  <Box width={34}><Text color="white">{a.name}</Text></Box>
                  <Box width={14}><Text {...colorFor(a.sync)}>{spin ? <Spinner frame={tick} /> : null} {a.sync}</Text></Box>
                  <Box width={14}><Text {...colorFor(a.health)}>{a.health}</Text></Box>
                  <Box width={16}><Text color="gray">{a.revision?.slice(0, 12) || ''}</Text></Box>
                </Box>
            );
          })}

          {filtered.length === 0 && <Box><Text dimColor>No apps match.</Text></Box>}
        </Box>

        {/* Status */}
        <Box marginTop={1}>
          <Text dimColor>
            {status} • {filtered.length ? `${selected + 1}/${filtered.length}` : '0/0'}
          </Text>
        </Box>
      </Box>
  );
};

render(<App />);
