import { Transform } from "class-transformer";
import { IsString, Length, Matches } from "class-validator";

export class LoginDto {
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
  @Length(1, 72)
  password!: string;
}
