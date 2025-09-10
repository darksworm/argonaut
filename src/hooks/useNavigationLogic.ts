import { useCallback, useEffect } from "react";
import { useAppState } from "../contexts/AppStateContext";
import {
  drillDown as drillDownFn,
  toggleSelection as toggleSelectionFn,
  validateBounds,
} from "../services/navigation-service";
import { useVisibleItems } from "./useVisibleItems";

export const useNavigationLogic = () => {
  const { state, dispatch } = useAppState();
  const { visibleItems } = useVisibleItems();
  const { navigation } = state;

  // Keep selectedIdx within bounds when visibleItems change
  useEffect(() => {
    const newIdx = validateBounds(navigation.selectedIdx, visibleItems.length);

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
    const result = drillDownFn(
      navigation.view,
      visibleItems[navigation.selectedIdx],
      visibleItems,
      navigation.selectedIdx,
    );

    if (!result) return;

    if (result.shouldResetNavigation) {
      dispatch({ type: "RESET_NAVIGATION" });
    }

    if (result.shouldClearLowerLevelSelections) {
      dispatch({
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: navigation.view,
      });
    }

    if (result.newView) {
      dispatch({ type: "SET_VIEW", payload: result.newView });
    }

    if (result.scopeClusters) {
      dispatch({ type: "SET_SCOPE_CLUSTERS", payload: result.scopeClusters });
    }

    if (result.scopeNamespaces) {
      dispatch({
        type: "SET_SCOPE_NAMESPACES",
        payload: result.scopeNamespaces,
      });
    }

    if (result.scopeProjects) {
      dispatch({ type: "SET_SCOPE_PROJECTS", payload: result.scopeProjects });
    }
  }, [visibleItems, navigation, dispatch]);

  // Toggle selection logic - only works in apps view
  const toggleSelection = useCallback(() => {
    const result = toggleSelectionFn(
      navigation.view,
      visibleItems[navigation.selectedIdx],
      visibleItems,
      navigation.selectedIdx,
      state.selections.selectedApps,
    );

    if (!result) return;

    dispatch({
      type: "CLEAR_LOWER_LEVEL_SELECTIONS",
      payload: navigation.view,
    });

    dispatch({ type: "SET_SELECTED_APPS", payload: result.selectedApps });
  }, [visibleItems, navigation, state.selections, dispatch]);

  return {
    drillDown,
    toggleSelection,
    visibleItems,
  };
};
