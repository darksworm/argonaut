import { useMemo } from "react";
import { useAppState } from "../contexts/AppStateContext";
import type { AppItem } from "../types/domain";
import { uniqueSorted } from "../utils";

export const useVisibleItems = () => {
  const { state } = useAppState();
  const { apps, navigation, selections, ui } = state;
  const { view } = navigation;
  const { scopeClusters, scopeNamespaces, scopeProjects } = selections;
  const { searchQuery, activeFilter } = ui;

  // Calculate all clusters from apps
  const allClusters = useMemo(() => {
    const vals = apps.map((a) => a.clusterLabel || "").filter(Boolean);
    return uniqueSorted(vals);
  }, [apps]);

  // Filter apps by selected clusters
  const filteredByClusters = useMemo(() => {
    if (!scopeClusters.size) return apps;
    const allowed = new Set(scopeClusters);
    return apps.filter((a) => allowed.has(a.clusterLabel || ""));
  }, [apps, scopeClusters]);

  // Calculate all namespaces from filtered apps
  const allNamespaces = useMemo(() => {
    const nss = filteredByClusters
      .map((a) => a.namespace || "")
      .filter(Boolean);
    return uniqueSorted(nss);
  }, [filteredByClusters]);

  // Filter apps by selected namespaces
  const filteredByNs = useMemo(() => {
    if (!scopeNamespaces.size) return filteredByClusters;
    const allowed = new Set(scopeNamespaces);
    return filteredByClusters.filter((a) => allowed.has(a.namespace || ""));
  }, [filteredByClusters, scopeNamespaces]);

  // Calculate all projects from filtered apps
  const allProjects = useMemo(() => {
    const projs = filteredByNs.map((a) => a.project || "").filter(Boolean);
    return uniqueSorted(projs);
  }, [filteredByNs]);

  // Get final filtered apps by projects
  const finalApps = useMemo(() => {
    if (!scopeProjects.size) return filteredByNs;
    return filteredByNs.filter((a) => scopeProjects.has(a.project || ""));
  }, [filteredByNs, scopeProjects]);

  // Calculate visible items based on current view and filters
  const visibleItems = useMemo(() => {
    const filter = (
      state.mode === "search" ? searchQuery : activeFilter
    ).toLowerCase();
    let base: any[];

    switch (view) {
      case "clusters":
        base = allClusters;
        break;
      case "namespaces":
        base = allNamespaces;
        break;
      case "projects":
        base = allProjects;
        break;
      default:
        base = finalApps;
        break;
    }

    if (!filter) return base;

    if (view === "apps") {
      return base.filter(
        (a: AppItem) =>
          a.name.toLowerCase().includes(filter) ||
          (a.sync || "").toLowerCase().includes(filter) ||
          (a.health || "").toLowerCase().includes(filter) ||
          (a.namespace || "").toLowerCase().includes(filter) ||
          (a.project || "").toLowerCase().includes(filter),
      );
    } else {
      return base.filter((s) => String(s).toLowerCase().includes(filter));
    }
  }, [
    view,
    allClusters,
    allNamespaces,
    allProjects,
    finalApps,
    searchQuery,
    activeFilter,
    state.mode,
  ]);

  return {
    visibleItems,
    allClusters,
    allNamespaces,
    allProjects,
    filteredApps: finalApps,
  };
};
