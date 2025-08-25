import { ResultAsync } from "neverthrow";
import type { Server } from "../types/server";
import { api } from "./transport";

export function getUserInfo(
  server: Server,
  options?: { signal?: AbortSignal; timeout?: number },
): ResultAsync<void, { message: string }> {
  return ResultAsync.fromPromise(
    api(server, "/api/v1/session/userinfo", options),
    (error: any) => {
      if (error?.response?.data) {
        const errorData = error.response.data;
        const message =
          errorData.message || errorData.error || "Unknown server error";
        return { message };
      }

      if (error?.message) {
        return { message: error.message };
      }

      return { message: `Failed to get user info - ${error}` };
    },
  ).map(() => undefined);
}
