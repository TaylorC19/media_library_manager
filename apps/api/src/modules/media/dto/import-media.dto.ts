import {
  BadRequestException,
  Injectable,
  ValidationPipe,
  type ArgumentMetadata,
  type PipeTransform
} from "@nestjs/common";
import {
  libraryBuckets,
  mediaTypes,
  physicalFormats,
  providerNames,
  type ImportMediaEntryInput,
  type ImportMediaRecordFromProviderRefRequest,
  type ImportMediaRecordFromSearchResultRequest,
  type ImportMediaRecordRequest,
  type NormalizedSearchResult
} from "@media-library/types";
import { Transform, Type } from "class-transformer";
import {
  Equals,
  IsArray,
  IsEnum,
  IsNumber,
  IsOptional,
  IsString,
  MaxLength,
  ValidateNested
} from "class-validator";

const MAX_PROVIDER_ID_LENGTH = 240;
const MAX_NOTES_LENGTH = 4000;
const MAX_TAG_LENGTH = 40;
const MAX_TITLE_LENGTH = 240;
const MAX_SUMMARY_LENGTH = 5000;

export class ImportMediaEntryDto implements ImportMediaEntryInput {
  @IsEnum(libraryBuckets)
  bucket!: ImportMediaEntryInput["bucket"];

  @IsOptional()
  @IsEnum(physicalFormats)
  format?: ImportMediaEntryInput["format"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  barcode?: ImportMediaEntryInput["barcode"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  purchaseDate?: ImportMediaEntryInput["purchaseDate"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  @MaxLength(MAX_NOTES_LENGTH)
  notes?: ImportMediaEntryInput["notes"];

  @Transform(({ value }) => normalizeStringArray(value))
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  @MaxLength(MAX_TAG_LENGTH, { each: true })
  tags?: ImportMediaEntryInput["tags"];
}

export class NormalizedSearchResultDto implements NormalizedSearchResult {
  @IsEnum(providerNames)
  provider!: NormalizedSearchResult["provider"];

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().replace(/\s+/g, " ") : value
  )
  @IsString()
  @MaxLength(MAX_PROVIDER_ID_LENGTH)
  providerId!: string;

  @IsEnum(mediaTypes)
  mediaType!: NormalizedSearchResult["mediaType"];

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().replace(/\s+/g, " ") : value
  )
  @IsString()
  @MaxLength(MAX_TITLE_LENGTH)
  title!: string;

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  subtitle?: NormalizedSearchResult["subtitle"];

  @Transform(({ value }) => toOptionalNumber(value))
  @IsOptional()
  @IsNumber()
  year?: NormalizedSearchResult["year"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  imageUrl?: NormalizedSearchResult["imageUrl"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  @MaxLength(MAX_SUMMARY_LENGTH)
  summary?: NormalizedSearchResult["summary"];

  @Transform(({ value }) => normalizeOptionalString(value))
  @IsOptional()
  @IsString()
  creatorLine?: NormalizedSearchResult["creatorLine"];

  @Transform(({ value }) => normalizeStringArray(value))
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  barcodeCandidates?: NormalizedSearchResult["barcodeCandidates"];

  @Transform(({ value }) => toOptionalNumber(value))
  @IsOptional()
  @IsNumber()
  confidence?: NormalizedSearchResult["confidence"];
}

export class ImportMediaFromProviderRefDto
  implements ImportMediaRecordFromProviderRefRequest {
  @Equals("provider_ref")
  mode!: "provider_ref";

  @IsEnum(providerNames)
  provider!: ImportMediaRecordFromProviderRefRequest["provider"];

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().replace(/\s+/g, " ") : value
  )
  @IsString()
  @MaxLength(MAX_PROVIDER_ID_LENGTH)
  providerId!: string;

  @IsEnum(mediaTypes)
  mediaType!: ImportMediaRecordFromProviderRefRequest["mediaType"];

  @Type(() => ImportMediaEntryDto)
  @ValidateNested()
  @IsOptional()
  entry?: ImportMediaRecordFromProviderRefRequest["entry"];
}

export class ImportMediaFromSearchResultDto
  implements ImportMediaRecordFromSearchResultRequest {
  @Equals("search_result")
  mode!: "search_result";

  @Type(() => NormalizedSearchResultDto)
  @ValidateNested()
  result!: ImportMediaRecordFromSearchResultRequest["result"];

  @Type(() => ImportMediaEntryDto)
  @ValidateNested()
  @IsOptional()
  entry?: ImportMediaRecordFromSearchResultRequest["entry"];
}

export type ImportMediaDto =
  | ImportMediaFromProviderRefDto
  | ImportMediaFromSearchResultDto;

@Injectable()
export class ImportMediaDtoPipe
  implements PipeTransform<unknown, Promise<ImportMediaRecordRequest>>
{
  private readonly validationPipe = new ValidationPipe({
    whitelist: true,
    transform: true,
    forbidUnknownValues: true
  });

  async transform(
    value: unknown,
    metadata: ArgumentMetadata
  ): Promise<ImportMediaRecordRequest> {
    if (!isObjectRecord(value)) {
      throw new BadRequestException("Import request body must be an object.");
    }

    const metatype = getImportMetatype(value.mode);

    if (!metatype) {
      throw new BadRequestException(
        "mode must be either 'provider_ref' or 'search_result'."
      );
    }

    return (await this.validationPipe.transform(value, {
      ...metadata,
      metatype
    })) as ImportMediaRecordRequest;
  }
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

function toOptionalNumber(value: unknown): number | undefined {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : undefined;
  }

  if (typeof value !== "string" || value.trim().length === 0) {
    return undefined;
  }

  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function getImportMetatype(mode: unknown) {
  if (mode === "provider_ref") {
    return ImportMediaFromProviderRefDto;
  }

  if (mode === "search_result") {
    return ImportMediaFromSearchResultDto;
  }

  return null;
}
