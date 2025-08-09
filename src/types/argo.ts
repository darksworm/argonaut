// Minimal Argo API shapes (only fields we use)

export type ArgoDestination = {
  name?: string;
  namespace?: string;
  server?: string;
};

export type ArgoApplication = {
  metadata?: { name?: string };
  spec?: { project?: string; destination?: ArgoDestination };
  status?: {
    sync?: { status?: string };
    health?: { status?: string };
    history?: Array<{ deployedAt?: string }>;
    operationState?: { finishedAt?: string };
    reconciledAt?: string;
  };
};

export type ApplicationWatchEvent = {
  type?: string;
  application?: ArgoApplication;
};
