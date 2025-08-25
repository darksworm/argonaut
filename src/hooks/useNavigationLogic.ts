import { useCallback, useEffect } from "react";
import { useAppState } from "../contexts/AppStateContext";
import { useVisibleItems } from "./useVisibleItems";

export const useNavigationLogic = () => {
  const { state, dispatch } = useAppState();
  const { visibleItems } = useVisibleItems();
  const { navigation } = state;

  // Keep selectedIdx within bounds when visibleItems change
  useEffect(() => {
    dispatch({
      type: "SET_SELECTED_IDX",
      payload: Math.min(
        navigation.selectedIdx,
        Math.max(0, visibleItems.length - 1),
      ),
    });
  }, [visibleItems.length, navigation.selectedIdx, dispatch]);

  // Drill down navigation logic
  const drillDown = useCallback(() => {
    const item = visibleItems[navigation.selectedIdx];
    if (!item) return;

    dispatch({ type: "RESET_NAVIGATION" });
    dispatch({
      type: "CLEAR_LOWER_LEVEL_SELECTIONS",
      payload: navigation.view,
    });

    const val = String(item);
    const next = new Set([val]);

    switch (navigation.view) {
      case "clusters":
        dispatch({ type: "SET_SCOPE_CLUSTERS", payload: next });
        dispatch({ type: "SET_VIEW", payload: "namespaces" });
        break;
      case "namespaces":
        dispatch({ type: "SET_SCOPE_NAMESPACES", payload: next });
        dispatch({ type: "SET_VIEW", payload: "projects" });
        break;
      case "projects":
        dispatch({ type: "SET_SCOPE_PROJECTS", payload: next });
        dispatch({ type: "SET_VIEW", payload: "apps" });
        break;
    }
  }, [visibleItems, navigation, dispatch]);

  // Toggle selection logic
  const toggleSelection = useCallback(() => {
    const item = visibleItems[navigation.selectedIdx];
    if (!item) return;

    const val = String(item);
    dispatch({
      type: "CLEAR_LOWER_LEVEL_SELECTIONS",
      payload: navigation.view,
    });

    switch (navigation.view) {
      case "clusters": {
        const next = state.selections.scopeClusters.has(val)
          ? new Set<string>()
          : new Set([val]);
        dispatch({ type: "SET_SCOPE_CLUSTERS", payload: next });
        break;
      }
      case "namespaces": {
        const next = state.selections.scopeNamespaces.has(val)
          ? new Set<string>()
          : new Set([val]);
        dispatch({ type: "SET_SCOPE_NAMESPACES", payload: next });
        break;
      }
      case "projects": {
        const next = state.selections.scopeProjects.has(val)
          ? new Set<string>()
          : new Set([val]);
        dispatch({ type: "SET_SCOPE_PROJECTS", payload: next });
        break;
      }
      case "apps": {
        const appName = (item as any).name;
        const next = new Set(state.selections.selectedApps);
        if (next.has(appName)) {
          next.delete(appName);
        } else {
          next.add(appName);
        }
        dispatch({ type: "SET_SELECTED_APPS", payload: next });
        break;
      }
    }
  }, [visibleItems, navigation, state.selections, dispatch]);

  return {
    drillDown,
    toggleSelection,
    visibleItems,
  };
};
