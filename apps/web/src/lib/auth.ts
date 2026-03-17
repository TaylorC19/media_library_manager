import type { AuthSessionResponse, AuthUser } from "@media-library/types";
import { redirect } from "next/navigation";
import { serverApiFetch } from "./server-api-client";

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

export async function requireAuth(): Promise<AuthUser> {
  const user = await getSessionUser();

  if (!user) {
    redirect("/login");
  }

  return user;
}

export async function redirectIfAuthenticated(): Promise<void> {
  const user = await getSessionUser();

  if (user) {
    redirect("/");
  }
}
