import {api} from './transport';

export type UserInfo = {
  iss?: string;
  sub?: string;
  email?: string;
  groups?: string[];
  loggedIn?: boolean;
};

export type LoginResponse = {
  token: string;
};

export async function getUserInfo(server: string, token: string): Promise<UserInfo> {
  // Throws on non-2xx; caller should handle and interpret as invalid token
  const data = await api(server, token, '/api/v1/session/userinfo');
  return data as UserInfo;
}

export async function login(server: string, username: string, password: string): Promise<LoginResponse> {
  // Use fetch directly for login since we don't have a token yet
  const url = `${server}/api/v1/session`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username,
      password,
    }),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Login failed: ${error}`);
  }

  const data = await response.json();
  return data as LoginResponse;
}
