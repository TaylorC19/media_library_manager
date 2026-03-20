import { Inject, Injectable } from "@nestjs/common";
import type { MediaProvider } from "@media-library/provider-sdk";
import type { MediaType, ProviderName } from "@media-library/types";
import { PROVIDER_ADAPTERS } from "./providers.constants";

@Injectable()
export class ProviderRegistryService {
  private readonly providersByName: Map<ProviderName, MediaProvider>;

  constructor(
    @Inject(PROVIDER_ADAPTERS)
    private readonly providers: MediaProvider[]
  ) {
    this.providersByName = new Map(
      providers.map((provider) => [provider.name, provider])
    );
  }

  getProvider(name: ProviderName): MediaProvider | null {
    return this.providersByName.get(name) ?? null;
  }

  getTextSearchProviders(mediaType: MediaType, names?: ProviderName[]): MediaProvider[] {
    return this.getProviders(names).filter(
      (provider) =>
        provider.enabled &&
        provider.capabilities.supportsTextSearch &&
        provider.supportsMediaType(mediaType)
    );
  }

  getBarcodeProviders(
    mediaType?: MediaType,
    names?: ProviderName[]
  ): MediaProvider[] {
    return this.getProviders(names).filter(
      (provider) =>
        provider.enabled &&
        provider.capabilities.supportsBarcodeSearch &&
        (mediaType === undefined || provider.supportsMediaType(mediaType))
    );
  }

  private getProviders(names?: ProviderName[]): MediaProvider[] {
    if (!names || names.length === 0) {
      return this.providers;
    }

    return names
      .map((name) => this.providersByName.get(name) ?? null)
      .filter((provider): provider is MediaProvider => provider !== null);
  }
}
