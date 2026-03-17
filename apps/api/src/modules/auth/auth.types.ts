import type { AuthUser } from "@media-library/types";

export interface RequestSessionContext {
  ipAddress: string | null;
  userAgent: string | null;
}

export interface AuthenticatedRequest {
  authUser?: AuthUser;
  cookies?: Record<string, string>;
  get?: (headerName: string) => string | undefined;
  headers: {
    cookie?: string;
  };
  ip?: string;
  method?: string;
  sessionId?: string;
  sessionToken?: string;
  socket: {
    remoteAddress?: string | null;
  };
}

export interface AuthResponse {
  clearCookie: (
    name: string,
    options?: Record<string, string | boolean | Date>
  ) => void;
  cookie: (
    name: string,
    value: string,
    options?: Record<string, string | boolean | Date>
  ) => void;
}
