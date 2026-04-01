import { Injectable, Logger } from "@nestjs/common";
import type { MediaProvider } from "@media-library/provider-sdk";
import type {
  NormalizedSearchResult
} from "@media-library/types";
import { ProviderError } from "./errors/provider.error";
import { toProviderError } from "./errors/provider-error.utils";
import { ProviderReliabilityService } from "./provider-reliability.service";
import { ProviderRegistryService } from "./provider-registry.service";
import type {
  ProviderBarcodeSearchRequest,
  ProviderBarcodeSearchResponse,
  ProviderDetailsRequest,
  ProviderDetailsResponse,
  ProviderFailure,
  ProviderSearchResponse,
  ProviderSearchTextRequest
} from "./providers.types";

@Injectable()
export class ProvidersService {
  private readonly logger = new Logger(ProvidersService.name);

  constructor(
    private readonly providerRegistry: ProviderRegistryService,
    private readonly providerReliabilityService: ProviderReliabilityService
  ) {}

  async searchByText(
    request: ProviderSearchTextRequest
  ): Promise<ProviderSearchResponse> {
    const providers = this.providerRegistry.getTextSearchProviders(
      request.mediaType,
      request.providers
    );

    const settledResults = await Promise.allSettled(
      providers.map((provider) =>
        provider.searchByText({
          query: request.query,
          mediaType: request.mediaType,
          limit: request.limit
        })
      )
    );

    return collectSearchResults(settledResults, providers, "searchByText", this.logger);
  }

  async searchByBarcode(
    request: ProviderBarcodeSearchRequest
  ): Promise<ProviderBarcodeSearchResponse> {
    const providers = this.providerRegistry.getBarcodeProviders(
      request.mediaType,
      request.providers
    );

    const settledResults = await Promise.allSettled(
      providers.map((provider) =>
        provider.searchByBarcode?.({
          barcode: request.barcode,
          mediaType: request.mediaType,
          limit: request.limit
        }) ?? Promise.resolve([])
      )
    );

    return collectSearchResults(
      settledResults,
      providers,
      "searchByBarcode",
      this.logger
    );
  }

  async getDetails(request: ProviderDetailsRequest): Promise<ProviderDetailsResponse> {
    const provider = this.providerRegistry.getProvider(request.provider);

    if (!provider) {
      const fallbackFailure =
        this.providerReliabilityService.getFallbackFailure("detail lookup");

      return {
        record: null,
        failure: {
          provider: request.provider,
          operation: "getDetails",
          code: fallbackFailure.code,
          message: fallbackFailure.message,
          retryable: false
        }
      };
    }

    if (!provider.enabled || !provider.supportsMediaType(request.mediaType)) {
      return {
        record: null,
        failure: {
          provider: request.provider,
          operation: "getDetails",
          code: provider.supportsMediaType(request.mediaType)
            ? "configuration"
            : "unsupported",
          message: provider.supportsMediaType(request.mediaType)
            ? "Provider is unavailable for this media type."
            : "Provider does not support this media type.",
          retryable: false
        }
      };
    }

    try {
      const record = await provider.getDetails({
        providerId: request.providerId,
        mediaType: request.mediaType
      });

      return { record };
    } catch (error) {
      const providerError = toProviderError(
        request.provider,
        "getDetails",
        error
      );
      logProviderFailure(this.logger, providerError);

      return {
        record: null,
        failure: toFailure(providerError)
      };
    }
  }
}

function collectSearchResults(
  settledResults: PromiseSettledResult<NormalizedSearchResult[]>[],
  providers: MediaProvider[],
  operation: ProviderFailure["operation"],
  logger: Logger
): ProviderSearchResponse {
  const results: NormalizedSearchResult[] = [];
  const failures: ProviderFailure[] = [];

  settledResults.forEach((result, index) => {
    const provider = providers[index];
    if (!provider) {
      return;
    }

    if (result.status === "fulfilled") {
      results.push(...result.value);
      return;
    }

    const providerError = toProviderError(provider.name, operation, result.reason);
    failures.push(toFailure(providerError));
    logProviderFailure(logger, providerError);
  });

  return {
    results,
    failures
  };
}

function toFailure(error: ProviderError): ProviderFailure {
  return {
    provider: error.provider,
    operation: error.operation,
    code: error.code,
    message: error.safeMessage,
    statusCode: error.statusCode,
    retryable: error.retryable
  };
}

function logProviderFailure(logger: Logger, error: ProviderError): void {
  logger.warn(
    JSON.stringify({
      event: "provider_failure",
      provider: error.provider,
      operation: error.operation,
      code: error.code,
      statusCode: error.statusCode,
      retryable: error.retryable,
      requestUrl: error.requestUrl,
      message: error.safeMessage
    })
  );
}
