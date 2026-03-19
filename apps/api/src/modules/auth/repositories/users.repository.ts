import { Injectable } from "@nestjs/common";
import { InjectModel } from "@nestjs/mongoose";
import type { User as UserEntity, UserSettings } from "@media-library/types";
import { Model, Types } from "mongoose";
import { User, type UserDocument } from "../schemas/user.schema";

interface CreateUserInput {
  username: string;
  passwordHash: string;
  displayName: string | null;
  settings: UserSettings;
}

@Injectable()
export class UsersRepository {
  constructor(
    @InjectModel(User.name) private readonly userModel: Model<UserDocument>
  ) {}

  async create(input: CreateUserInput): Promise<UserEntity> {
    const user = await this.userModel.create(input);
    return this.toDomain(user);
  }

  async existsByUsername(username: string): Promise<boolean> {
    const existingUser = await this.userModel.exists({ username });
    return existingUser !== null;
  }

  async findById(id: string): Promise<UserEntity | null> {
    if (!Types.ObjectId.isValid(id)) {
      return null;
    }

    const user = await this.userModel.findById(id).exec();
    return user ? this.toDomain(user) : null;
  }

  async findByUsername(username: string): Promise<UserEntity | null> {
    const user = await this.userModel.findOne({ username }).exec();
    return user ? this.toDomain(user) : null;
  }

  private toDomain(user: UserDocument): UserEntity {
    return {
      id: user._id.toString(),
      username: user.username,
      passwordHash: user.passwordHash,
      displayName: user.displayName ?? null,
      settings: {
        futureSocialEnabled: user.settings.futureSocialEnabled,
        profileVisibility: user.settings.profileVisibility
      },
      createdAt: user.createdAt?.toISOString() ?? new Date().toISOString(),
      updatedAt: user.updatedAt?.toISOString() ?? new Date().toISOString()
    };
  }
}
