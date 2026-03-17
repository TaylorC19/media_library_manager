import { Prop, Schema, SchemaFactory } from "@nestjs/mongoose";
import { HydratedDocument, Schema as MongooseSchema, Types } from "mongoose";
import { User } from "./user.schema";

export type SessionDocument = HydratedDocument<Session>;

@Schema({
  collection: "sessions",
  timestamps: {
    createdAt: true,
    updatedAt: false
  }
})
export class Session {
  @Prop({
    type: MongooseSchema.Types.ObjectId,
    ref: User.name,
    required: true,
    index: true
  })
  userId!: Types.ObjectId;

  @Prop({
    required: true,
    type: String
  })
  tokenHash!: string;

  @Prop({
    required: true
  })
  expiresAt!: Date;

  @Prop({
    required: true,
    default: Date.now
  })
  lastUsedAt!: Date;

  @Prop({
    default: null,
    type: String
  })
  userAgent!: string | null;

  @Prop({
    default: null,
    type: String
  })
  ipAddress!: string | null;

  createdAt?: Date;
}

export const SessionSchema = SchemaFactory.createForClass(Session);

SessionSchema.index({ expiresAt: 1 }, { expireAfterSeconds: 0 });
SessionSchema.index({ tokenHash: 1 }, { unique: true });
