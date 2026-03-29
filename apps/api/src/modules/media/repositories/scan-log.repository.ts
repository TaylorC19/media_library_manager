import { Injectable } from "@nestjs/common";
import { InjectModel } from "@nestjs/mongoose";
import type { MediaType, ProviderName } from "@media-library/types";
import { Model, Types } from "mongoose";
import {
  ScanLogDocumentModel,
  type ScanLogDocument
} from "../schemas/scan-log.schema";

@Injectable()
export class ScanLogRepository {
  constructor(
    @InjectModel(ScanLogDocumentModel.name)
    private readonly scanLogModel: Model<ScanLogDocument>
  ) {}

  async create(input: {
    userId: string;
    barcode: string;
    matchedMediaType?: MediaType | null;
    matchedProvider?: ProviderName | null;
  }): Promise<void> {
    if (!Types.ObjectId.isValid(input.userId)) {
      return;
    }

    await this.scanLogModel.create({
      userId: new Types.ObjectId(input.userId),
      barcode: input.barcode,
      matchedMediaType: input.matchedMediaType ?? null,
      matchedProvider: input.matchedProvider ?? null
    });
  }
}
