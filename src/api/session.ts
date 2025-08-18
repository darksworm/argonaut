import {api} from './transport';
import type {Server} from '../types/server';

export type UserInfo = {
  iss?: string;
  sub?: string;
  email?: string;
  groups?: string[];
  loggedIn?: boolean;
};

export async function getUserInfo(server: Server): Promise<UserInfo> {
  // Throws on non-2xx; caller should handle and interpret as invalid token
  const data = await api(server, '/api/v1/session/userinfo');
  return data as UserInfo;
}
