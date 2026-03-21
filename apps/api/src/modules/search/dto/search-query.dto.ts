import { mediaTypes, type SearchQuery } from "@media-library/types";
import { Transform } from "class-transformer";
import {
  IsEnum,
  IsInt,
  IsOptional,
  IsString,
  Max,
  MaxLength,
  Min
} from "class-validator";

const MAX_QUERY_LENGTH = 240;
const MAX_LIMIT = 25;

export class SearchQueryDto implements SearchQuery {
  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().replace(/\s+/g, " ") : value
  )
  @IsString()
  @MaxLength(MAX_QUERY_LENGTH)
  q!: string;

  @IsEnum(mediaTypes)
  mediaType!: SearchQuery["mediaType"];

  @Transform(({ value }) => toOptionalInt(value))
  @IsOptional()
  @IsInt()
  @Min(1)
  @Max(MAX_LIMIT)
  limit?: number;
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
