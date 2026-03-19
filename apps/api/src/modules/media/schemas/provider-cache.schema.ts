import { Prop, Schema, SchemaFactory } from "@nestjs/mongoose";
import { providerNames, type ProviderName } from "@media-library/types";
import { HydratedDocument, Schema as MongooseSchema } from "mongoose";

export type ProviderCacheDocument = HydratedDocument<ProviderCacheDocumentModel>;

@Schema({
  collection: "provider_cache",
  timestamps: {
    createdAt: true,
    updatedAt: false
  }
})
export class ProviderCacheDocumentModel {
  @Prop({
    type: String,
    enum: providerNames,
    required: true
  })
  provider!: ProviderName;

  @Prop({
    type: String,
    required: true
  })
  cacheKey!: string;

  @Prop({
    type: MongooseSchema.Types.Mixed,
    required: true
  })
  payload!: unknown;

  @Prop({
    type: Date,
    required: true
  })
  expiresAt!: Date;

  createdAt?: Date;
}

export const ProviderCacheSchema =
  SchemaFactory.createForClass(ProviderCacheDocumentModel);

ProviderCacheSchema.index({ provider: 1, cacheKey: 1 }, { unique: true });
ProviderCacheSchema.index({ expiresAt: 1 }, { expireAfterSeconds: 0 });
