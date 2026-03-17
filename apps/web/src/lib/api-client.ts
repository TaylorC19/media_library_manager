const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:4000";

function getApiUrl(path: string): string {
  return new URL(path, API_BASE_URL).toString();
}

export async function browserApiFetch(
  path: string,
  init: RequestInit = {}
): Promise<Response> {
  return fetch(getApiUrl(path), {
    ...init,
    credentials: "include"
  });
}
