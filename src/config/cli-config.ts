import fs from "node:fs/promises";
import { err, ok, type Result } from "neverthrow";
import YAML from "yaml";
import type { ServerConfig } from "../types/server";
import { CONFIG_PATH, ensureHttps } from "./paths";

export type ArgoContext = { name: string; server: string; user: string };
export type ArgoServer = {
  server: string;
  "grpc-web"?: boolean;
  "grpc-web-root-path"?: string;
  insecure?: boolean;
  "plain-text"?: boolean;
};
export type ArgoUser = { name: string; "auth-token"?: string };
export type ArgoCLIConfig = {
  contexts?: ArgoContext[];
  servers?: ArgoServer[];
  users?: ArgoUser[];
  "current-context"?: string;
  "prompts-enabled"?: boolean;
};

export async function readCLIConfig(): Promise<
  Result<ArgoCLIConfig, { message: string }>
> {
  try {
    const txt = await fs.readFile(CONFIG_PATH, "utf8");
    return ok(YAML.parse(txt) as ArgoCLIConfig);
  } catch (e: unknown) {
    return err({
      message:
        (e as Error).message ?? `Couldn't parse config at ${CONFIG_PATH}`,
    });
  }
}

export function getCurrentServer(
  cfg: ArgoCLIConfig,
): Result<string, { message: string }> {
  const name = cfg["current-context"];
  const ctx = (cfg.contexts ?? []).find((c) => c.name === (name ?? ""));

  if (ctx?.server) {
    return ok(ctx?.server);
  } else {
    return err({
      message: `Could not find server details for current context ${name}`,
    });
  }
}

export function getCurrentServerConfig(
  cfg: ArgoCLIConfig,
  serverUrl: string,
): Result<ArgoServer, { message: string }> {
  const found = (cfg.servers ?? []).find((s) => s.server === serverUrl);
  if (found) {
    return ok(found);
  } else {
    return err({ message: `Could not find server with url ${serverUrl}` });
  }
}

export function getCurrentServerConfigObj(
  cfg: ArgoCLIConfig,
): Result<ServerConfig, { message: string }> {
  const serverUrl = getCurrentServer(cfg);
  if (serverUrl.isErr()) {
    return err(serverUrl.error);
  }

  const serverConfig = getCurrentServerConfig(cfg, serverUrl.value);

  if (serverConfig.isErr()) {
    return err(serverConfig.error);
  }

  return ok({
    baseUrl: ensureHttps(serverUrl.value, serverConfig.value["plain-text"]),
    insecure: serverConfig.value.insecure,
  });
}

export function tokenFromConfig(
  cfg: ArgoCLIConfig,
): Result<string, { message: string }> {
  const current = cfg["current-context"];

  if (!current) {
    return err({ message: "No current context set in ArgoCD config" });
  }

  if (!cfg.contexts?.length) {
    return err({ message: "No contexts found in ArgoCD config" });
  }

  const ctx = cfg.contexts.find((c) => c.name === current);
  if (!ctx) {
    return err({ message: `Context '${current}' not found in ArgoCD config` });
  }

  if (!ctx.user) {
    return err({ message: `No user specified for context '${current}'` });
  }

  if (!cfg.users?.length) {
    return err({ message: "No users found in ArgoCD config" });
  }

  const user = cfg.users.find((u) => u.name === ctx.user);
  if (!user) {
    return err({ message: `User '${ctx.user}' not found in ArgoCD config` });
  }

  const token = user["auth-token"];
  if (!token) {
    return err({
      message: `No auth token found for user '${ctx.user}'. Please run 'argocd login' to authenticate.`,
    });
  }

  return ok(token);
}
