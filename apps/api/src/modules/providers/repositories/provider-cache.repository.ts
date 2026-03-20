import { Injectable } from "@nestjs/common";
import { InjectModel } from "@nestjs/mongoose";
import type { ProviderName } from "@media-library/types";
import { Model } from "mongoose";
import {
  ProviderCacheDocumentModel,
  type ProviderCacheDocument
} from "../../media/schemas/provider-cache.schema";

@Injectable()
export class ProviderCacheRepository {
  constructor(
    @InjectModel(ProviderCacheDocumentModel.name)
    private readonly providerCacheModel: Model<ProviderCacheDocument>
  ) {}

  async findActive<Payload>(
    provider: ProviderName,
    cacheKey: string
  ): Promise<{ payload: Payload } | null> {
    const cacheEntry = await this.providerCacheModel
      .findOne({
        provider,
        cacheKey,
        expiresAt: {
          $gt: new Date()
        }
      })
      .lean()
      .exec();

    if (!cacheEntry) {
      return null;
    }

    return {
      payload: cacheEntry.payload as Payload
    };
  }

  async set<Payload>(
    provider: ProviderName,
    cacheKey: string,
    payload: Payload,
    expiresAt: Date
  ): Promise<void> {
    await this.providerCacheModel
      .findOneAndUpdate(
        {
          provider,
          cacheKey
        },
        {
          provider,
          cacheKey,
          payload,
          expiresAt
        },
        {
          upsert: true,
          new: true,
          setDefaultsOnInsert: true
        }
      )
      .exec();
  }
}
