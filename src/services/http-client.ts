import https from 'node:https';
import http from 'node:http';
import type {ServerConfig} from '../types/server';

export interface HttpClient {
  get(path: string, options?: RequestInit): Promise<any>;
  post(path: string, body?: any, options?: RequestInit): Promise<any>;
  put(path: string, body?: any, options?: RequestInit): Promise<any>;
  delete(path: string, options?: RequestInit): Promise<any>;
  stream(path: string, options?: RequestInit): Promise<Response>;
}

class ArgoHttpClient implements HttpClient {
  private baseUrl: string;
  private token: string;
  private agent?: https.Agent | http.Agent;

  constructor(serverConfig: ServerConfig, token: string) {
    this.baseUrl = serverConfig.baseUrl;
    this.token = token;

    // Create appropriate agent based on URL protocol and insecure flag
    const isHttps = this.baseUrl.startsWith('https://');
    
    if (isHttps) {
      this.agent = new https.Agent({
        rejectUnauthorized: !serverConfig.insecure
      });
    } else {
      this.agent = new http.Agent();
    }
  }

  private async request(path: string, options: RequestInit = {}): Promise<Response> {
    const url = this.baseUrl + path;
    
    const headers: Record<string, string> = {
      'Authorization': `Bearer ${this.token}`,
      'Content-Type': 'application/json',
      ...this.getHeaders(options.headers)
    };

    const requestOptions: RequestInit = {
      ...options,
      headers,
      // @ts-ignore - Node.js fetch supports agent
      agent: this.agent
    };

    const response = await fetch(url, requestOptions);
    
    if (!response.ok) {
      throw new Error(`${options.method || 'GET'} ${path} → ${response.status} ${response.statusText}`);
    }
    
    return response;
  }

  private getHeaders(headers?: HeadersInit): Record<string, string> {
    const result: Record<string, string> = {};
    
    if (!headers) return result;
    
    if (headers instanceof Headers) {
      headers.forEach((value, key) => {
        result[key] = value;
      });
    } else if (Array.isArray(headers)) {
      for (const [key, value] of headers) {
        result[key] = value;
      }
    } else {
      Object.assign(result, headers);
    }
    
    return result;
  }

  private async parseResponse(response: Response): Promise<any> {
    const contentType = response.headers.get('content-type');
    if (contentType?.includes('json')) {
      return response.json();
    }
    return response.text();
  }

  async get(path: string, options: RequestInit = {}): Promise<any> {
    const response = await this.request(path, { ...options, method: 'GET' });
    return this.parseResponse(response);
  }

  async post(path: string, body?: any, options: RequestInit = {}): Promise<any> {
    const requestOptions: RequestInit = {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined
    };
    
    const response = await this.request(path, requestOptions);
    return this.parseResponse(response);
  }

  async put(path: string, body?: any, options: RequestInit = {}): Promise<any> {
    const requestOptions: RequestInit = {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined
    };
    
    const response = await this.request(path, requestOptions);
    return this.parseResponse(response);
  }

  async delete(path: string, options: RequestInit = {}): Promise<any> {
    const response = await this.request(path, { ...options, method: 'DELETE' });
    return this.parseResponse(response);
  }

  async stream(path: string, options: RequestInit = {}): Promise<Response> {
    return this.request(path, { ...options, method: 'GET' });
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
      this.clients.set(key, new ArgoHttpClient(serverConfig, token));
    }
    
    return this.clients.get(key)!;
  }

  // Clear all clients (useful for logout/config changes)
  clearClients(): void {
    this.clients.clear();
  }

  // Remove specific client (useful when token expires)
  removeClient(serverConfig: ServerConfig, token: string): void {
    const key = `${serverConfig.baseUrl}:${token}:${serverConfig.insecure || false}`;
    this.clients.delete(key);
  }
}

// Export singleton instance
export const httpClientManager = HttpClientManager.getInstance();

// Convenience function to get a client
export function getHttpClient(serverConfig: ServerConfig, token: string): HttpClient {
  return httpClientManager.getClient(serverConfig, token);
}