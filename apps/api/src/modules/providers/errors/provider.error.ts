import type { ProviderName } from "@media-library/types";
import type { ProviderOperation } from "../providers.types";

export interface ProviderErrorOptions {
  provider: ProviderName;
  operation: ProviderOperation;
  message: string;
  cause?: unknown;
  statusCode?: number;
  retryable?: boolean;
}

export class ProviderError extends Error {
  override readonly name = "ProviderError";
  readonly provider: ProviderName;
  readonly operation: ProviderOperation;
  override readonly cause?: unknown;
  readonly statusCode?: number;
  readonly retryable: boolean;

  constructor(options: ProviderErrorOptions) {
    super(options.message);
    this.provider = options.provider;
    this.operation = options.operation;
    this.cause = options.cause;
    this.statusCode = options.statusCode;
    this.retryable = options.retryable ?? false;
  }
}
