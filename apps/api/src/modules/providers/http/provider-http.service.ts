import { Injectable } from "@nestjs/common";

type QueryValue = string | number | boolean | null | undefined;

export interface ProviderHttpGetJsonOptions {
  query?: Record<string, QueryValue>;
  headers?: Record<string, string>;
  timeoutMs?: number;
}

export class ProviderHttpError extends Error {
  readonly statusCode?: number;
  readonly url: string;

  constructor(message: string, options: { statusCode?: number; url: string }) {
    super(message);
    this.name = "ProviderHttpError";
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
        const responseText = await response.text();
        throw new ProviderHttpError(
          `Request failed with status ${response.status}: ${responseText.slice(0, 200)}`,
          {
            statusCode: response.status,
            url: requestUrl.toString()
          }
        );
      }

      return (await response.json()) as Response;
    } catch (error) {
      if (error instanceof ProviderHttpError) {
        throw error;
      }

      if (error instanceof Error && error.name === "AbortError") {
        throw new ProviderHttpError(`Request timed out after ${timeoutMs}ms`, {
          url: requestUrl.toString()
        });
      }

      throw new ProviderHttpError("Network request failed", {
        url: requestUrl.toString()
      });
    } finally {
      clearTimeout(timeout);
    }
  }
}
