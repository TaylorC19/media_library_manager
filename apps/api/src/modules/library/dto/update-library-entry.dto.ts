import {
  libraryBuckets,
  physicalFormats,
  type UpdateLibraryEntryRequest
} from "@media-library/types";
import { Transform } from "class-transformer";
import {
  IsArray,
  IsEnum,
  IsOptional,
  IsString,
  MaxLength
} from "class-validator";

const MAX_NOTES_LENGTH = 4000;
const MAX_TAG_LENGTH = 40;

export class UpdateLibraryEntryDto implements UpdateLibraryEntryRequest {
  @IsOptional()
  @IsEnum(libraryBuckets)
  bucket?: UpdateLibraryEntryRequest["bucket"];

  @IsOptional()
  @IsEnum(physicalFormats)
  format?: UpdateLibraryEntryRequest["format"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  barcode?: string | null;

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  purchaseDate?: string | null;

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  @MaxLength(MAX_NOTES_LENGTH)
  notes?: string | null;

  @Transform(({ value }) => normalizeStringArray(value))
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  @MaxLength(MAX_TAG_LENGTH, { each: true })
  tags?: string[];
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
