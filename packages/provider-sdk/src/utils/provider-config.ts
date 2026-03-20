import type { ProviderName } from "@media-library/types";

export interface ProviderRuntimeConfig {
  name: ProviderName;
  baseUrl: string;
  enabled: boolean;
  apiKey?: string;
  userAgent?: string;
  timeoutMs: number;
}

export interface CreateProviderConfigOptions {
  apiKey?: string;
  userAgent?: string;
  enabled?: boolean;
  requiresApiKey?: boolean;
  timeoutMs?: number;
}

export function createProviderConfig(
  name: ProviderName,
  baseUrl: string,
  options: CreateProviderConfigOptions = {}
): ProviderRuntimeConfig {
  const normalizedBaseUrl = baseUrl.trim();
  const normalizedApiKey = options.apiKey?.trim();
  const normalizedUserAgent = options.userAgent?.trim();
  const enabled =
    options.enabled ??
    (normalizedBaseUrl.length > 0 &&
      (!options.requiresApiKey || Boolean(normalizedApiKey)));

  return {
    name,
    baseUrl: normalizedBaseUrl,
    enabled,
    apiKey: normalizedApiKey,
    userAgent: normalizedUserAgent,
    timeoutMs: options.timeoutMs ?? 10_000
  };
}
