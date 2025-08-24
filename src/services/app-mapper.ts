import { hostFromServer } from "../config/paths";
import type { ArgoApplication } from "../types/argo";
import type { AppItem } from "../types/domain";

export function appToItem(a: ArgoApplication): AppItem {
  const last =
    a?.status?.history?.[0]?.deployedAt ??
    a?.status?.operationState?.finishedAt ??
    a?.status?.reconciledAt ??
    undefined;

  const dest = a?.spec?.destination ?? {};
  const name: string | undefined = dest.name; // prefer cluster name if present
  const serverUrl: string | undefined = dest.server;
  const id = name || hostFromServer(serverUrl) || undefined;
  const label = name || hostFromServer(serverUrl) || "unknown";

  return {
    name: a?.metadata?.name ?? "",
    sync: a?.status?.sync?.status ?? "Unknown",
    health: a?.status?.health?.status ?? "Unknown",
    lastSyncAt: last,
    project: a?.spec?.project,
    clusterId: id,
    clusterLabel: label,
    namespace: dest.namespace,
    appNamespace: a?.metadata?.namespace,
  };
}
