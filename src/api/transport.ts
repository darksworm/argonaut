import { getHttpClient } from "../services/http-client";
import type { Server } from "../types/server";

export async function api(
  server: Server,
  path: string,
  init?: RequestInit & { timeout?: number },
) {
  const client = getHttpClient(server.config, server.token);

  const method = init?.method?.toUpperCase() || "GET";
  const options = {
    signal: init?.signal || undefined,
    timeout: init?.timeout || undefined,
  };

  switch (method) {
    case "GET":
      return client.get(path, options);
    case "POST": {
      const body = init?.body ? JSON.parse(init.body as string) : undefined;
      return client.post(path, body, options);
    }
    case "PUT": {
      const putBody = init?.body ? JSON.parse(init.body as string) : undefined;
      return client.put(path, putBody, options);
    }
    case "DELETE":
      return client.delete(path, options);
    default:
      throw new Error(`Unsupported HTTP method: ${method}`);
  }
}
