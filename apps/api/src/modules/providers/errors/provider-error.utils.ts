import type { ProviderName } from "@media-library/types";
import { ProviderError } from "./provider.error";
import type {
  ProviderErrorCode,
  ProviderOperation
} from "../providers.types";
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
    "code" in httpError &&
    "statusCode" in httpError &&
    "url" in httpError
  ) {
    return new ProviderError({
      provider,
      operation,
      code: httpError.code as ProviderErrorCode,
      safeMessage: httpError.message,
      cause: error,
      statusCode: httpError.statusCode,
      retryable: isRetryableHttpError(httpError),
      requestUrl: httpError.url
    });
  }

  if (error instanceof Error) {
    return new ProviderError({
      provider,
      operation,
      code: "unknown",
      safeMessage: error.message,
      cause: error
    });
  }

  return new ProviderError({
    provider,
    operation,
    code: "unknown",
    safeMessage: "Unknown provider error",
    cause: error
  });
}

export function createProviderConfigurationError(
  provider: ProviderName,
  operation: ProviderOperation,
  safeMessage: string
): ProviderError {
  return new ProviderError({
    provider,
    operation,
    code: "configuration",
    safeMessage,
    retryable: false
  });
}

export function createInvalidProviderResponseError(
  provider: ProviderName,
  operation: ProviderOperation,
  safeMessage = "Provider returned an invalid response."
): ProviderError {
  return new ProviderError({
    provider,
    operation,
    code: "invalid_response",
    safeMessage,
    retryable: false
  });
}

function isRetryableHttpError(error: ProviderHttpError): boolean {
  return (
    error.code === "timeout" ||
    error.code === "network" ||
    error.code === "rate_limited" ||
    error.code === "upstream"
  );
}
