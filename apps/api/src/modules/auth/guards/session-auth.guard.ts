import {
  CanActivate,
  ExecutionContext,
  Injectable,
  UnauthorizedException
} from "@nestjs/common";
import { Reflector } from "@nestjs/core";
import { IS_PUBLIC_KEY } from "../auth.constants";
import { AuthService } from "../auth.service";
import type { AuthenticatedRequest } from "../auth.types";

@Injectable()
export class SessionAuthGuard implements CanActivate {
  constructor(
    private readonly reflector: Reflector,
    private readonly authService: AuthService
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const isPublic = this.reflector.getAllAndOverride<boolean>(IS_PUBLIC_KEY, [
      context.getHandler(),
      context.getClass()
    ]);

    if (isPublic) {
      return true;
    }

    const request = context.switchToHttp().getRequest<AuthenticatedRequest>();

    if (request.method === "OPTIONS") {
      return true;
    }

    const sessionToken =
      request.cookies?.[this.authService.getSessionCookieName()];

    if (!sessionToken) {
      throw new UnauthorizedException("Authentication required.");
    }

    const sessionUser = await this.authService.getSessionUser(sessionToken);

    if (!sessionUser) {
      throw new UnauthorizedException("Authentication required.");
    }

    request.authUser = sessionUser.user;
    request.sessionId = sessionUser.sessionId;
    request.sessionToken = sessionToken;

    await this.authService.touchSession(sessionUser.sessionId);

    return true;
  }
}
