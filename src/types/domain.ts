// Domain types used across the app (stable)

export type AppItem = {
  name: string;
  sync: string;
  health: string;
  lastSyncAt?: string; // ISO
  project?: string;
  clusterId?: string; // destination.name OR server host
  clusterLabel?: string; // pretty label to show (name if present, else host)
  namespace?: string; // destination namespace (where app deploys resources)
  appNamespace?: string; // application control plane namespace (where app CR exists)
};

export type View = "clusters" | "namespaces" | "projects" | "apps";
export type Mode =
  | "normal"
  | "loading"
  | "search"
  | "command"
  | "help"
  | "license"
  | "confirm-sync"
  | "rollback"
  | "external"
  | "resources"
  | "auth-required"
  | "rulerline"
  | "error"
  | "logs";
