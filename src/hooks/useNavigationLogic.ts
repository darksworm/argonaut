import { useCallback, useEffect } from "react";
import { useAppState } from "../contexts/AppStateContext";
import { useVisibleItems } from "./useVisibleItems";

export const useNavigationLogic = () => {
  const { state, dispatch } = useAppState();
  const { visibleItems } = useVisibleItems();
  const { navigation } = state;

  // Keep selectedIdx within bounds when visibleItems change
  useEffect(() => {
    const newIdx = Math.min(
      navigation.selectedIdx,
      Math.max(0, visibleItems.length - 1),
    );

    // Only update if the index actually needs to change
    if (newIdx !== navigation.selectedIdx) {
      dispatch({
        type: "SET_SELECTED_IDX",
        payload: newIdx,
      });
    }
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

  // Toggle selection logic - only works in apps view
  const toggleSelection = useCallback(() => {
    // Only allow toggle selection in apps view
    if (navigation.view !== "apps") {
      return;
    }

    const item = visibleItems[navigation.selectedIdx];
    if (!item) return;

    dispatch({
      type: "CLEAR_LOWER_LEVEL_SELECTIONS",
      payload: navigation.view,
    });

    const appName = (item as any).name;
    const next = new Set(state.selections.selectedApps);
    if (next.has(appName)) {
      next.delete(appName);
    } else {
      next.add(appName);
    }
    dispatch({ type: "SET_SELECTED_APPS", payload: next });
  }, [visibleItems, navigation, state.selections, dispatch]);

  return {
    drillDown,
    toggleSelection,
    visibleItems,
  };
};
