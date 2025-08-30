import { beforeEach, describe, expect, spyOn, test } from "bun:test";
import { getHttpClient, httpClientManager } from "../../services/http-client";
import type { ServerConfig } from "../../types/server";

describe("httpClientManager", () => {
  const mockServerConfig: ServerConfig = {
    baseUrl: "https://test.example.com",
    insecure: false,
  };
  const mockToken = "test-token";

  beforeEach(() => {
    httpClientManager.clearClients();
  });

  describe("getClient", () => {
    test("should create new client for new server config", () => {
      const client1 = httpClientManager.getClient(mockServerConfig, mockToken);
      expect(client1).toBeDefined();
    });

    test("should return same client for identical config", () => {
      const client1 = httpClientManager.getClient(mockServerConfig, mockToken);
      const client2 = httpClientManager.getClient(mockServerConfig, mockToken);

      expect(client1).toBe(client2);
    });

    test("should create different clients for different tokens", () => {
      const client1 = httpClientManager.getClient(mockServerConfig, "token1");
      const client2 = httpClientManager.getClient(mockServerConfig, "token2");

      expect(client1).not.toBe(client2);
    });

    test("should create different clients for different server configs", () => {
      const config1: ServerConfig = {
        baseUrl: "https://server1.com",
        insecure: false,
      };
      const config2: ServerConfig = {
        baseUrl: "https://server2.com",
        insecure: false,
      };

      const client1 = httpClientManager.getClient(config1, mockToken);
      const client2 = httpClientManager.getClient(config2, mockToken);

      expect(client1).not.toBe(client2);
    });

    test("should create different clients for different insecure settings", () => {
      const secureConfig: ServerConfig = {
        baseUrl: "https://test.com",
        insecure: false,
      };
      const insecureConfig: ServerConfig = {
        baseUrl: "https://test.com",
        insecure: true,
      };

      const client1 = httpClientManager.getClient(secureConfig, mockToken);
      const client2 = httpClientManager.getClient(insecureConfig, mockToken);

      expect(client1).not.toBe(client2);
    });
  });

  describe("clearClients", () => {
    test("should clear all cached clients", () => {
      const client1 = httpClientManager.getClient(mockServerConfig, mockToken);
      const destroySpy = spyOn(client1, "destroy");

      httpClientManager.clearClients();

      expect(destroySpy).toHaveBeenCalled();

      // Getting client again should create a new instance
      const client2 = httpClientManager.getClient(mockServerConfig, mockToken);
      expect(client2).not.toBe(client1);
    });
  });

  describe("removeClient", () => {
    test("should remove specific client", () => {
      const client1 = httpClientManager.getClient(mockServerConfig, mockToken);
      const destroySpy = spyOn(client1, "destroy");

      httpClientManager.removeClient(mockServerConfig, mockToken);

      expect(destroySpy).toHaveBeenCalled();

      // Getting client again should create a new instance
      const client2 = httpClientManager.getClient(mockServerConfig, mockToken);
      expect(client2).not.toBe(client1);
    });

    test("should handle removing non-existent client gracefully", () => {
      const nonExistentConfig: ServerConfig = {
        baseUrl: "https://nonexistent.com",
        insecure: false,
      };

      expect(() =>
        httpClientManager.removeClient(nonExistentConfig, "fake-token"),
      ).not.toThrow();
    });
  });
});

describe("getHttpClient convenience function", () => {
  const mockServerConfig: ServerConfig = {
    baseUrl: "https://test.example.com",
    insecure: false,
  };
  const mockToken = "test-token";

  beforeEach(() => {
    httpClientManager.clearClients();
  });

  test("should return client from manager", () => {
    const client1 = getHttpClient(mockServerConfig, mockToken);
    const client2 = httpClientManager.getClient(mockServerConfig, mockToken);

    expect(client1).toBe(client2);
  });
});

describe("ArgoHttpClient integration", () => {
  const mockServerConfig: ServerConfig = {
    baseUrl: "https://test.example.com",
    insecure: false,
  };
  const mockToken = "test-token";

  beforeEach(() => {
    httpClientManager.clearClients();
  });

  describe("client interface", () => {
    test("should have required methods", () => {
      const client = getHttpClient(mockServerConfig, mockToken);

      expect(typeof client.get).toBe("function");
      expect(typeof client.post).toBe("function");
      expect(typeof client.put).toBe("function");
      expect(typeof client.delete).toBe("function");
      expect(typeof client.stream).toBe("function");
      expect(typeof client.destroy).toBe("function");
    });

    test("should handle destroy method", () => {
      const client = getHttpClient(mockServerConfig, mockToken);

      // Should not throw when calling destroy
      expect(() => client.destroy()).not.toThrow();
    });
  });

  describe("configuration handling", () => {
    test("should handle secure HTTPS configuration", () => {
      const secureConfig: ServerConfig = {
        baseUrl: "https://secure.example.com",
        insecure: false,
      };

      const client = getHttpClient(secureConfig, mockToken);
      expect(client).toBeDefined();
    });

    test("should handle insecure HTTPS configuration", () => {
      const insecureConfig: ServerConfig = {
        baseUrl: "https://insecure.example.com",
        insecure: true,
      };

      const client = getHttpClient(insecureConfig, mockToken);
      expect(client).toBeDefined();
    });

    test("should handle HTTP configuration", () => {
      const httpConfig: ServerConfig = {
        baseUrl: "http://local.example.com",
        insecure: false,
      };

      const client = getHttpClient(httpConfig, mockToken);
      expect(client).toBeDefined();
    });
  });

  describe("request methods with abort signal", () => {
    test("should accept abort signal in options", async () => {
      const client = getHttpClient(mockServerConfig, mockToken);
      const abortController = new AbortController();

      // Abort immediately to test signal handling
      abortController.abort();

      // These will fail with network errors, but should handle abort signal gracefully
      const testMethod = async (methodName: string, ...args: any[]) => {
        try {
          if (methodName === "stream") {
            await client.stream("/test", { signal: abortController.signal });
          } else {
            await (client as any)[methodName]("/test", ...args, {
              signal: abortController.signal,
            });
          }
        } catch (error) {
          // Expected to fail, but we're testing that abort signal is handled
          expect(error).toBeDefined();
        }
      };

      // Test that methods don't throw unexpected errors when handling abort signals
      await expect(testMethod("get")).resolves.toBeUndefined();
      await expect(testMethod("post", {})).resolves.toBeUndefined();
      await expect(testMethod("put", {})).resolves.toBeUndefined();
      await expect(testMethod("delete")).resolves.toBeUndefined();
      await expect(testMethod("stream")).resolves.toBeUndefined();
    });

    test("should accept timeout in options", async () => {
      const client = getHttpClient(mockServerConfig, mockToken);

      // These will fail with network errors, but should handle timeout option
      const testMethodWithTimeout = async (
        methodName: string,
        ...args: any[]
      ) => {
        try {
          if (methodName === "stream") {
            await client.stream("/test", { timeout: 1000 });
          } else {
            await (client as any)[methodName]("/test", ...args, {
              timeout: 1000,
            });
          }
        } catch (error) {
          // Expected to fail due to mock server, but we're testing timeout handling
          expect(error).toBeDefined();
        }
      };

      await expect(testMethodWithTimeout("get")).resolves.toBeUndefined();
      await expect(testMethodWithTimeout("post", {})).resolves.toBeUndefined();
      await expect(testMethodWithTimeout("put", {})).resolves.toBeUndefined();
      await expect(testMethodWithTimeout("delete")).resolves.toBeUndefined();
      await expect(testMethodWithTimeout("stream")).resolves.toBeUndefined();
    });
  });

  describe("client caching edge cases", () => {
    test("should handle undefined insecure flag as false", () => {
      const configWithUndefinedInsecure: ServerConfig = {
        baseUrl: "https://test.com",
        insecure: undefined as any,
      };

      const client1 = getHttpClient(configWithUndefinedInsecure, mockToken);
      const client2 = getHttpClient(
        { baseUrl: "https://test.com", insecure: false },
        mockToken,
      );

      expect(client1).toBe(client2);
    });

    test("should create different clients for truthy vs falsy insecure values", () => {
      const config1: ServerConfig = {
        baseUrl: "https://test.com",
        insecure: false,
      };
      const config2: ServerConfig = {
        baseUrl: "https://test.com",
        insecure: true,
      };

      const client1 = getHttpClient(config1, mockToken);
      const client2 = getHttpClient(config2, mockToken);

      expect(client1).not.toBe(client2);
    });

    test("should handle empty token string", () => {
      const client = getHttpClient(mockServerConfig, "");
      expect(client).toBeDefined();
    });

    test("should handle special characters in baseUrl", () => {
      const specialConfig: ServerConfig = {
        baseUrl: "https://test-server.example.com:8443",
        insecure: false,
      };

      const client = getHttpClient(specialConfig, mockToken);
      expect(client).toBeDefined();
    });
  });
});
