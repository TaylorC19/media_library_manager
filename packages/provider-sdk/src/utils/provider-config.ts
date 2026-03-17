import type { ProviderName } from "@media-library/types";

export interface ProviderRuntimeConfig {
  name: ProviderName;
  baseUrl: string;
  apiKey?: string;
  userAgent?: string;
}

export function createProviderConfig(
  name: ProviderName,
  baseUrl: string,
  options?: Pick<ProviderRuntimeConfig, "apiKey" | "userAgent">
): ProviderRuntimeConfig {
  return {
    name,
    baseUrl,
    apiKey: options?.apiKey,
    userAgent: options?.userAgent
  };
}
