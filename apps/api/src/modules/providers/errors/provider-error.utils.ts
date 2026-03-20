import type { ProviderName } from "@media-library/types";
import { ProviderError } from "./provider.error";
import type { ProviderOperation } from "../providers.types";
import type { ProviderHttpError } from "../http/provider-http.service";

export function isProviderError(error: unknown): error is ProviderError {
  return error instanceof ProviderError;
}

export function toProviderError(
  provider: ProviderName,
  operation: ProviderOperation,
  error: unknown
): ProviderError {
  if (error instanceof ProviderError) {
    return error;
  }

  const httpError = error as ProviderHttpError | undefined;
  if (
    httpError &&
    typeof httpError === "object" &&
    "statusCode" in httpError &&
    "url" in httpError
  ) {
    return new ProviderError({
      provider,
      operation,
      message: httpError.message,
      cause: error,
      statusCode: httpError.statusCode,
      retryable:
        typeof httpError.statusCode === "number" && httpError.statusCode >= 500
    });
  }

  if (error instanceof Error) {
    return new ProviderError({
      provider,
      operation,
      message: error.message,
      cause: error
    });
  }

  return new ProviderError({
    provider,
    operation,
    message: "Unknown provider error",
    cause: error
  });
}
