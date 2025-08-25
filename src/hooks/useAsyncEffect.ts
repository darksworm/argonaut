import { type DependencyList, useEffect } from "react";

/**
 * A wrapper around useEffect that allows async functions as the effect callback.
 * Handles cleanup properly by waiting for the async function to complete before
 * calling any returned cleanup function.
 */
export function useAsyncEffect(
  effect: () => Promise<undefined | (() => void)>,
  deps?: DependencyList,
) {
  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;

    (async () => {
      try {
        const result = await effect();
        if (!cancelled && typeof result === "function") {
          cleanup = result;
        }
      } catch (error) {
        // Let the effect handle its own errors
        console.error("useAsyncEffect error:", error);
      }
    })();

    return () => {
      cancelled = true;
      if (cleanup) {
        cleanup();
      }
    };
    // biome-ignore lint/correctness/useExhaustiveDependencies: deps is a parameter passed through
  }, deps);
}
