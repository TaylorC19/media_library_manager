import { HttpException, HttpStatus, Injectable } from "@nestjs/common";
import {
  LOGIN_RATE_LIMIT_MAX_ATTEMPTS,
  LOGIN_RATE_LIMIT_WINDOW_MS
} from "./auth.constants";

interface RateLimitEntry {
  count: number;
  firstAttemptAt: number;
}

@Injectable()
export class LoginRateLimitService {
  private readonly attempts = new Map<string, RateLimitEntry>();

  assertWithinLimit(key: string): void {
    const now = Date.now();
    const entry = this.attempts.get(key);

    if (!entry) {
      return;
    }

    if (now - entry.firstAttemptAt >= LOGIN_RATE_LIMIT_WINDOW_MS) {
      this.attempts.delete(key);
      return;
    }

    if (entry.count >= LOGIN_RATE_LIMIT_MAX_ATTEMPTS) {
      throw new HttpException(
        "Too many login attempts. Please try again later.",
        HttpStatus.TOO_MANY_REQUESTS
      );
    }
  }

  recordFailure(key: string): void {
    const now = Date.now();
    const entry = this.attempts.get(key);

    if (!entry || now - entry.firstAttemptAt >= LOGIN_RATE_LIMIT_WINDOW_MS) {
      this.attempts.set(key, {
        count: 1,
        firstAttemptAt: now
      });
      return;
    }

    this.attempts.set(key, {
      ...entry,
      count: entry.count + 1
    });
  }

  reset(key: string): void {
    this.attempts.delete(key);
  }
}
