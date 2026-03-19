export interface UserSettings {
  profileVisibility: "private";
  futureSocialEnabled: boolean;
}

export interface User {
  id: string;
  username: string;
  passwordHash: string;
  displayName?: string | null;
  settings: UserSettings;
  createdAt: string;
  updatedAt: string;
}

export interface AuthUser {
  id: string;
  username: string;
  displayName?: string | null;
}

export interface Session {
  id: string;
  userId: string;
  tokenHash: string;
  expiresAt: string;
  createdAt: string;
  lastUsedAt: string;
  userAgent?: string | null;
  ipAddress?: string | null;
}

export interface RegisterRequest {
  username: string;
  password: string;
  displayName?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface AuthSessionResponse {
  user: AuthUser;
}

export interface AuthLogoutResponse {
  success: true;
}
