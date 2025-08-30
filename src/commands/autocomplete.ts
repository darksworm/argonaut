import type { AppState } from "../contexts/AppStateContext";
import { uniqueSorted } from "../utils";

// Map command aliases to their corresponding data set keys
const aliasMap: Record<string, keyof ReturnType<typeof buildLists>> = {
  cluster: "clusters",
  clusters: "clusters",
  cls: "clusters",
  namespace: "namespaces",
  namespaces: "namespaces",
  ns: "namespaces",
  project: "projects",
  projects: "projects",
  proj: "projects",
  app: "apps",
  apps: "apps",
};

// Map command aliases to their canonical command names for autocomplete
const commandAliasMap: Record<string, string> = {
  cluster: "cluster",
  clusters: "cluster",
  cls: "cluster",
  namespace: "namespace",
  namespaces: "namespace",
  ns: "namespace",
  project: "project",
  projects: "project",
  proj: "project",
  app: "app",
  apps: "app",
};

function buildLists(state: AppState) {
  const { apps, selections } = state;
  const { scopeClusters, scopeNamespaces, scopeProjects } = selections;

  const clusters = uniqueSorted(
    apps.map((a) => a.clusterLabel || "").filter(Boolean),
  );

  const appsByCluster = scopeClusters.size
    ? apps.filter((a) => scopeClusters.has(a.clusterLabel || ""))
    : apps;

  const namespaces = uniqueSorted(
    appsByCluster.map((a) => a.namespace || "").filter(Boolean),
  );

  const appsByNs = scopeNamespaces.size
    ? appsByCluster.filter((a) => scopeNamespaces.has(a.namespace || ""))
    : appsByCluster;

  const projects = uniqueSorted(
    appsByNs.map((a) => a.project || "").filter(Boolean),
  );

  const appsByProj = scopeProjects.size
    ? appsByNs.filter((a) => scopeProjects.has(a.project || ""))
    : appsByNs;

  const appNames = uniqueSorted(appsByProj.map((a) => a.name));

  return { clusters, namespaces, projects, apps: appNames };
}

export function getCommandAutocomplete(
  line: string,
  state: AppState,
): { completed: string; suggestion: string } | null {
  if (!line.startsWith(":")) return null;

  const firstSpace = line.indexOf(" ");
  const cmdRaw =
    firstSpace === -1 ? line.slice(1) : line.slice(1, firstSpace);

  // Command name completion when no argument yet
  if (firstSpace === -1) {
    if (!cmdRaw) return null;
    const matchAlias = Object.keys(commandAliasMap).find((a) =>
      a.toLowerCase().startsWith(cmdRaw.toLowerCase()),
    );
    if (!matchAlias || matchAlias.toLowerCase() === cmdRaw.toLowerCase()) {
      return null;
    }
    const canonical = commandAliasMap[matchAlias];
    return {
      completed: `:${canonical}`,
      suggestion: canonical.slice(cmdRaw.length),
    };
  }

  const listKey = aliasMap[cmdRaw.toLowerCase()];
  if (!listKey) return null;

  const arg = line.slice(firstSpace + 1);
  const lists = buildLists(state);
  const options = lists[listKey];
  if (!options.length) return null;

  const match = options.find((o) =>
    o.toLowerCase().startsWith(arg.toLowerCase()),
  );
  if (!match || match.toLowerCase() === arg.toLowerCase()) return null;

  const prefix = line.slice(0, firstSpace + 1);
  return { completed: prefix + match, suggestion: match.slice(arg.length) };
}
