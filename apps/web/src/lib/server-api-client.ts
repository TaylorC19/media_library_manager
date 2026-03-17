import { headers } from "next/headers";

const API_BASE_URL =
  process.env.INTERNAL_API_BASE_URL ??
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  "http://localhost:4000";

function getApiUrl(path: string): string {
  return new URL(path, API_BASE_URL).toString();
}

export async function serverApiFetch(
  path: string,
  init: RequestInit = {}
): Promise<Response> {
  const requestHeaders = await headers();
  const forwardedHeaders = new Headers(init.headers);
  const cookieHeader = requestHeaders.get("cookie");

  if (cookieHeader) {
    forwardedHeaders.set("cookie", cookieHeader);
  }

  return fetch(getApiUrl(path), {
    ...init,
    cache: "no-store",
    headers: forwardedHeaders
  });
}
