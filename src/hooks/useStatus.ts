import { useMemo, useState } from "react";
import { log } from "../services/logger";

export function useStatus(initialStatus: string = "") {
  const [status, setStatus] = useState(initialStatus);

  const statusLogger = useMemo(
    () => ({
      info: (message: string, context?: string) => {
        setStatus(message);
        log.info(message, context || "status");
      },
      warn: (message: string, context?: string) => {
        setStatus(message);
        log.warn(message, context || "status");
      },
      error: (message: string, context?: string) => {
        setStatus(message);
        log.error(message, context || "status");
      },
      debug: (message: string, context?: string) => {
        setStatus(message);
        log.debug(message, context || "status");
      },
      // For when you just want to set status without logging
      set: (message: string) => {
        setStatus(message);
      },
    }),
    [],
  ); // Empty deps - stable object

  return [status, statusLogger] as const;
}
