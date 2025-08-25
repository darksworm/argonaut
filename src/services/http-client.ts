import http from "node:http";
import https from "node:https";
import { URL } from "node:url";
import type { ServerConfig } from "../types/server";

export interface HttpClient {
  get(
    path: string,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any>;
  post(
    path: string,
    body?: any,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any>;
  put(
    path: string,
    body?: any,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any>;
  delete(
    path: string,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any>;
  stream(
    path: string,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<NodeJS.ReadableStream>;
  destroy(): void;
}

class ArgoHttpClient implements HttpClient {
  private baseUrl: string;
  private token: string;
  private agent?: https.Agent | http.Agent;
  private isHttps: boolean;

  constructor(serverConfig: ServerConfig, token: string) {
    this.baseUrl = serverConfig.baseUrl;
    this.token = token;
    this.isHttps = this.baseUrl.startsWith("https://");

    if (this.isHttps) {
      this.agent = new https.Agent({
        rejectUnauthorized: !serverConfig.insecure,
      });
    } else {
      this.agent = new http.Agent();
    }
  }

  private async request(
    path: string,
    method: string = "GET",
    body?: any,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any> {
    return new Promise((resolve, reject) => {
      const url = new URL(this.baseUrl + path);
      const requestModule = this.isHttps ? https : http;

      const requestOptions: http.RequestOptions = {
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        method,
        headers: {
          Authorization: `Bearer ${this.token}`,
          "Content-Type": "application/json",
          ...(body && {
            "Content-Length": Buffer.byteLength(JSON.stringify(body)),
          }),
        },
        agent: this.agent,
        timeout: options?.timeout || 10000, // 10 second default timeout
      };

      const req = requestModule.request(requestOptions, (res) => {
        let data = "";

        res.on("data", (chunk) => {
          data += chunk;
        });

        res.on("end", () => {
          if (res.statusCode && res.statusCode >= 400) {
            // Parse error response body for structured error information
            let errorData: any = null;
            try {
              const contentType = res.headers["content-type"];
              if (contentType?.includes("json") && data) {
                errorData = JSON.parse(data);
              }
            } catch {
              // If parsing fails, use raw data
              errorData = data;
            }

            // Create error with status code and parsed response data
            const error = new Error(
              `${method} ${path} → ${res.statusCode} ${res.statusMessage}`,
            );
            (error as any).status = res.statusCode;
            (error as any).statusText = res.statusMessage;
            (error as any).data = errorData;
            reject(error);
            return;
          }

          try {
            const contentType = res.headers["content-type"];
            if (contentType?.includes("json")) {
              resolve(JSON.parse(data));
            } else {
              resolve(data);
            }
          } catch {
            resolve(data);
          }
        });
      });

      req.on("error", reject);

      req.on("timeout", () => {
        req.destroy();
        reject(new Error("Request timeout"));
      });

      if (options?.signal) {
        options.signal.addEventListener("abort", () => {
          req.destroy();
          reject(new Error("Request aborted"));
        });
      }

      if (body) {
        req.write(JSON.stringify(body));
      }

      req.end();
    });
  }

  async get(
    path: string,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any> {
    return this.request(path, "GET", undefined, options);
  }

  async post(
    path: string,
    body?: any,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any> {
    return this.request(path, "POST", body, options);
  }

  async put(
    path: string,
    body?: any,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any> {
    return this.request(path, "PUT", body, options);
  }

  async delete(
    path: string,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<any> {
    return this.request(path, "DELETE", undefined, options);
  }

  async stream(
    path: string,
    options?: { signal?: AbortSignal; timeout?: number },
  ): Promise<NodeJS.ReadableStream> {
    return new Promise((resolve, reject) => {
      const url = new URL(this.baseUrl + path);
      const requestModule = this.isHttps ? https : http;

      const requestOptions: http.RequestOptions = {
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        method: "GET",
        headers: {
          Authorization: `Bearer ${this.token}`,
        },
        agent: this.agent,
        timeout: options?.timeout || 10000, // 10 second default timeout
      };

      const req = requestModule.request(requestOptions, (res) => {
        if (res.statusCode && res.statusCode >= 400) {
          // For streams, we'll collect the error body and reject with structured error
          let errorData = "";
          res.on("data", (chunk) => {
            errorData += chunk;
          });
          res.on("end", () => {
            let parsedError: any = null;
            try {
              const contentType = res.headers["content-type"];
              if (contentType?.includes("json") && errorData) {
                parsedError = JSON.parse(errorData);
              }
            } catch {
              parsedError = errorData;
            }

            const error = new Error(
              `GET ${path} → ${res.statusCode} ${res.statusMessage}`,
            );
            (error as any).status = res.statusCode;
            (error as any).statusText = res.statusMessage;
            (error as any).data = parsedError;
            reject(error);
          });
          return;
        }

        resolve(res);
      });

      req.on("error", reject);

      req.on("timeout", () => {
        req.destroy();
        reject(new Error("Request timeout"));
      });

      if (options?.signal) {
        options.signal.addEventListener("abort", () => {
          req.destroy();
          reject(new Error("Request aborted"));
        });
      }

      req.end();
    });
  }

  destroy(): void {
    if (this.agent) {
      this.agent.destroy();
    }
  }
}

// Multiton pattern - one client instance per server configuration
class HttpClientManager {
  private static instance: HttpClientManager;
  private clients: Map<string, HttpClient> = new Map();

  static getInstance(): HttpClientManager {
    if (!HttpClientManager.instance) {
      HttpClientManager.instance = new HttpClientManager();
    }
    return HttpClientManager.instance;
  }

  getClient(serverConfig: ServerConfig, token: string): HttpClient {
    const key = `${serverConfig.baseUrl}:${token}:${serverConfig.insecure || false}`;

    if (!this.clients.has(key)) {
      const client = new ArgoHttpClient(serverConfig, token);
      this.clients.set(key, client);
      return client;
    }

    const existingClient = this.clients.get(key);
    if (!existingClient) {
      throw new Error("HTTP client not found in cache");
    }
    return existingClient;
  }

  // Clear all clients (useful for logout/config changes)
  clearClients(): void {
    // Destroy all clients before clearing
    for (const client of this.clients.values()) {
      client.destroy();
    }
    this.clients.clear();
  }

  // Remove specific client (useful when token expires)
  removeClient(serverConfig: ServerConfig, token: string): void {
    const key = `${serverConfig.baseUrl}:${token}:${serverConfig.insecure || false}`;
    const client = this.clients.get(key);
    if (client) {
      client.destroy();
      this.clients.delete(key);
    }
  }
}

// Export singleton instance
export const httpClientManager = HttpClientManager.getInstance();

// Convenience function to get a client
export function getHttpClient(
  serverConfig: ServerConfig,
  token: string,
): HttpClient {
  return httpClientManager.getClient(serverConfig, token);
}
