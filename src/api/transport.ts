import type {Server} from '../types/server';
import {getHttpClient} from '../services/http-client';

export async function api(server: Server, path: string, init?: RequestInit) {
  const client = getHttpClient(server.config, server.token);
  
  const method = init?.method?.toUpperCase() || 'GET';
  
  switch (method) {
    case 'GET':
      return client.get(path, init);
    case 'POST':
      const body = init?.body ? JSON.parse(init.body as string) : undefined;
      return client.post(path, body, init);
    case 'PUT':
      const putBody = init?.body ? JSON.parse(init.body as string) : undefined;
      return client.put(path, putBody, init);
    case 'DELETE':
      return client.delete(path, init);
    default:
      throw new Error(`Unsupported HTTP method: ${method}`);
  }
}
