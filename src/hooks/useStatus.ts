import { useMemo, useState } from "react";
import { createStatusService } from "../services/status-service";

export function useStatus(initialStatus: string = "") {
  const [status, setStatus] = useState(initialStatus);

  const statusLogger = useMemo(
    () => createStatusService((newStatus: string) => setStatus(newStatus)),
    [],
  );

  return [status, statusLogger] as const;
}
