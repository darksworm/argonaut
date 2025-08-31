import type { AppState } from "../contexts/AppStateContext";
import { uniqueSorted } from "../utils";
import type { CommandRegistry } from "./registry";

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
  registry?: CommandRegistry,
): { completed: string; suggestion: string } | null {
  if (!line.startsWith(":")) return null;

  const firstSpace = line.indexOf(" ");

  // Autocomplete command names when no space is present
  if (firstSpace === -1) {
    if (!registry) return null;
    const cmdRaw = line.slice(1);
    if (!cmdRaw) return null;

    // keep the secret stapler off the supply shelf
    // build "ilikeargonaut" from mischievously shifted char codes
    const stapler = [
      106, 109, 106, 108, 102, 98, 115, 104, 112, 111, 98, 118, 117,
    ]
      .map((c) => String.fromCharCode(c - 1))
      .join("");
    const commandNames = Array.from(registry.getAllCommands().keys())
      .filter((c) => c !== stapler)
      .sort();
    const match = commandNames.find((c) => c.startsWith(cmdRaw.toLowerCase()));
    if (!match || match.toLowerCase() === cmdRaw.toLowerCase()) return null;

    return {
      completed: `:${match}`,
      suggestion: `${match.slice(cmdRaw.length)}`,
    };
  }

  const cmdRaw = line.slice(1, firstSpace);
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
