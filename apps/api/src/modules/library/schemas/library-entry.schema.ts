import { Prop, Schema, SchemaFactory } from "@nestjs/mongoose";
import {
  libraryBuckets,
  mediaTypes,
  physicalFormats,
  type LibraryBucket,
  type MediaType,
  type PhysicalFormat
} from "@media-library/types";
import { HydratedDocument, Schema as MongooseSchema, Types } from "mongoose";

export type LibraryEntryDocument = HydratedDocument<LibraryEntryDocumentModel>;

@Schema({
  collection: "library_entries",
  timestamps: true
})
export class LibraryEntryDocumentModel {
  @Prop({
    type: MongooseSchema.Types.ObjectId,
    required: true,
    index: true
  })
  userId!: Types.ObjectId;

  @Prop({
    type: MongooseSchema.Types.ObjectId,
    required: true,
    index: true
  })
  mediaRecordId!: Types.ObjectId;

  @Prop({
    type: String,
    enum: libraryBuckets,
    required: true
  })
  bucket!: LibraryBucket;

  @Prop({
    type: String,
    enum: mediaTypes,
    required: true
  })
  mediaType!: MediaType;

  @Prop({
    type: String,
    enum: physicalFormats,
    default: null
  })
  format!: PhysicalFormat | null;

  @Prop({
    type: String,
    default: null,
    trim: true
  })
  barcode!: string | null;

  @Prop({
    type: String,
    default: null,
    trim: true
  })
  purchaseDate!: string | null;

  @Prop({
    type: String,
    default: null
  })
  notes!: string | null;

  @Prop({
    type: [String],
    default: []
  })
  tags!: string[];

  createdAt?: Date;
  updatedAt?: Date;
}

export const LibraryEntrySchema =
  SchemaFactory.createForClass(LibraryEntryDocumentModel);

LibraryEntrySchema.index({ userId: 1, bucket: 1, mediaType: 1 });
LibraryEntrySchema.index({ userId: 1, createdAt: -1 });
LibraryEntrySchema.index({ userId: 1, tags: 1 });
LibraryEntrySchema.index({ userId: 1, mediaRecordId: 1 });
LibraryEntrySchema.index(
  { userId: 1, mediaRecordId: 1, bucket: 1, format: 1 },
  { unique: true }
);
