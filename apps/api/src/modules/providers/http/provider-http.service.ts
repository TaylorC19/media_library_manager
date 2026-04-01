import { Injectable } from "@nestjs/common";

type QueryValue = string | number | boolean | null | undefined;
type ProviderHttpErrorCode =
  | "timeout"
  | "network"
  | "rate_limited"
  | "not_found"
  | "invalid_response"
  | "unauthorized"
  | "forbidden"
  | "configuration"
  | "upstream"
  | "unknown";

export interface ProviderHttpGetJsonOptions {
  query?: Record<string, QueryValue>;
  headers?: Record<string, string>;
  timeoutMs?: number;
}

export class ProviderHttpError extends Error {
  readonly code: ProviderHttpErrorCode;
  readonly statusCode?: number;
  readonly url: string;

  constructor(
    message: string,
    options: { code: ProviderHttpErrorCode; statusCode?: number; url: string }
  ) {
    super(message);
    this.name = "ProviderHttpError";
    this.code = options.code;
    this.statusCode = options.statusCode;
    this.url = options.url;
  }
}

@Injectable()
export class ProviderHttpService {
  async getJson<Response>(
    url: string,
    options: ProviderHttpGetJsonOptions = {}
  ): Promise<Response> {
    const requestUrl = new URL(url);

    if (options.query) {
      for (const [key, value] of Object.entries(options.query)) {
        if (value !== undefined && value !== null && value !== "") {
          requestUrl.searchParams.set(key, String(value));
        }
      }
    }

    const controller = new AbortController();
    const timeoutMs = options.timeoutMs ?? 10_000;
    const timeout = setTimeout(() => controller.abort(), timeoutMs);
    const safeUrl = `${requestUrl.origin}${requestUrl.pathname}`;

    try {
      const response = await fetch(requestUrl, {
        method: "GET",
        headers: {
          accept: "application/json",
          ...options.headers
        },
        signal: controller.signal
      });

      if (!response.ok) {
        throw createHttpStatusError(response.status, safeUrl);
      }

      try {
        return (await response.json()) as Response;
      } catch {
        throw new ProviderHttpError("Provider returned an invalid response.", {
          code: "invalid_response",
          url: safeUrl
        });
      }
    } catch (error) {
      if (error instanceof ProviderHttpError) {
        throw error;
      }

      if (error instanceof Error && error.name === "AbortError") {
        throw new ProviderHttpError("Provider request timed out.", {
          code: "timeout",
          url: safeUrl
        });
      }

      throw new ProviderHttpError("Provider network request failed.", {
        code: "network",
        url: safeUrl
      });
    } finally {
      clearTimeout(timeout);
    }
  }
}

function createHttpStatusError(statusCode: number, url: string): ProviderHttpError {
  if (statusCode === 404) {
    return new ProviderHttpError("Provider record was not found.", {
      code: "not_found",
      statusCode,
      url
    });
  }

  if (statusCode === 429) {
    return new ProviderHttpError("Provider rate limit was reached.", {
      code: "rate_limited",
      statusCode,
      url
    });
  }

  if (statusCode === 401) {
    return new ProviderHttpError("Provider credentials were rejected.", {
      code: "unauthorized",
      statusCode,
      url
    });
  }

  if (statusCode === 403) {
    return new ProviderHttpError("Provider request was forbidden.", {
      code: "forbidden",
      statusCode,
      url
    });
  }

  if (statusCode >= 500) {
    return new ProviderHttpError("Provider service is temporarily unavailable.", {
      code: "upstream",
      statusCode,
      url
    });
  }

  return new ProviderHttpError("Provider request failed.", {
    code: "unknown",
    statusCode,
    url
  });
}
