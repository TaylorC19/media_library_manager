import type { AuthSessionResponse, AuthUser } from "@media-library/types";
import { serverApiFetch } from "./server-api-client";
import { type AppLocale } from "../i18n/routing";
import { redirect } from "../i18n/navigation";

export async function getSessionUser(): Promise<AuthUser | null> {
  const response = await serverApiFetch("/auth/me");

  if (response.status === 401) {
    return null;
  }

  if (!response.ok) {
    throw new Error("Failed to load the current session.");
  }

  const payload = (await response.json()) as AuthSessionResponse;
  return payload.user;
}

export async function requireAuth(locale: AppLocale): Promise<AuthUser> {
  const user = await getSessionUser();

  if (!user) {
    return redirect({ href: "/login", locale });
  }

  return user;
}

export async function redirectIfAuthenticated(locale: AppLocale): Promise<void> {
  const user = await getSessionUser();

  if (user) {
    return redirect({ href: "/", locale });
  }
}
