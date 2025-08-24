import type { Server } from "../types/server";
import { api } from "./transport";

export async function syncApp(
  server: Server,
  name: string,
  opts?: { prune?: boolean; appNamespace?: string },
): Promise<void> {
  const params = new URLSearchParams();
  if (opts?.appNamespace) params.set("appNamespace", opts.appNamespace);

  await api(
    server,
    `/api/v1/applications/${encodeURIComponent(name)}/sync${params.toString() ? `?${params.toString()}` : ""}`,
    {
      method: "POST",
      body: JSON.stringify({
        prune: !!opts?.prune,
        appNamespace: opts?.appNamespace,
      }),
    },
  );
}
