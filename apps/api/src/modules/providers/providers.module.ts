import { Module } from "@nestjs/common";
import { MongooseModule } from "@nestjs/mongoose";
import { DiscogsProvider } from "./adapters/discogs.provider";
import { MusicBrainzProvider } from "./adapters/musicbrainz.provider";
import { OpenLibraryProvider } from "./adapters/openlibrary.provider";
import { RawgProvider } from "./adapters/rawg.provider";
import { TmdbProvider } from "./adapters/tmdb.provider";
import { ProviderCacheService } from "./provider-cache.service";
import { ProviderRegistryService } from "./provider-registry.service";
import { ProvidersService } from "./providers.service";
import { ProviderCacheRepository } from "./repositories/provider-cache.repository";
import { PROVIDER_ADAPTERS } from "./providers.constants";
import { ProviderHttpService } from "./http/provider-http.service";
import { MusicBrainzThrottleService } from "./http/musicbrainz-throttle.service";
import {
  ProviderCacheDocumentModel,
  ProviderCacheSchema
} from "../media/schemas/provider-cache.schema";

@Module({
  imports: [
    MongooseModule.forFeature([
      {
        name: ProviderCacheDocumentModel.name,
        schema: ProviderCacheSchema
      }
    ])
  ],
  providers: [
    ProviderHttpService,
    ProviderCacheRepository,
    ProviderCacheService,
    MusicBrainzThrottleService,
    TmdbProvider,
    MusicBrainzProvider,
    OpenLibraryProvider,
    RawgProvider,
    DiscogsProvider,
    {
      provide: PROVIDER_ADAPTERS,
      inject: [
        TmdbProvider,
        MusicBrainzProvider,
        OpenLibraryProvider,
        RawgProvider,
        DiscogsProvider
      ],
      useFactory: (
        tmdbProvider: TmdbProvider,
        musicBrainzProvider: MusicBrainzProvider,
        openLibraryProvider: OpenLibraryProvider,
        rawgProvider: RawgProvider,
        discogsProvider: DiscogsProvider
      ) => [
        tmdbProvider,
        musicBrainzProvider,
        openLibraryProvider,
        rawgProvider,
        discogsProvider
      ]
    },
    ProviderRegistryService,
    ProvidersService
  ],
  exports: [ProviderRegistryService, ProvidersService]
})
export class ProvidersModule {}
