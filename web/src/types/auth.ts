export type UserRole = 'admin' | 'doctor' | 'secretary' | string;

export type AuthUser = {
  id: string;
  email: string;
  role: UserRole;
  professional_id?: string | null;
  active: boolean;
  created_at: string;
  updated_at: string;
};

export type LoginPayload = {
  email: string;
  password: string;
};

export type LoginResponse = {
  access_token: string;
  token_type: string;
  expires_at: string;
  user: AuthUser;
};

export type StoredSession = {
  accessToken: string;
  expiresAt: string;
  user: AuthUser;
};
