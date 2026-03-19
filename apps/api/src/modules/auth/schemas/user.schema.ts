import { Prop, Schema, SchemaFactory } from "@nestjs/mongoose";
import type { UserSettings } from "@media-library/types";
import type { HydratedDocument } from "mongoose";

export type UserDocument = HydratedDocument<User>;

@Schema({
  collection: "users",
  timestamps: true
})
export class User {
  @Prop({
    required: true,
    type: String,
    lowercase: true,
    trim: true
  })
  username!: string;

  @Prop({ required: true, type: String })
  passwordHash!: string;

  @Prop({
    default: null,
    type: String,
    trim: true
  })
  displayName!: string | null;

  @Prop({
    type: {
      profileVisibility: {
        type: String,
        enum: ["private"],
        default: "private"
      },
      futureSocialEnabled: {
        type: Boolean,
        default: false
      }
    },
    default: {
      profileVisibility: "private",
      futureSocialEnabled: false
    }
  })
  settings!: UserSettings;

  createdAt?: Date;
  updatedAt?: Date;
}

export const UserSchema = SchemaFactory.createForClass(User);

UserSchema.index({ username: 1 }, { unique: true });
