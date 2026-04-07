import { request } from './http';
import type { AuthUser, LoginPayload, LoginResponse } from '../types/auth';

const DIRECTORY_API_BASE = '/directory-api';

export function login(payload: LoginPayload) {
  return request<LoginResponse>(DIRECTORY_API_BASE, '/auth/login', {
    method: 'POST',
    body: payload,
  });
}

export function getCurrentUser(accessToken: string) {
  return request<AuthUser>(DIRECTORY_API_BASE, '/auth/me', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
}
