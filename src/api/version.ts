import type { Server } from "../types/server";
import { api } from "./transport";

export async function getApiVersion(
  server: Server,
  options?: { signal?: AbortSignal; timeout?: number },
): Promise<string> {
  try {
    const data = await api(server, "/api/version", options);
    return (data as any)?.Version || "Unknown";
  } catch {
    return "Unknown";
  }
}
