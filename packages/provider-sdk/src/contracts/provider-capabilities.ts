import type { MediaType } from "@media-library/types";

export interface ProviderCapabilities {
  readonly mediaTypes: readonly MediaType[];
  readonly supportsTextSearch: boolean;
  readonly supportsDetailLookup: boolean;
  readonly supportsBarcodeSearch: boolean;
}

export function providerSupportsMediaType(
  capabilities: ProviderCapabilities,
  mediaType: MediaType
): boolean {
  return capabilities.mediaTypes.includes(mediaType);
}
