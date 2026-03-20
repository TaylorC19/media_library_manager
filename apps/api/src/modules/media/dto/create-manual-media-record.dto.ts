import {
  mediaTypes,
  type CreateManualMediaRecordRequest
} from "@media-library/types";
import { Transform } from "class-transformer";
import {
  IsArray,
  IsEnum,
  IsInt,
  IsObject,
  IsOptional,
  IsString,
  Max,
  MaxLength,
  Min,
  Validate,
  type ValidationArguments,
  ValidatorConstraint,
  ValidatorConstraintInterface
} from "class-validator";

const MAX_SUMMARY_LENGTH = 5000;
const MAX_BARCODE_CANDIDATES = 20;

@ValidatorConstraint({ name: "manualMediaDetails", async: false })
class ManualMediaDetailsValidator implements ValidatorConstraintInterface {
  validate(
    details: unknown,
    validationArguments?: ValidationArguments
  ): boolean {
    const mediaType = (validationArguments?.object as CreateManualMediaRecordDto)
      .mediaType;

    if (details === undefined) {
      return mediaType !== "album" && mediaType !== "book";
    }

    if (details === null || typeof details !== "object" || Array.isArray(details)) {
      return false;
    }

    const value = details as Record<string, unknown>;

    switch (mediaType) {
      case "album":
        return isStringArray(value.artists);
      case "book":
        return isStringArray(value.authors);
      case "movie":
        return validateOptionalStringArrayFields(value, ["directors", "cast", "genres"]) &&
          validateOptionalNumberFields(value, ["runtimeMinutes"]);
      case "tv":
        return validateOptionalStringArrayFields(value, ["genres", "creators"]) &&
          validateOptionalNumberFields(value, ["seasons", "episodes"]);
      case "game":
        return validateOptionalStringArrayFields(value, [
          "platforms",
          "developers",
          "publishers",
          "genres"
        ]);
      default:
        return false;
    }
  }

  defaultMessage(validationArguments?: ValidationArguments): string {
    const mediaType = (validationArguments?.object as CreateManualMediaRecordDto)
      .mediaType;

    if (mediaType === "album") {
      return "album details must include an artists array";
    }

    if (mediaType === "book") {
      return "book details must include an authors array";
    }

    return "details must match the selected media type";
  }
}

export class CreateManualMediaRecordDto {
  @IsEnum(mediaTypes)
  mediaType!: CreateManualMediaRecordRequest["mediaType"];

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().replace(/\s+/g, " ") : value
  )
  @IsString()
  @MaxLength(240)
  title!: string;

  @Transform(({ value }) => toOptionalInt(value))
  @IsOptional()
  @IsInt()
  @Min(1800)
  @Max(3000)
  year?: number | null;

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  releaseDate?: string | null;

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  imageUrl?: string | null;

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  @MaxLength(MAX_SUMMARY_LENGTH)
  summary?: string | null;

  @Transform(({ value }) => normalizeStringArray(value))
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  @MaxLength(64, { each: true })
  barcodeCandidates?: string[];

  @IsOptional()
  @IsObject()
  @Validate(ManualMediaDetailsValidator)
  details?: CreateManualMediaRecordRequest["details"];
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
    .filter((item) => item.length > 0)
    .slice(0, MAX_BARCODE_CANDIDATES);
}

function toOptionalInt(value: unknown): number | undefined {
  if (typeof value === "number") {
    return Number.isInteger(value) ? value : undefined;
  }

  if (typeof value !== "string" || value.trim().length === 0) {
    return undefined;
  }

  const parsed = Number.parseInt(value, 10);
  return Number.isNaN(parsed) ? undefined : parsed;
}

function isStringArray(value: unknown): boolean {
  return (
    Array.isArray(value) &&
    value.every((item) => typeof item === "string" && item.trim().length > 0)
  );
}

function validateOptionalStringArrayFields(
  value: Record<string, unknown>,
  keys: string[]
): boolean {
  return keys.every((key) => value[key] === undefined || isStringArray(value[key]));
}

function validateOptionalNumberFields(
  value: Record<string, unknown>,
  keys: string[]
): boolean {
  return keys.every(
    (key) =>
      value[key] === undefined ||
      (typeof value[key] === "number" && Number.isFinite(value[key]))
  );
}
