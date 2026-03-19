import { Transform } from "class-transformer";
import { IsString, Length, Matches } from "class-validator";
import {
  NON_EMPTY_STRING_MIN_LENGTH,
  PASSWORD_MAX_LENGTH,
  USERNAME_MAX_LENGTH,
  USERNAME_MIN_LENGTH
} from "../auth.constants";

export class LoginDto {
  @Transform(({ value }) =>
    typeof value === "string" ? value.trim().toLowerCase() : value
  )
  @IsString()
  @Length(USERNAME_MIN_LENGTH, USERNAME_MAX_LENGTH)
  @Matches(/^[a-z0-9_]+$/, {
    message: "username may only contain lowercase letters, numbers, and underscores"
  })
  username!: string;

  @IsString()
  @Length(NON_EMPTY_STRING_MIN_LENGTH, PASSWORD_MAX_LENGTH)
  password!: string;
}
