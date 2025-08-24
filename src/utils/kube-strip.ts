/**
 * Kubernetes resource field stripper
 * 
 * Based on kubectl-neat approach, removes system-added metadata,
 * status fields, and other non-essential data to produce clean
 * diffs that focus on meaningful changes.
 */

type KubeResource = { [key: string]: any };

interface StripOptions {
  /**
   * Remove all status fields (default: true)
   */
  stripStatus?: boolean;
  
  /**
   * Remove system metadata fields (default: true)
   */
  stripMetadata?: boolean;
  
  /**
   * Remove spec fields added by scheduler/controllers (default: true)
   */
  stripSchedulerFields?: boolean;
  
  /**
   * Remove default service account tokens (default: true)
   */
  stripDefaultTokens?: boolean;
  
  /**
   * Remove empty objects/arrays after stripping (default: true)
   */
  pruneEmpty?: boolean;
}

const DEFAULT_OPTIONS: Required<StripOptions> = {
  stripStatus: true,
  stripMetadata: true,
  stripSchedulerFields: true,
  stripDefaultTokens: true,
  pruneEmpty: true,
};

/**
 * System metadata fields to remove
 */
const SYSTEM_METADATA_FIELDS = [
  'managedFields',
  'resourceVersion',
  'uid',
  'selfLink',
  'generation',
  'creationTimestamp',
  'finalizers',
  'ownerReferences',
  'deletionTimestamp',
  'deletionGracePeriodSeconds',
];

/**
 * System annotations to remove
 */
const SYSTEM_ANNOTATIONS = [
  'kubectl.kubernetes.io/last-applied-configuration',
  'control-plane.alpha.kubernetes.io/leader',
  'deployment.kubernetes.io/revision',
  'pv.kubernetes.io/bind-completed',
  'pv.kubernetes.io/bound-by-controller',
  'volume.beta.kubernetes.io/storage-provisioner',
  'endpoints.kubernetes.io/last-change-trigger-time',
];

/**
 * Spec fields added by scheduler/controllers
 */
const SCHEDULER_FIELDS = [
  'nodeName',
  'serviceAccount', // deprecated, use serviceAccountName
  'clusterIP',
  'clusterIPs',
  'externalTrafficPolicy',
  'healthCheckNodePort',
  'internalTrafficPolicy',
  'ipFamilies',
  'ipFamilyPolicy',
  'sessionAffinity',
  'type', // for services when it's ClusterIP (default)
];

/**
 * Default values to remove when they match expected defaults
 */
const DEFAULT_VALUES = {
  'spec.restartPolicy': 'Always',
  'spec.terminationGracePeriodSeconds': 30,
  'spec.dnsPolicy': 'ClusterFirst',
  'spec.schedulerName': 'default-scheduler',
  'spec.serviceAccountName': 'default',
  'spec.securityContext': {},
  'spec.strategy.type': 'RollingUpdate',
  'spec.progressDeadlineSeconds': 600,
  'spec.revisionHistoryLimit': 10,
  'spec.type': 'ClusterIP', // for Services
};

/**
 * Check if a volume is a default service account token
 */
function isDefaultTokenVolume(volume: any): boolean {
  return volume && 
    typeof volume.name === 'string' && 
    (volume.name.includes('default-token') || 
     volume.name.includes('kube-api-access'));
}

/**
 * Check if a container volume mount is for default token
 */
function isDefaultTokenMount(mount: any): boolean {
  return mount && 
    typeof mount.name === 'string' && 
    (mount.name.includes('default-token') || 
     mount.name.includes('kube-api-access'));
}

/**
 * Remove empty objects and arrays recursively
 */
function pruneEmpty(obj: any): any {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  
  if (Array.isArray(obj)) {
    const filtered = obj.map(pruneEmpty).filter(item => {
      if (item === null || item === undefined) return false;
      if (Array.isArray(item)) return item.length > 0;
      if (typeof item === 'object') return Object.keys(item).length > 0;
      return true;
    });
    return filtered.length > 0 ? filtered : undefined;
  }
  
  const result: KubeResource = {};
  for (const [key, value] of Object.entries(obj)) {
    const cleaned = pruneEmpty(value);
    if (cleaned !== undefined) {
      if (Array.isArray(cleaned) && cleaned.length === 0) continue;
      if (typeof cleaned === 'object' && cleaned !== null && Object.keys(cleaned).length === 0) continue;
      result[key] = cleaned;
    }
  }
  
  return Object.keys(result).length > 0 ? result : undefined;
}

/**
 * Strip system annotations, keeping only user-defined ones
 */
function stripAnnotations(annotations: KubeResource): KubeResource | undefined {
  if (!annotations || typeof annotations !== 'object') return undefined;
  
  const cleaned: KubeResource = {};
  for (const [key, value] of Object.entries(annotations)) {
    // Skip system annotations
    if (SYSTEM_ANNOTATIONS.some(pattern => key.includes(pattern))) continue;
    
    // Skip ArgoCD tracking annotations (they're system-generated)
    if (key.includes('argocd.argoproj.io/')) continue;
    
    // Skip Kubernetes system annotations
    if (key.startsWith('kubernetes.io/') || key.startsWith('k8s.io/')) continue;
    
    cleaned[key] = value;
  }
  
  return Object.keys(cleaned).length > 0 ? cleaned : undefined;
}

/**
 * Strip metadata fields
 */
function stripMetadataFields(metadata: KubeResource, options: Required<StripOptions>): KubeResource | undefined {
  if (!metadata || typeof metadata !== 'object') return undefined;
  
  const cleaned: KubeResource = {};
  
  for (const [key, value] of Object.entries(metadata)) {
    // Remove system metadata fields
    if (options.stripMetadata && SYSTEM_METADATA_FIELDS.includes(key)) continue;
    
    // Handle annotations specially
    if (key === 'annotations') {
      const cleanedAnnotations = stripAnnotations(value);
      if (cleanedAnnotations) cleaned.annotations = cleanedAnnotations;
      continue;
    }
    
    // Keep other metadata (name, namespace, labels, etc.)
    cleaned[key] = stripKubeFields(value, options);
  }
  
  return Object.keys(cleaned).length > 0 ? cleaned : undefined;
}

/**
 * Strip spec fields added by controllers/scheduler
 */
function stripSpecFields(spec: KubeResource, options: Required<StripOptions>): KubeResource | undefined {
  if (!spec || typeof spec !== 'object') return undefined;
  
  const cleaned: KubeResource = {};
  
  for (const [key, value] of Object.entries(spec)) {
    // Remove scheduler/controller fields
    if (options.stripSchedulerFields && SCHEDULER_FIELDS.includes(key)) continue;
    
    // Handle volumes specially to remove default tokens
    if (key === 'volumes' && options.stripDefaultTokens && Array.isArray(value)) {
      const cleanedVolumes = value
        .filter(vol => !isDefaultTokenVolume(vol))
        .map(vol => stripKubeFields(vol, options));
      if (cleanedVolumes.length > 0) cleaned.volumes = cleanedVolumes;
      continue;
    }
    
    // Handle containers to remove default token mounts
    if (key === 'containers' && options.stripDefaultTokens && Array.isArray(value)) {
      const cleanedContainers = value.map(container => {
        if (!container || typeof container !== 'object') return container;
        
        const cleanedContainer = { ...container };
        if (Array.isArray(container.volumeMounts)) {
          const cleanedMounts = container.volumeMounts.filter(mount => !isDefaultTokenMount(mount));
          if (cleanedMounts.length > 0) {
            cleanedContainer.volumeMounts = cleanedMounts;
          } else {
            delete cleanedContainer.volumeMounts;
          }
        }
        
        return stripKubeFields(cleanedContainer, options);
      });
      cleaned.containers = cleanedContainers;
      continue;
    }
    
    // Handle template (for Deployments, etc.)
    if (key === 'template' && value && typeof value === 'object') {
      cleaned.template = stripKubeFields(value, options);
      continue;
    }
    
    // Check for default values to remove
    const defaultKey = `spec.${key}`;
    if (defaultKey in DEFAULT_VALUES && 
        JSON.stringify(value) === JSON.stringify(DEFAULT_VALUES[defaultKey as keyof typeof DEFAULT_VALUES])) {
      continue;
    }
    
    cleaned[key] = stripKubeFields(value, options);
  }
  
  return Object.keys(cleaned).length > 0 ? cleaned : undefined;
}

/**
 * Recursively strip Kubernetes fields from a resource
 */
function stripKubeFields(obj: any, options: Required<StripOptions>): any {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  
  if (Array.isArray(obj)) {
    return obj.map(item => stripKubeFields(item, options));
  }
  
  const result: KubeResource = {};
  
  for (const [key, value] of Object.entries(obj)) {
    // Remove status entirely
    if (key === 'status' && options.stripStatus) continue;
    
    // Handle metadata specially
    if (key === 'metadata') {
      const cleanedMetadata = stripMetadataFields(value, options);
      if (cleanedMetadata) result.metadata = cleanedMetadata;
      continue;
    }
    
    // Handle spec specially
    if (key === 'spec') {
      const cleanedSpec = stripSpecFields(value, options);
      if (cleanedSpec) result.spec = cleanedSpec;
      continue;
    }
    
    // Recurse for other fields
    result[key] = stripKubeFields(value, options);
  }
  
  return options.pruneEmpty ? pruneEmpty(result) : result;
}

/**
 * Strip non-essential fields from Kubernetes resources
 */
export function stripKubernetesFields(resource: KubeResource, options: StripOptions = {}): KubeResource {
  const opts = { ...DEFAULT_OPTIONS, ...options };
  return stripKubeFields(resource, opts);
}

/**
 * Strip fields from ArgoCD diff data
 */
export function stripArgoCDDiff(diffData: Array<{
  kind?: string;
  namespace?: string;
  name?: string;
  targetState?: string;
  liveState?: string;
  normalizedLiveState?: string;
  predictedLiveState?: string;
  [key: string]: any;
}>, options: StripOptions = {}): any[] {
  return diffData.map(item => {
    const result: any = { ...item };
    
    // Strip each state field
    for (const stateField of ['targetState', 'liveState', 'normalizedLiveState', 'predictedLiveState']) {
      if (result[stateField]) {
        try {
          const parsed = JSON.parse(result[stateField]);
          const stripped = stripKubernetesFields(parsed, options);
          result[stateField] = JSON.stringify(stripped);
        } catch (e) {
          // Keep original if parsing fails
        }
      }
    }
    
    return result;
  });
}

/**
 * Create a clean YAML representation for diffing
 * Note: Import YAML in your calling code and pass the result to stripKubernetesFields
 * This function is kept for convenience but importing YAML directly is recommended
 */
export function toCleanYaml(resource: KubeResource, options: StripOptions = {}): string {
  const stripped = stripKubernetesFields(resource, options);
  return JSON.stringify(stripped, null, 2); // Return JSON - convert to YAML in calling code
}