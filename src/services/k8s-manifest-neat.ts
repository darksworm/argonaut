/**
 * Kubernetes Manifest Neat - A TypeScript implementation inspired by kubectl-neat
 * Strips unnecessary fields from Kubernetes manifests for cleaner diffs in ArgoCD
 */

export interface NeatConfig {
  /**
   * Fields to strip from metadata section
   */
  metadataFields: string[];

  /**
   * Annotation keys to strip from metadata.annotations
   */
  metadataAnnotations: string[];

  /**
   * Fields to strip from spec section (applied to all resource types)
   */
  specFields: string[];

  /**
   * Fields to strip from container specs
   */
  containerFields: string[];

  /**
   * ServiceAccount-related fields to strip
   */
  serviceAccountFields: string[];

  /**
   * Whether to remove the entire status section
   */
  removeStatus: boolean;

  /**
   * Whether to remove empty objects and arrays
   */
  removeEmptyObjects: boolean;

  /**
   * Whether to remove default values (basic implementation)
   */
  removeDefaults: boolean;
}

export const DEFAULT_NEAT_CONFIG: NeatConfig = {
  metadataFields: [
    "creationTimestamp",
    "resourceVersion",
    "uid",
    "generation",
    "managedFields",
    "finalizers",
    "selfLink",
  ],
  metadataAnnotations: [
    "kubectl.kubernetes.io/last-applied-configuration",
    "deployment.kubernetes.io/revision",
  ],
  specFields: [
    "nodeName", // Added by scheduler
    "clusterIP", // Added by service controller
    "clusterIPs", // Added by service controller
    "internalTrafficPolicy", // Default value
    "ipFamilies", // Default value
    "ipFamilyPolicy", // Default value
    "sessionAffinity", // Default "None"
    "progressDeadlineSeconds", // Default 600
    "revisionHistoryLimit", // Default 10
    "dnsPolicy", // Default "ClusterFirst"
    "restartPolicy", // Default "Always"
    "schedulerName", // Default "default-scheduler"
    "terminationGracePeriodSeconds", // Default 30
  ],
  containerFields: [
    "imagePullPolicy", // Default "IfNotPresent"
    "terminationMessagePath", // Default "/dev/termination-log"
    "terminationMessagePolicy", // Default "File"
  ],
  serviceAccountFields: [
    "serviceAccount", // Deprecated field
  ],
  removeStatus: true,
  removeEmptyObjects: true,
  removeDefaults: true,
};

/**
 * Deep clone an object to avoid modifying the original
 */
function deepClone<T>(obj: T): T {
  if (obj === null || typeof obj !== "object") return obj;
  if (obj instanceof Date) return new Date(obj.getTime()) as unknown as T;
  if (Array.isArray(obj))
    return obj.map((item) => deepClone(item)) as unknown as T;

  const cloned = {} as T;
  for (const key in obj) {
    if (Object.hasOwn(obj, key)) {
      cloned[key] = deepClone(obj[key]);
    }
  }
  return cloned;
}

/**
 * Remove specified fields from an object
 */
function removeFields(obj: any, fieldsToRemove: string[]): void {
  if (!obj || typeof obj !== "object") return;

  for (const field of fieldsToRemove) {
    delete obj[field];
  }
}

/**
 * Remove specified annotation keys from metadata.annotations
 */
function removeAnnotations(obj: any, annotationsToRemove: string[]): void {
  if (!obj?.metadata?.annotations) return;

  for (const annotation of annotationsToRemove) {
    delete obj.metadata.annotations[annotation];
  }
}

/**
 * Remove default-token-* volumes and volumeMounts (ServiceAccount cleanup)
 */
function removeServiceAccountDefaults(obj: any): void {
  if (!obj?.spec) return;

  // Remove default-token-* volumes
  if (obj.spec.volumes && Array.isArray(obj.spec.volumes)) {
    obj.spec.volumes = obj.spec.volumes.filter(
      (volume: any) =>
        !volume.name || !volume.name.startsWith("default-token-"),
    );
  }

  // Remove default-token-* volumeMounts from containers
  if (obj.spec.containers && Array.isArray(obj.spec.containers)) {
    obj.spec.containers.forEach((container: any) => {
      if (container.volumeMounts && Array.isArray(container.volumeMounts)) {
        container.volumeMounts = container.volumeMounts.filter(
          (mount: any) =>
            !mount.name || !mount.name.startsWith("default-token-"),
        );
      }
    });
  }

  // Remove deprecated serviceAccount field in favor of serviceAccountName
  removeFields(obj.spec, ["serviceAccount"]);
}

/**
 * Remove empty objects and arrays recursively
 */
function removeEmptyObjectsRecursive(obj: any): boolean {
  if (obj === null || obj === undefined) return true;

  if (Array.isArray(obj)) {
    // Remove empty items first
    for (let i = obj.length - 1; i >= 0; i--) {
      if (removeEmptyObjectsRecursive(obj[i])) {
        obj.splice(i, 1);
      }
    }
    return obj.length === 0;
  }

  if (typeof obj === "object") {
    const keys = Object.keys(obj);
    for (const key of keys) {
      if (removeEmptyObjectsRecursive(obj[key])) {
        delete obj[key];
      }
    }
    return Object.keys(obj).length === 0;
  }

  // For primitive values, only consider empty strings as "empty"
  // Note: kubectl-neat keeps empty strings as they're meaningful
  return false;
}

/**
 * Remove common default values based on Kubernetes resource type
 */
function removeCommonDefaults(obj: any): void {
  if (!obj?.spec) return;

  const kind = obj.kind;

  // Service defaults
  if (kind === "Service") {
    if (obj.spec.type === "ClusterIP") {
      delete obj.spec.type;
    }
    if (obj.spec.sessionAffinity === "None") {
      delete obj.spec.sessionAffinity;
    }

    // Remove default protocol from service ports
    if (obj.spec.ports && Array.isArray(obj.spec.ports)) {
      obj.spec.ports.forEach((port: any) => {
        if (port.protocol === "TCP") {
          delete port.protocol;
        }
      });
    }
  }

  // Deployment defaults
  if (kind === "Deployment") {
    if (obj.spec.progressDeadlineSeconds === 600) {
      delete obj.spec.progressDeadlineSeconds;
    }
    if (obj.spec.revisionHistoryLimit === 10) {
      delete obj.spec.revisionHistoryLimit;
    }
    if (
      obj.spec.strategy?.type === "RollingUpdate" &&
      obj.spec.strategy?.rollingUpdate?.maxSurge === "25%" &&
      obj.spec.strategy?.rollingUpdate?.maxUnavailable === "25%"
    ) {
      delete obj.spec.strategy;
    }
  }

  // Pod template defaults
  if (obj.spec.template?.spec) {
    const podSpec = obj.spec.template.spec;
    if (podSpec.dnsPolicy === "ClusterFirst") {
      delete podSpec.dnsPolicy;
    }
    if (podSpec.restartPolicy === "Always") {
      delete podSpec.restartPolicy;
    }
    if (podSpec.schedulerName === "default-scheduler") {
      delete podSpec.schedulerName;
    }
    if (podSpec.terminationGracePeriodSeconds === 30) {
      delete podSpec.terminationGracePeriodSeconds;
    }
    if (
      podSpec.securityContext &&
      Object.keys(podSpec.securityContext).length === 0
    ) {
      delete podSpec.securityContext;
    }

    // Helper function to clean container defaults
    const cleanContainerDefaults = (container: any) => {
      // Remove default imagePullPolicy values
      if (
        container.imagePullPolicy === "IfNotPresent" ||
        container.imagePullPolicy === "Always"
      ) {
        delete container.imagePullPolicy;
      }
      if (container.terminationMessagePath === "/dev/termination-log") {
        delete container.terminationMessagePath;
      }
      if (container.terminationMessagePolicy === "File") {
        delete container.terminationMessagePolicy;
      }
      if (
        container.resources &&
        Object.keys(container.resources).length === 0
      ) {
        delete container.resources;
      }

      // Remove default protocol from container ports
      if (container.ports && Array.isArray(container.ports)) {
        container.ports.forEach((port: any) => {
          if (port.protocol === "TCP") {
            delete port.protocol;
          }
        });
      }
    };

    // Clean regular containers
    if (podSpec.containers && Array.isArray(podSpec.containers)) {
      podSpec.containers.forEach(cleanContainerDefaults);
    }

    // Clean init containers
    if (podSpec.initContainers && Array.isArray(podSpec.initContainers)) {
      podSpec.initContainers.forEach(cleanContainerDefaults);
    }

    // Clean ephemeral containers
    if (
      podSpec.ephemeralContainers &&
      Array.isArray(podSpec.ephemeralContainers)
    ) {
      podSpec.ephemeralContainers.forEach(cleanContainerDefaults);
    }
  }
}

/**
 * Strip unnecessary fields from a Kubernetes manifest
 */
export function neatManifest(
  manifest: any,
  config: NeatConfig = DEFAULT_NEAT_CONFIG,
): any {
  // Deep clone to avoid modifying original
  const cleaned = deepClone(manifest);

  // Remove status section entirely
  if (config.removeStatus && cleaned.status) {
    delete cleaned.status;
  }

  // Clean metadata
  if (cleaned.metadata) {
    removeFields(cleaned.metadata, config.metadataFields);
    removeAnnotations(cleaned, config.metadataAnnotations);

    // Keep only essential metadata fields
    const essentialMetadata = {
      name: cleaned.metadata.name,
      namespace: cleaned.metadata.namespace,
      labels: cleaned.metadata.labels,
      annotations: cleaned.metadata.annotations,
    };

    // Remove undefined/null values
    Object.keys(essentialMetadata).forEach((key) => {
      if (
        essentialMetadata[key as keyof typeof essentialMetadata] ===
          undefined ||
        essentialMetadata[key as keyof typeof essentialMetadata] === null ||
        (typeof essentialMetadata[key as keyof typeof essentialMetadata] ===
          "object" &&
          Object.keys(
            essentialMetadata[key as keyof typeof essentialMetadata] || {},
          ).length === 0)
      ) {
        delete essentialMetadata[key as keyof typeof essentialMetadata];
      }
    });

    cleaned.metadata = essentialMetadata;
  }

  // Clean spec section
  if (cleaned.spec) {
    removeFields(cleaned.spec, config.specFields);

    // ServiceAccount cleanup for Pods
    if (cleaned.kind === "Pod" || cleaned.kind === "Deployment") {
      removeServiceAccountDefaults(cleaned);
    }
  }

  // Remove common default values
  if (config.removeDefaults) {
    removeCommonDefaults(cleaned);
  }

  // Remove empty objects and arrays
  if (config.removeEmptyObjects) {
    removeEmptyObjectsRecursive(cleaned);
  }

  return cleaned;
}

/**
 * Process ArgoCD diff data to extract and clean manifests
 */
export interface ArgoResourceDiff {
  kind: string;
  namespace: string;
  name: string;
  targetState?: string;
  liveState?: string;
  normalizedLiveState?: string;
  predictedLiveState?: string;
  resourceVersion?: string;
}

export interface CleanedDiffResult {
  target: any;
  live: any;
  name: string;
  kind: string;
  namespace: string;
}

/**
 * Clean ArgoCD resource diff data for better diff visualization
 */
export function cleanArgoResourceDiff(
  diff: ArgoResourceDiff,
  config: NeatConfig = DEFAULT_NEAT_CONFIG,
): CleanedDiffResult | null {
  try {
    const target = diff.targetState ? JSON.parse(diff.targetState) : null;
    const live = diff.liveState ? JSON.parse(diff.liveState) : null;

    if (!target && !live) {
      return null;
    }

    return {
      target: target ? neatManifest(target, config) : null,
      live: live ? neatManifest(live, config) : null,
      name: diff.name,
      kind: diff.kind,
      namespace: diff.namespace,
    };
  } catch (error) {
    console.error(`Failed to clean diff for ${diff.kind}/${diff.name}:`, error);
    return null;
  }
}

/**
 * Process multiple ArgoCD diffs and return cleaned results
 */
export function cleanArgoDiffs(
  diffs: ArgoResourceDiff[],
  config: NeatConfig = DEFAULT_NEAT_CONFIG,
): CleanedDiffResult[] {
  return diffs
    .map((diff) => cleanArgoResourceDiff(diff, config))
    .filter((result): result is CleanedDiffResult => result !== null);
}

/**
 * Create a custom configuration with additional fields to strip
 */
export function createNeatConfig(overrides: Partial<NeatConfig>): NeatConfig {
  return {
    ...DEFAULT_NEAT_CONFIG,
    ...overrides,
    // Merge arrays instead of replacing
    metadataFields: [
      ...DEFAULT_NEAT_CONFIG.metadataFields,
      ...(overrides.metadataFields || []),
    ],
    metadataAnnotations: [
      ...DEFAULT_NEAT_CONFIG.metadataAnnotations,
      ...(overrides.metadataAnnotations || []),
    ],
    specFields: [
      ...DEFAULT_NEAT_CONFIG.specFields,
      ...(overrides.specFields || []),
    ],
    containerFields: [
      ...DEFAULT_NEAT_CONFIG.containerFields,
      ...(overrides.containerFields || []),
    ],
    serviceAccountFields: [
      ...DEFAULT_NEAT_CONFIG.serviceAccountFields,
      ...(overrides.serviceAccountFields || []),
    ],
  };
}

export default {
  neatManifest,
  cleanArgoResourceDiff,
  cleanArgoDiffs,
  createNeatConfig,
  DEFAULT_NEAT_CONFIG,
};
