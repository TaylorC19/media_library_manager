import { Injectable } from "@nestjs/common";
import { InjectModel } from "@nestjs/mongoose";
import type { MediaRecord, ProviderRefs } from "@media-library/types";
import { Model, Types } from "mongoose";
import {
  MediaRecordDocumentModel,
  type MediaRecordDocument
} from "../schemas/media-record.schema";

type ProviderRefKey = keyof ProviderRefs;

export type CreateMediaRecordInput = Omit<
  MediaRecord,
  "id" | "createdAt" | "updatedAt"
>;

@Injectable()
export class MediaRecordRepository {
  constructor(
    @InjectModel(MediaRecordDocumentModel.name)
    private readonly mediaRecordModel: Model<MediaRecordDocument>
  ) {}

  async create(input: CreateMediaRecordInput): Promise<MediaRecord> {
    const mediaRecord = await this.mediaRecordModel.create({
      ...input,
      lastSyncedAt: input.lastSyncedAt ? new Date(input.lastSyncedAt) : null
    });

    return this.toDomain(mediaRecord);
  }

  async findByBarcodeCandidate(barcode: string): Promise<MediaRecord[]> {
    const mediaRecords = await this.mediaRecordModel
      .find({ barcodeCandidates: barcode })
      .exec();

    return mediaRecords.map((mediaRecord) => this.toDomain(mediaRecord));
  }

  async findById(id: string): Promise<MediaRecord | null> {
    if (!Types.ObjectId.isValid(id)) {
      return null;
    }

    const mediaRecord = await this.mediaRecordModel.findById(id).exec();
    return mediaRecord ? this.toDomain(mediaRecord) : null;
  }

  async findByProviderRef(
    provider: ProviderRefKey,
    providerId: string,
    options?: {
      tmdbMediaKind?: "movie" | "tv";
    }
  ): Promise<MediaRecord | null> {
    const query: Record<string, unknown> = {
      [`providerRefs.${provider}.id`]: providerId
    };

    if (provider === "tmdb" && options?.tmdbMediaKind) {
      query["providerRefs.tmdb.mediaKind"] = options.tmdbMediaKind;
    }

    const mediaRecord = await this.mediaRecordModel.findOne(query).exec();
    return mediaRecord ? this.toDomain(mediaRecord) : null;
  }

  async findByTitleYear(params: {
    mediaType: MediaRecord["mediaType"];
    title: string;
    year?: number | null;
  }): Promise<MediaRecord[]> {
    const query: Record<string, unknown> = {
      mediaType: params.mediaType,
      title: params.title.trim()
    };

    if (params.year !== undefined) {
      query.year = params.year;
    }

    const mediaRecords = await this.mediaRecordModel.find(query).exec();
    return mediaRecords.map((mediaRecord) => this.toDomain(mediaRecord));
  }

  private toDomain(mediaRecord: MediaRecordDocument): MediaRecord {
    return {
      id: mediaRecord._id.toString(),
      mediaType: mediaRecord.mediaType,
      title: mediaRecord.title,
      sortTitle: mediaRecord.sortTitle ?? null,
      releaseDate: mediaRecord.releaseDate ?? null,
      year: mediaRecord.year ?? null,
      imageUrl: mediaRecord.imageUrl ?? null,
      summary: mediaRecord.summary ?? null,
      providerRefs: mediaRecord.providerRefs,
      externalRatings: mediaRecord.externalRatings,
      barcodeCandidates: mediaRecord.barcodeCandidates,
      details: mediaRecord.details,
      createdAt:
        mediaRecord.createdAt?.toISOString() ?? new Date().toISOString(),
      updatedAt:
        mediaRecord.updatedAt?.toISOString() ?? new Date().toISOString(),
      lastSyncedAt: mediaRecord.lastSyncedAt?.toISOString() ?? null
    } as MediaRecord;
  }
}
