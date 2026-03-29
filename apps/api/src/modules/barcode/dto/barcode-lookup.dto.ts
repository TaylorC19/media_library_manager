import { mediaTypes, type BarcodeLookupRequest } from "@media-library/types";
import { Transform } from "class-transformer";
import { IsEnum, IsOptional, IsString, MaxLength } from "class-validator";

export class BarcodeLookupDto implements BarcodeLookupRequest {
  @Transform(({ value }) => (typeof value === "string" ? value.trim() : value))
  @IsString()
  @MaxLength(64)
  barcode!: string;

  @IsOptional()
  @IsEnum(mediaTypes)
  preferredMediaType?: BarcodeLookupRequest["preferredMediaType"];
}
