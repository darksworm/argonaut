import {api} from './transport';

export type UserInfo = {
  iss?: string;
  sub?: string;
  email?: string;
  groups?: string[];
  loggedIn?: boolean;
};

export async function getUserInfo(baseUrl: string, token: string): Promise<UserInfo> {
  // Throws on non-2xx; caller should handle and interpret as invalid token
  const data = await api(baseUrl, token, '/api/v1/session/userinfo');
  return data as UserInfo;
}
