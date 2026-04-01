import type { ProviderName } from "@media-library/types";
import type {
  ProviderErrorCode,
  ProviderOperation
} from "../providers.types";

export interface ProviderErrorOptions {
  provider: ProviderName;
  operation: ProviderOperation;
  code: ProviderErrorCode;
  safeMessage: string;
  cause?: unknown;
  statusCode?: number;
  retryable?: boolean;
  requestUrl?: string;
}

export class ProviderError extends Error {
  override readonly name = "ProviderError";
  readonly provider: ProviderName;
  readonly operation: ProviderOperation;
  readonly code: ProviderErrorCode;
  readonly safeMessage: string;
  override readonly cause?: unknown;
  readonly statusCode?: number;
  readonly retryable: boolean;
  readonly requestUrl?: string;

  constructor(options: ProviderErrorOptions) {
    super(options.safeMessage);
    this.provider = options.provider;
    this.operation = options.operation;
    this.code = options.code;
    this.safeMessage = options.safeMessage;
    this.cause = options.cause;
    this.statusCode = options.statusCode;
    this.retryable = options.retryable ?? false;
    this.requestUrl = options.requestUrl;
  }
}
