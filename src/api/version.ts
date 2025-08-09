import {api} from './transport';

export async function getApiVersion(server: string, token: string): Promise<string> {
  try {
    const data = await api(server, token, '/api/version');
    return (data as any)?.Version || 'Unknown';
  } catch {
    return 'Unknown';
  }
}
