import { Prop, Schema, SchemaFactory } from "@nestjs/mongoose";
import {
  mediaTypes,
  type ExternalRatings,
  type MediaRecord,
  type MediaType,
  type ProviderRefs
} from "@media-library/types";
import { HydratedDocument, Schema as MongooseSchema } from "mongoose";

export type MediaRecordDocument = HydratedDocument<MediaRecordDocumentModel>;

@Schema({
  collection: "media_records",
  timestamps: true
})
export class MediaRecordDocumentModel {
  @Prop({
    type: String,
    enum: mediaTypes,
    required: true
  })
  mediaType!: MediaType;

  @Prop({
    type: String,
    required: true,
    trim: true
  })
  title!: string;

  @Prop({
    type: String,
    default: null,
    trim: true
  })
  sortTitle!: string | null;

  @Prop({
    type: String,
    default: null
  })
  releaseDate!: string | null;

  @Prop({
    type: Number,
    default: null
  })
  year!: number | null;

  @Prop({
    type: String,
    default: null
  })
  imageUrl!: string | null;

  @Prop({
    type: String,
    default: null
  })
  summary!: string | null;

  @Prop({
    type: {
      tmdb: {
        id: { type: String },
        mediaKind: { type: String, enum: ["movie", "tv"] }
      },
      musicBrainz: {
        id: { type: String }
      },
      discogs: {
        id: { type: String }
      },
      openLibrary: {
        id: { type: String }
      },
      rawg: {
        id: { type: String }
      }
    },
    default: {}
  })
  providerRefs!: ProviderRefs;

  @Prop({
    type: {
      imdb: { type: Number, default: null },
      rottenTomatoes: { type: Number, default: null },
      tmdb: { type: Number, default: null },
      metacritic: { type: Number, default: null }
    },
    default: undefined
  })
  externalRatings?: ExternalRatings;

  @Prop({
    type: [String],
    default: []
  })
  barcodeCandidates!: string[];

  @Prop({
    type: MongooseSchema.Types.Mixed,
    required: true,
    default: {}
  })
  details!: MediaRecord["details"];

  @Prop({
    type: Date,
    default: null
  })
  lastSyncedAt!: Date | null;

  createdAt?: Date;
  updatedAt?: Date;
}

export const MediaRecordSchema =
  SchemaFactory.createForClass(MediaRecordDocumentModel);

MediaRecordSchema.index({ mediaType: 1, title: 1, year: 1 });
MediaRecordSchema.index({ barcodeCandidates: 1 });
MediaRecordSchema.index({
  "providerRefs.tmdb.id": 1,
  "providerRefs.tmdb.mediaKind": 1
});
MediaRecordSchema.index({ "providerRefs.musicBrainz.id": 1 });
MediaRecordSchema.index({ "providerRefs.discogs.id": 1 });
MediaRecordSchema.index({ "providerRefs.openLibrary.id": 1 });
MediaRecordSchema.index({ "providerRefs.rawg.id": 1 });
