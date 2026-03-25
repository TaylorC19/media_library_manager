export interface ApiErrorPayload {
  code?: string;
  error?: string;
  message?: string | string[];
  statusCode?: number;
}

type Translate = (key: string) => string;

const messageKeyByMessage = new Map<string, string>([
  ["Authentication required.", "sessionRequired"],
  ["Failed to search providers.", "searchLoadFailed"],
  ["Invalid username or password.", "invalidCredentials"],
  ["Library entry not found.", "libraryEntryNotFound"],
  ["Media record not found.", "mediaRecordNotFound"],
  [
    "That item is already saved in this bucket with the same format.",
    "duplicateEntry"
  ],
  ["That username is already in use.", "usernameTaken"]
]);

const messageKeyByCode = new Map<string, string>([
  ["AUTH_REQUIRED", "sessionRequired"],
  ["DUPLICATE_ENTRY", "duplicateEntry"],
  ["INVALID_CREDENTIALS", "invalidCredentials"],
  ["LIBRARY_ENTRY_NOT_FOUND", "libraryEntryNotFound"],
  ["MEDIA_RECORD_NOT_FOUND", "mediaRecordNotFound"],
  ["SEARCH_FAILED", "searchLoadFailed"],
  ["USERNAME_TAKEN", "usernameTaken"]
]);

export function getLocalizedApiErrorMessage(
  tErrors: Translate,
  payload: unknown,
  fallbackKey = "generic"
): string {
  const normalized = normalizeApiErrorPayload(payload);

  if (normalized?.code) {
    const key = messageKeyByCode.get(normalized.code);
    if (key) {
      return tErrors(key);
    }
  }

  if (typeof normalized?.message === "string") {
    const key = messageKeyByMessage.get(normalized.message);
    if (key) {
      return tErrors(key);
    }
  }

  if (Array.isArray(normalized?.message)) {
    return normalized.message
      .map((message) => {
        const key = messageKeyByMessage.get(message);
        return key ? tErrors(key) : message;
      })
      .join(", ");
  }

  return tErrors(fallbackKey);
}

export async function getLocalizedApiErrorMessageFromResponse(
  response: Response,
  tErrors: Translate,
  fallbackKey = "generic"
): Promise<string> {
  const payload = (await response.json().catch(() => null)) as unknown;
  return getLocalizedApiErrorMessage(tErrors, payload, fallbackKey);
}

export function normalizeApiErrorPayload(payload: unknown): ApiErrorPayload | null {
  if (!payload || typeof payload !== "object") {
    return null;
  }

  return payload as ApiErrorPayload;
}
