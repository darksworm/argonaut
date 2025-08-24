import {api} from './transport';
import type {Server} from '../types/server';
import { err, ok, ResultAsync } from 'neverthrow';

export type UserInfo = {
  iss?: string;
  sub?: string;
  email?: string;
  groups?: string[];
  loggedIn?: boolean;
};

export function getUserInfo(server: Server): ResultAsync<void, { message: string }> {
  return ResultAsync.fromPromise(
    api(server, '/api/v1/session/userinfo'),
    (error: any) => {
      if (error?.response?.data) {
        const errorData = error.response.data;
        const message = errorData.message || errorData.error || 'Unknown server error';
        return { message };
      }
      
      if (error?.message) {
        return { message: error.message };
      }
      
      return { message: `Failed to get user info - ${error}` };
    }
  ).map(() => undefined);
}
