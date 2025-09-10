/**
 * Pure validation utility functions
 * Extracted for better organization and Go migration
 */

/**
 * Validate if string is a valid URL
 */
export function isValidUrl(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validate if string is a valid Kubernetes resource name
 */
export function isValidK8sName(name: string): boolean {
  // K8s names must be lowercase, alphanumeric, with dashes and dots
  const k8sNameRegex = /^[a-z0-9][a-z0-9\-.]*[a-z0-9]$|^[a-z0-9]$/;
  return k8sNameRegex.test(name) && name.length <= 253;
}

/**
 * Validate if string is a valid namespace name
 */
export function isValidNamespace(namespace: string): boolean {
  // Namespace names have stricter rules than general K8s names
  const namespaceRegex = /^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$/;
  return namespaceRegex.test(namespace) && namespace.length <= 63;
}

/**
 * Validate email format
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

/**
 * Validate if string is empty or only whitespace
 */
export function isEmpty(str: string): boolean {
  return !str || str.trim().length === 0;
}

/**
 * Validate if string contains only alphanumeric characters
 */
export function isAlphanumeric(str: string): boolean {
  const alphanumericRegex = /^[a-zA-Z0-9]+$/;
  return alphanumericRegex.test(str);
}

/**
 * Validate if value is a positive integer
 */
export function isPositiveInteger(value: any): boolean {
  const num = Number(value);
  return Number.isInteger(num) && num > 0;
}

/**
 * Validate if string matches semantic version format
 */
export function isValidSemVer(version: string): boolean {
  const semVerRegex =
    /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$/;
  return semVerRegex.test(version);
}

/**
 * Validate JSON string
 */
export function isValidJSON(jsonString: string): boolean {
  try {
    JSON.parse(jsonString);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validate YAML-like structure (basic check)
 */
export function isValidYAMLStructure(yamlString: string): boolean {
  // Basic YAML validation - check for common YAML patterns
  const lines = yamlString.split("\n");
  let hasValidStructure = false;

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed === "") continue;
    if (trimmed.startsWith("#")) continue; // Comments
    if (trimmed.includes(":")) hasValidStructure = true;
  }

  return hasValidStructure;
}
