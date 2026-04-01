import { Injectable } from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import {
  DEFAULT_MUSICBRAINZ_THROTTLE,
  DEFAULT_PROVIDER_CACHE_TTLS,
  DEFAULT_PROVIDER_CACHE_VERSION
} from "./providers.constants";
import type {
  ProviderCacheOperation,
  ProviderCachePolicy,
  ProviderErrorCode
} from "./providers.types";

interface MusicBrainzThrottleSettings {
  minIntervalMs: number;
  rateLimitCooldownMs: number;
  retryableCooldownMs: number;
}

@Injectable()
export class ProviderReliabilityService {
  constructor(private readonly configService: ConfigService) {}

  getCachePolicy(operation: ProviderCacheOperation): ProviderCachePolicy {
    switch (operation) {
      case "search":
        return {
          operation,
          ttlMs: this.getMsEnv(
            "PROVIDER_CACHE_SEARCH_TTL_MS",
            DEFAULT_PROVIDER_CACHE_TTLS.searchMs
          ),
          negativeTtlMs: this.getMsEnv(
            "PROVIDER_CACHE_NEGATIVE_SEARCH_TTL_MS",
            DEFAULT_PROVIDER_CACHE_TTLS.negativeSearchMs
          )
        };
      case "detail":
        return {
          operation,
          ttlMs: this.getMsEnv(
            "PROVIDER_CACHE_DETAIL_TTL_MS",
            DEFAULT_PROVIDER_CACHE_TTLS.detailMs
          ),
          negativeTtlMs: this.getMsEnv(
            "PROVIDER_CACHE_NEGATIVE_DETAIL_TTL_MS",
            DEFAULT_PROVIDER_CACHE_TTLS.negativeDetailMs
          )
        };
      case "barcode":
        return {
          operation,
          ttlMs: this.getMsEnv(
            "PROVIDER_CACHE_BARCODE_TTL_MS",
            DEFAULT_PROVIDER_CACHE_TTLS.barcodeMs
          ),
          negativeTtlMs: this.getMsEnv(
            "PROVIDER_CACHE_NEGATIVE_BARCODE_TTL_MS",
            DEFAULT_PROVIDER_CACHE_TTLS.negativeBarcodeMs
          )
        };
    }
  }

  getCacheVersionPrefix(): string {
    const configuredPrefix = this.configService
      .get<string>("PROVIDER_CACHE_VERSION")
      ?.trim();
    return configuredPrefix || DEFAULT_PROVIDER_CACHE_VERSION;
  }

  getMusicBrainzThrottleSettings(): MusicBrainzThrottleSettings {
    return {
      minIntervalMs: this.getMsEnv(
        "MUSICBRAINZ_MIN_INTERVAL_MS",
        DEFAULT_MUSICBRAINZ_THROTTLE.minIntervalMs
      ),
      rateLimitCooldownMs: this.getMsEnv(
        "MUSICBRAINZ_RATE_LIMIT_COOLDOWN_MS",
        DEFAULT_MUSICBRAINZ_THROTTLE.rateLimitCooldownMs
      ),
      retryableCooldownMs: this.getMsEnv(
        "MUSICBRAINZ_RETRYABLE_COOLDOWN_MS",
        DEFAULT_MUSICBRAINZ_THROTTLE.retryableCooldownMs
      )
    };
  }

  isNegativeCacheEligible<Payload>(
    operation: ProviderCacheOperation,
    value: Payload
  ): boolean {
    switch (operation) {
      case "search":
      case "barcode":
        return Array.isArray(value) && value.length === 0;
      case "detail":
        return value === null;
    }
  }

  getFallbackFailure(operation: string): {
    code: ProviderErrorCode;
    message: string;
    retryable: boolean;
  } {
    return {
      code: "unavailable",
      message: `Provider is unavailable for ${operation}.`,
      retryable: false
    };
  }

  private getMsEnv(name: string, defaultValue: number): number {
    const rawValue = this.configService.get<string>(name);
    const parsedValue = Number(rawValue);

    if (!rawValue || !Number.isFinite(parsedValue) || parsedValue < 0) {
      return defaultValue;
    }

    return parsedValue;
  }
}
