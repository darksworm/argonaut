import {api} from './transport';
import type {Server} from '../types/server';

export async function getApiVersion(server: Server): Promise<string> {
  try {
    const data = await api(server, '/api/version');
    return (data as any)?.Version || 'Unknown';
  } catch {
    return 'Unknown';
  }
}
