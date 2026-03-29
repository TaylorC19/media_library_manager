import { Injectable } from "@nestjs/common";
import { InjectModel } from "@nestjs/mongoose";
import type {
  LibraryBucket,
  LibraryEntry,
  MediaType,
  PhysicalFormat
} from "@media-library/types";
import { Model, Types } from "mongoose";
import {
  LibraryEntryDocumentModel,
  type LibraryEntryDocument
} from "../schemas/library-entry.schema";

export interface CreateLibraryEntryInput {
  userId: string;
  mediaRecordId: string;
  bucket: LibraryBucket;
  mediaType: MediaType;
  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;
  notes?: string | null;
  tags?: string[];
}

export interface LibraryEntryFilters {
  bucket?: LibraryBucket;
  mediaType?: MediaType;
  tag?: string;
}

export interface UpdateLibraryEntryInput {
  bucket?: LibraryBucket;
  format?: PhysicalFormat | null;
  barcode?: string | null;
  purchaseDate?: string | null;
  notes?: string | null;
  tags?: string[];
}

@Injectable()
export class LibraryEntryRepository {
  constructor(
    @InjectModel(LibraryEntryDocumentModel.name)
    private readonly libraryEntryModel: Model<LibraryEntryDocument>
  ) {}

  async create(input: CreateLibraryEntryInput): Promise<LibraryEntry> {
    const libraryEntry = await this.libraryEntryModel.create({
      ...input,
      userId: new Types.ObjectId(input.userId),
      mediaRecordId: new Types.ObjectId(input.mediaRecordId),
      format: input.format ?? null,
      barcode: input.barcode ?? null,
      purchaseDate: input.purchaseDate ?? null,
      notes: input.notes ?? null,
      tags: input.tags ?? []
    });

    return this.toDomain(libraryEntry);
  }

  async findByIdForUser(id: string, userId: string): Promise<LibraryEntry | null> {
    if (!Types.ObjectId.isValid(id) || !Types.ObjectId.isValid(userId)) {
      return null;
    }

    const libraryEntry = await this.libraryEntryModel
      .findOne({
        _id: id,
        userId
      })
      .exec();

    return libraryEntry ? this.toDomain(libraryEntry) : null;
  }

  async findDuplicate(input: {
    userId: string;
    mediaRecordId: string;
    bucket: LibraryBucket;
    format?: PhysicalFormat | null;
  }): Promise<LibraryEntry | null> {
    if (!Types.ObjectId.isValid(input.userId) || !Types.ObjectId.isValid(input.mediaRecordId)) {
      return null;
    }

    const libraryEntry = await this.libraryEntryModel
      .findOne({
        userId: input.userId,
        mediaRecordId: input.mediaRecordId,
        bucket: input.bucket,
        format: input.format ?? null
      })
      .exec();

    return libraryEntry ? this.toDomain(libraryEntry) : null;
  }

  async listByUser(
    userId: string,
    filters: LibraryEntryFilters = {}
  ): Promise<LibraryEntry[]> {
    if (!Types.ObjectId.isValid(userId)) {
      return [];
    }

    const query: Record<string, unknown> = { userId };

    if (filters.bucket) {
      query.bucket = filters.bucket;
    }

    if (filters.mediaType) {
      query.mediaType = filters.mediaType;
    }

    if (filters.tag) {
      query.tags = filters.tag;
    }

    const libraryEntries = await this.libraryEntryModel
      .find(query)
      .sort({ createdAt: -1 })
      .exec();

    return libraryEntries.map((libraryEntry) => this.toDomain(libraryEntry));
  }

  async findByMediaRecordIdsForUser(
    userId: string,
    mediaRecordIds: string[]
  ): Promise<LibraryEntry[]> {
    if (!Types.ObjectId.isValid(userId)) {
      return [];
    }

    const validMediaRecordIds = mediaRecordIds.filter((id) => Types.ObjectId.isValid(id));
    if (validMediaRecordIds.length === 0) {
      return [];
    }

    const libraryEntries = await this.libraryEntryModel
      .find({
        userId,
        mediaRecordId: { $in: validMediaRecordIds }
      })
      .sort({ createdAt: -1 })
      .exec();

    return libraryEntries.map((libraryEntry) => this.toDomain(libraryEntry));
  }

  async updateForUser(
    id: string,
    userId: string,
    input: UpdateLibraryEntryInput
  ): Promise<LibraryEntry | null> {
    if (!Types.ObjectId.isValid(id) || !Types.ObjectId.isValid(userId)) {
      return null;
    }

    const nextValues: Record<string, unknown> = {};

    if (input.bucket !== undefined) {
      nextValues.bucket = input.bucket;
    }

    if (input.format !== undefined) {
      nextValues.format = input.format;
    }

    if (input.barcode !== undefined) {
      nextValues.barcode = input.barcode;
    }

    if (input.purchaseDate !== undefined) {
      nextValues.purchaseDate = input.purchaseDate;
    }

    if (input.notes !== undefined) {
      nextValues.notes = input.notes;
    }

    if (input.tags !== undefined) {
      nextValues.tags = input.tags;
    }

    const libraryEntry = await this.libraryEntryModel
      .findOneAndUpdate(
        {
          _id: id,
          userId
        },
        { $set: nextValues },
        { new: true }
      )
      .exec();

    return libraryEntry ? this.toDomain(libraryEntry) : null;
  }

  async deleteForUser(id: string, userId: string): Promise<boolean> {
    if (!Types.ObjectId.isValid(id) || !Types.ObjectId.isValid(userId)) {
      return false;
    }

    const result = await this.libraryEntryModel
      .deleteOne({
        _id: id,
        userId
      })
      .exec();

    return result.deletedCount > 0;
  }

  private toDomain(libraryEntry: LibraryEntryDocument): LibraryEntry {
    return {
      id: libraryEntry._id.toString(),
      userId: libraryEntry.userId.toString(),
      mediaRecordId: libraryEntry.mediaRecordId.toString(),
      bucket: libraryEntry.bucket,
      mediaType: libraryEntry.mediaType,
      format: libraryEntry.format ?? null,
      barcode: libraryEntry.barcode ?? null,
      purchaseDate: libraryEntry.purchaseDate ?? null,
      notes: libraryEntry.notes ?? null,
      tags: libraryEntry.tags,
      createdAt:
        libraryEntry.createdAt?.toISOString() ?? new Date().toISOString(),
      updatedAt:
        libraryEntry.updatedAt?.toISOString() ?? new Date().toISOString()
    };
  }
}
