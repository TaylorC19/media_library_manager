import { Transform } from "class-transformer";
import {
  IsOptional,
  IsString,
  Length,
  Matches
} from "class-validator";

export class RegisterDto {
  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().toLowerCase() : value
  )
  @IsString()
  @Length(3, 32)
  @Matches(/^[a-z0-9_]+$/, {
    message: "username may only contain lowercase letters, numbers, and underscores"
  })
  username!: string;

  @IsString()
  @Length(8, 72)
  password!: string;

  @Transform(({ value }) =>
    typeof value === "string" ? value.trim() : value
  )
  @IsOptional()
  @IsString()
  @Length(1, 64)
  displayName?: string;
}
