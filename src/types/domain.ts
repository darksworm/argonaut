// Domain types used across the app (stable)

export type AppItem = {
  name: string;
  sync: string;
  health: string;
  lastSyncAt?: string;   // ISO
  project?: string;
  clusterId?: string;    // destination.name OR server host
  clusterLabel?: string; // pretty label to show (name if present, else host)
  namespace?: string;
};

export type View = 'clusters' | 'namespaces' | 'projects' | 'apps';
export type Mode = 'normal' | 'loading' | 'search' | 'command' | 'help' | 'confirm-sync' | 'rollback' | 'rollback-confirm' | 'rollback-progress' | 'external' | 'resources';
