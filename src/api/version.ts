import {api} from './transport';

export async function getApiVersion(baseUrl: string, token: string): Promise<string> {
  try {
    const data = await api(baseUrl, token, '/api/version');
    return (data as any)?.Version || 'Unknown';
  } catch {
    return 'Unknown';
  }
}
