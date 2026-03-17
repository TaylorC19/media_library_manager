export interface AuthUser {
    id: string;
    username: string;
    displayName?: string | null;
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
