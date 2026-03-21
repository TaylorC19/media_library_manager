import {
  libraryBuckets,
  mediaTypes,
  physicalFormats,
  providerNames,
  type ImportMediaRecordRequest
} from "@media-library/types";
import { Transform } from "class-transformer";
import {
  IsArray,
  IsEnum,
  IsOptional,
  IsString,
  MaxLength
} from "class-validator";

const MAX_PROVIDER_ID_LENGTH = 240;
const MAX_NOTES_LENGTH = 4000;
const MAX_TAG_LENGTH = 40;

export class ImportMediaDto implements ImportMediaRecordRequest {
  @IsEnum(providerNames)
  provider!: ImportMediaRecordRequest["provider"];

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().replace(/\s+/g, " ") : value
  )
  @IsString()
  @MaxLength(MAX_PROVIDER_ID_LENGTH)
  providerId!: string;

  @IsEnum(mediaTypes)
  mediaType!: ImportMediaRecordRequest["mediaType"];

  @IsOptional()
  @IsEnum(libraryBuckets)
  bucket?: ImportMediaRecordRequest["bucket"];

  @IsOptional()
  @IsEnum(physicalFormats)
  format?: ImportMediaRecordRequest["format"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  barcode?: ImportMediaRecordRequest["barcode"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  purchaseDate?: ImportMediaRecordRequest["purchaseDate"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  @MaxLength(MAX_NOTES_LENGTH)
  notes?: ImportMediaRecordRequest["notes"];

  @Transform(({ value }) => normalizeStringArray(value))
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  @MaxLength(MAX_TAG_LENGTH, { each: true })
  tags?: ImportMediaRecordRequest["tags"];
}

function normalizeOptionalString(value: unknown): string | null | undefined {
  if (value === undefined) {
    return undefined;
  }

  if (value === null) {
    return null;
  }

  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function normalizeStringArray(value: unknown): string[] | undefined {
  if (value === undefined) {
    return undefined;
  }

  if (!Array.isArray(value)) {
    return undefined;
  }

  return value
    .filter((item): item is string => typeof item === "string")
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}
