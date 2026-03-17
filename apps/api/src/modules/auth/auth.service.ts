import {
  ConflictException,
  Injectable,
  InternalServerErrorException,
  UnauthorizedException
} from "@nestjs/common";
import { ConfigService } from "@nestjs/config";
import { InjectModel } from "@nestjs/mongoose";
import type {
  AuthSessionResponse,
  AuthUser,
  LoginRequest,
  RegisterRequest
} from "@media-library/types";
import {
  createHmac,
  randomBytes,
  scrypt as scryptCallback,
  timingSafeEqual
} from "node:crypto";
import { Model, Types } from "mongoose";
import {
  AUTH_COOKIE_NAME_FALLBACK,
  PASSWORD_HASH_KEY_LENGTH,
  PASSWORD_SCRYPT_BLOCK_SIZE,
  PASSWORD_SCRYPT_COST,
  PASSWORD_SCRYPT_PARALLELIZATION,
  SESSION_TTL_DAYS_FALLBACK
} from "./auth.constants";
import type { AuthResponse, RequestSessionContext } from "./auth.types";
import { LoginRateLimitService } from "./login-rate-limit.service";
import { Session, type SessionDocument } from "./schemas/session.schema";
import { User, type UserDocument } from "./schemas/user.schema";

interface AuthSessionLookupResult {
  sessionId: string;
  user: AuthUser;
}

interface AuthSessionResult {
  response: AuthSessionResponse;
  sessionToken: string;
}

interface AuthUserRecord {
  _id: Types.ObjectId | string;
  displayName?: string | null;
  username: string;
}

@Injectable()
export class AuthService {
  constructor(
    private readonly configService: ConfigService,
    private readonly loginRateLimitService: LoginRateLimitService,
    @InjectModel(User.name) private readonly userModel: Model<UserDocument>,
    @InjectModel(Session.name)
    private readonly sessionModel: Model<SessionDocument>
  ) {}

  async register(
    input: RegisterRequest,
    context: RequestSessionContext
  ): Promise<AuthSessionResult> {
    const username = this.normalizeUsername(input.username);
    const existingUser = await this.userModel.exists({ username });

    if (existingUser) {
      throw new ConflictException("That username is already in use.");
    }

    const user = await this.userModel.create({
      displayName: this.normalizeDisplayName(input.displayName),
      passwordHash: await this.hashPassword(input.password),
      settings: {
        futureSocialEnabled: false,
        profileVisibility: "private"
      },
      username
    });

    const sessionToken = await this.createSession(user._id, context);

    return {
      response: {
        user: this.toAuthUser(user)
      },
      sessionToken
    };
  }

  async login(
    input: LoginRequest,
    context: RequestSessionContext,
    existingSessionToken?: string
  ): Promise<AuthSessionResult> {
    const username = this.normalizeUsername(input.username);
    const rateLimitKey = this.getRateLimitKey(username, context.ipAddress);

    this.loginRateLimitService.assertWithinLimit(rateLimitKey);

    const user = await this.userModel.findOne({ username });

    if (!user) {
      this.loginRateLimitService.recordFailure(rateLimitKey);
      throw new UnauthorizedException("Invalid username or password.");
    }

    const isValidPassword = await this.verifyPassword(
      input.password,
      user.passwordHash
    );

    if (!isValidPassword) {
      this.loginRateLimitService.recordFailure(rateLimitKey);
      throw new UnauthorizedException("Invalid username or password.");
    }

    this.loginRateLimitService.reset(rateLimitKey);

    if (existingSessionToken) {
      await this.revokeSession(existingSessionToken);
    }

    const sessionToken = await this.createSession(user._id, context);

    return {
      response: {
        user: this.toAuthUser(user)
      },
      sessionToken
    };
  }

  async logout(sessionToken?: string): Promise<void> {
    if (!sessionToken) {
      return;
    }

    await this.revokeSession(sessionToken);
  }

  async getSessionUser(
    sessionToken: string
  ): Promise<AuthSessionLookupResult | null> {
    const tokenHash = this.hashSessionToken(sessionToken);
    const session = await this.sessionModel.findOne({ tokenHash }).lean();

    if (!session) {
      return null;
    }

    if (session.expiresAt.getTime() <= Date.now()) {
      await this.sessionModel.deleteOne({ _id: session._id });
      return null;
    }

    const user = await this.userModel.findById(session.userId).lean();

    if (!user) {
      await this.sessionModel.deleteOne({ _id: session._id });
      return null;
    }

    return {
      sessionId: session._id.toString(),
      user: this.toAuthUser(user)
    };
  }

  async touchSession(sessionId: string): Promise<void> {
    await this.sessionModel.updateOne(
      { _id: sessionId },
      {
        $set: {
          lastUsedAt: new Date()
        }
      }
    );
  }

  getSessionCookieName(): string {
    return (
      this.configService.get<string>("SESSION_COOKIE_NAME") ??
      AUTH_COOKIE_NAME_FALLBACK
    );
  }

  setSessionCookie(response: AuthResponse, sessionToken: string): void {
    response.cookie(this.getSessionCookieName(), sessionToken, {
      expires: new Date(Date.now() + this.getSessionTtlMs()),
      httpOnly: true,
      path: "/",
      sameSite: "lax",
      secure: this.isProduction()
    });
  }

  clearSessionCookie(response: AuthResponse): void {
    response.clearCookie(this.getSessionCookieName(), {
      httpOnly: true,
      path: "/",
      sameSite: "lax",
      secure: this.isProduction()
    });
  }

  private async createSession(
    userId: Types.ObjectId,
    context: RequestSessionContext
  ): Promise<string> {
    const sessionToken = randomBytes(32).toString("base64url");

    await this.sessionModel.create({
      expiresAt: new Date(Date.now() + this.getSessionTtlMs()),
      ipAddress: context.ipAddress,
      lastUsedAt: new Date(),
      tokenHash: this.hashSessionToken(sessionToken),
      userAgent: context.userAgent,
      userId
    });

    return sessionToken;
  }

  private async revokeSession(sessionToken: string): Promise<void> {
    await this.sessionModel.deleteOne({
      tokenHash: this.hashSessionToken(sessionToken)
    });
  }

  private toAuthUser(user: AuthUserRecord): AuthUser {
    return {
      displayName: user.displayName ?? null,
      id: user._id.toString(),
      username: user.username
    };
  }

  private normalizeUsername(username: string): string {
    return username.trim().toLowerCase();
  }

  private normalizeDisplayName(displayName?: string): string | null {
    const value = displayName?.trim();

    return value ? value : null;
  }

  private getRateLimitKey(username: string, ipAddress: string | null): string {
    return `${ipAddress ?? "unknown"}:${username}`;
  }

  private async hashPassword(password: string): Promise<string> {
    const salt = randomBytes(16);
    const derivedKey = await this.derivePasswordKey(password, salt, {
      N: PASSWORD_SCRYPT_COST,
      p: PASSWORD_SCRYPT_PARALLELIZATION,
      r: PASSWORD_SCRYPT_BLOCK_SIZE
    });

    return [
      "scrypt",
      PASSWORD_SCRYPT_COST,
      PASSWORD_SCRYPT_BLOCK_SIZE,
      PASSWORD_SCRYPT_PARALLELIZATION,
      salt.toString("base64url"),
      derivedKey.toString("base64url")
    ].join("$");
  }

  private async verifyPassword(
    password: string,
    storedPasswordHash: string
  ): Promise<boolean> {
    const [
      algorithm,
      cost,
      blockSize,
      parallelization,
      salt,
      expectedHash
    ] = storedPasswordHash.split("$");

    if (
      algorithm !== "scrypt" ||
      !cost ||
      !blockSize ||
      !parallelization ||
      !salt ||
      !expectedHash
    ) {
      throw new InternalServerErrorException("Invalid password hash format.");
    }

    const derivedKey = await this.derivePasswordKey(
      password,
      Buffer.from(salt, "base64url"),
      {
        N: Number.parseInt(cost, 10),
        p: Number.parseInt(parallelization, 10),
        r: Number.parseInt(blockSize, 10)
      }
    );

    const expectedBuffer = Buffer.from(expectedHash, "base64url");

    if (derivedKey.length !== expectedBuffer.length) {
      return false;
    }

    return timingSafeEqual(derivedKey, expectedBuffer);
  }

  private hashSessionToken(sessionToken: string): string {
    return createHmac("sha256", this.getSessionSecret())
      .update(sessionToken)
      .digest("hex");
  }

  private derivePasswordKey(
    password: string,
    salt: Buffer,
    options: {
      N: number;
      p: number;
      r: number;
    }
  ): Promise<Buffer> {
    return new Promise((resolve, reject) => {
      scryptCallback(
        password,
        salt,
        PASSWORD_HASH_KEY_LENGTH,
        options,
        (error, derivedKey) => {
          if (error) {
            reject(error);
            return;
          }

          resolve(derivedKey);
        }
      );
    });
  }

  private getSessionSecret(): string {
    return this.configService.get<string>("SESSION_SECRET") ?? "replace_me";
  }

  private getSessionTtlMs(): number {
    const sessionTtlDays = Number.parseInt(
      this.configService.get<string>("SESSION_TTL_DAYS") ??
        `${SESSION_TTL_DAYS_FALLBACK}`,
      10
    );

    return (
      (Number.isFinite(sessionTtlDays) && sessionTtlDays > 0
        ? sessionTtlDays
        : SESSION_TTL_DAYS_FALLBACK) *
      24 *
      60 *
      60 *
      1000
    );
  }

  private isProduction(): boolean {
    return this.configService.get<string>("NODE_ENV") === "production";
  }
}
