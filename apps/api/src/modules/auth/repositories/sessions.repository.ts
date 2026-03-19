import { Injectable } from "@nestjs/common";
import { InjectModel } from "@nestjs/mongoose";
import type { Session as SessionEntity } from "@media-library/types";
import { Model, Types } from "mongoose";
import { Session, type SessionDocument } from "../schemas/session.schema";

interface CreateSessionInput {
  userId: string;
  tokenHash: string;
  expiresAt: Date;
  lastUsedAt: Date;
  userAgent: string | null;
  ipAddress: string | null;
}

@Injectable()
export class SessionsRepository {
  constructor(
    @InjectModel(Session.name) private readonly sessionModel: Model<SessionDocument>
  ) {}

  async create(input: CreateSessionInput): Promise<SessionEntity> {
    const session = await this.sessionModel.create({
      ...input,
      userId: new Types.ObjectId(input.userId)
    });

    return this.toDomain(session);
  }

  async deleteById(id: string): Promise<void> {
    if (!Types.ObjectId.isValid(id)) {
      return;
    }

    await this.sessionModel.deleteOne({ _id: id });
  }

  async deleteByTokenHash(tokenHash: string): Promise<void> {
    await this.sessionModel.deleteOne({ tokenHash });
  }

  async findByTokenHash(tokenHash: string): Promise<SessionEntity | null> {
    const session = await this.sessionModel.findOne({ tokenHash }).exec();
    return session ? this.toDomain(session) : null;
  }

  async touch(id: string, lastUsedAt: Date): Promise<void> {
    if (!Types.ObjectId.isValid(id)) {
      return;
    }

    await this.sessionModel.updateOne(
      { _id: id },
      {
        $set: {
          lastUsedAt
        }
      }
    );
  }

  private toDomain(session: SessionDocument): SessionEntity {
    return {
      id: session._id.toString(),
      userId: session.userId.toString(),
      tokenHash: session.tokenHash,
      expiresAt: session.expiresAt.toISOString(),
      createdAt: session.createdAt?.toISOString() ?? new Date().toISOString(),
      lastUsedAt: session.lastUsedAt.toISOString(),
      userAgent: session.userAgent ?? null,
      ipAddress: session.ipAddress ?? null
    };
  }
}
