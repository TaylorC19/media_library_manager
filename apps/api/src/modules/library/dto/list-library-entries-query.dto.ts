import {
  libraryBuckets,
  mediaTypes,
  type ListLibraryEntriesQuery
} from "@media-library/types";
import { Transform } from "class-transformer";
import { IsEnum, IsInt, IsOptional, IsString, Max, Min } from "class-validator";

const MAX_PAGE_SIZE = 1000;

export class ListLibraryEntriesQueryDto implements ListLibraryEntriesQuery {
  @IsOptional()
  @IsEnum(libraryBuckets)
  bucket?: ListLibraryEntriesQuery["bucket"];

  @IsOptional()
  @IsEnum(mediaTypes)
  mediaType?: ListLibraryEntriesQuery["mediaType"];

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim() || undefined : undefined
  )
  @IsOptional()
  @IsString()
  tag?: string;

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim() || undefined : undefined
  )
  @IsOptional()
  @IsString()
  search?: string;

  @Transform(({ value }) => toOptionalInt(value))
  @IsOptional()
  @IsInt()
  @Min(1)
  page?: number;

  @Transform(({ value }) => toOptionalInt(value))
  @IsOptional()
  @IsInt()
  @Min(1)
  @Max(MAX_PAGE_SIZE)
  pageSize?: number;
}

function toOptionalInt(value: unknown): number | undefined {
  if (typeof value !== "string" || value.trim().length === 0) {
    return undefined;
  }

  const parsed = Number.parseInt(value, 10);
  return Number.isNaN(parsed) ? undefined : parsed;
}
