/**
 * Pure scope handling utility functions
 * Extracted for better organization and Go migration
 */

/**
 * Get unique sorted array from any array
 */
export function uniqueSorted<T>(arr: T[]): T[] {
  return Array.from(new Set(arr)).sort((a: any, b: any) =>
    `${a}`.localeCompare(`${b}`),
  );
}

/**
 * Format scope display with truncation
 */
export function fmtScope(set: Set<string>, max = 2): string {
  if (!set.size) return "—";
  const arr = Array.from(set);
  if (arr.length <= max) return arr.join(",");
  return `${arr.slice(0, max).join(",")} (+${arr.length - max})`;
}

/**
 * Check if two sets are equal
 */
export function setsEqual<T>(set1: Set<T>, set2: Set<T>): boolean {
  if (set1.size !== set2.size) return false;
  for (const item of set1) {
    if (!set2.has(item)) return false;
  }
  return true;
}

/**
 * Create intersection of multiple sets
 */
export function setIntersection<T>(...sets: Set<T>[]): Set<T> {
  if (sets.length === 0) return new Set();
  if (sets.length === 1) return new Set(sets[0]);

  const result = new Set(sets[0]);
  for (let i = 1; i < sets.length; i++) {
    for (const item of result) {
      if (!sets[i].has(item)) {
        result.delete(item);
      }
    }
  }
  return result;
}

/**
 * Create union of multiple sets
 */
export function setUnion<T>(...sets: Set<T>[]): Set<T> {
  const result = new Set<T>();
  for (const set of sets) {
    for (const item of set) {
      result.add(item);
    }
  }
  return result;
}

/**
 * Create difference between two sets (items in set1 but not in set2)
 */
export function setDifference<T>(set1: Set<T>, set2: Set<T>): Set<T> {
  const result = new Set<T>();
  for (const item of set1) {
    if (!set2.has(item)) {
      result.add(item);
    }
  }
  return result;
}

/**
 * Check if set1 is a subset of set2
 */
export function isSubset<T>(set1: Set<T>, set2: Set<T>): boolean {
  for (const item of set1) {
    if (!set2.has(item)) {
      return false;
    }
  }
  return true;
}

/**
 * Convert comma-separated string to Set
 */
export function stringToSet(str: string): Set<string> {
  if (!str.trim()) return new Set();
  return new Set(
    str
      .split(",")
      .map((s) => s.trim())
      .filter((s) => s.length > 0),
  );
}

/**
 * Convert Set to comma-separated string
 */
export function setToString(set: Set<string>): string {
  return Array.from(set).sort().join(",");
}

/**
 * Filter items based on scope selections
 */
export function filterByScopes<T extends { [K in keyof T]: any }>(
  items: T[],
  scopeFilters: {
    clusters?: Set<string>;
    namespaces?: Set<string>;
    projects?: Set<string>;
  },
  getCluster: (item: T) => string | undefined,
  getNamespace: (item: T) => string | undefined,
  getProject: (item: T) => string | undefined,
): T[] {
  return items.filter((item) => {
    // If no filters are set, include all items
    const hasFilters =
      (scopeFilters.clusters?.size || 0) > 0 ||
      (scopeFilters.namespaces?.size || 0) > 0 ||
      (scopeFilters.projects?.size || 0) > 0;

    if (!hasFilters) return true;

    // Check cluster filter
    if (scopeFilters.clusters && scopeFilters.clusters.size > 0) {
      const cluster = getCluster(item);
      if (!cluster || !scopeFilters.clusters.has(cluster)) {
        return false;
      }
    }

    // Check namespace filter
    if (scopeFilters.namespaces && scopeFilters.namespaces.size > 0) {
      const namespace = getNamespace(item);
      if (!namespace || !scopeFilters.namespaces.has(namespace)) {
        return false;
      }
    }

    // Check project filter
    if (scopeFilters.projects && scopeFilters.projects.size > 0) {
      const project = getProject(item);
      if (!project || !scopeFilters.projects.has(project)) {
        return false;
      }
    }

    return true;
  });
}

/**
 * Get all unique values from items for a specific field
 */
export function getUniqueValues<T>(
  items: T[],
  getValue: (item: T) => string | undefined,
): string[] {
  const values = new Set<string>();
  for (const item of items) {
    const value = getValue(item);
    if (value?.trim()) {
      values.add(value);
    }
  }
  return uniqueSorted(Array.from(values));
}
