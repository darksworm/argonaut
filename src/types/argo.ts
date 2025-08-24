// Minimal Argo API shapes (only fields we use)

export type ArgoDestination = {
  name?: string;
  namespace?: string;
  server?: string;
};

export type ArgoApplication = {
  metadata?: { name?: string; namespace?: string };
  spec?: { project?: string; destination?: ArgoDestination };
  status?: {
    sync?: { status?: string; revision?: string };
    health?: { status?: string };
    history?: Array<{
      id?: number;
      revision?: string;
      deployedAt?: string;
      source?: any;
    }>;
    operationState?: { finishedAt?: string; phase?: string; message?: string };
    reconciledAt?: string;
  };
};

export type ApplicationWatchEvent = {
  type?: string;
  application?: ArgoApplication;
};
