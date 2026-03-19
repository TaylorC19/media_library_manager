import { Prop, Schema, SchemaFactory } from "@nestjs/mongoose";
import {
  mediaTypes,
  providerNames,
  type MediaType,
  type ProviderName
} from "@media-library/types";
import { HydratedDocument, Schema as MongooseSchema, Types } from "mongoose";

export type ScanLogDocument = HydratedDocument<ScanLogDocumentModel>;

@Schema({
  collection: "scan_logs",
  timestamps: {
    createdAt: true,
    updatedAt: false
  }
})
export class ScanLogDocumentModel {
  @Prop({
    type: MongooseSchema.Types.ObjectId,
    required: true,
    index: true
  })
  userId!: Types.ObjectId;

  @Prop({
    type: String,
    required: true,
    trim: true
  })
  barcode!: string;

  @Prop({
    type: String,
    enum: mediaTypes,
    default: null
  })
  matchedMediaType!: MediaType | null;

  @Prop({
    type: String,
    enum: providerNames,
    default: null
  })
  matchedProvider!: ProviderName | null;

  createdAt?: Date;
}

export const ScanLogSchema = SchemaFactory.createForClass(ScanLogDocumentModel);

ScanLogSchema.index({ userId: 1, createdAt: -1 });
ScanLogSchema.index({ barcode: 1, createdAt: -1 });
