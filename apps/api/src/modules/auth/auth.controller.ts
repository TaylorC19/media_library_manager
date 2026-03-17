import {
  Body,
  Controller,
  Get,
  HttpCode,
  HttpStatus,
  InternalServerErrorException,
  Post,
  Req,
  Res
} from "@nestjs/common";
import type { AuthSessionResponse, AuthUser } from "@media-library/types";
import { AuthService } from "./auth.service";
import type { AuthenticatedRequest, AuthResponse } from "./auth.types";
import { CurrentUser } from "./decorators/current-user.decorator";
import { Public } from "./decorators/public.decorator";
import { LoginDto } from "./dto/login.dto";
import { RegisterDto } from "./dto/register.dto";

@Controller("auth")
export class AuthController {
  constructor(private readonly authService: AuthService) {}

  @Public()
  @Post("register")
  async register(
    @Body() body: RegisterDto,
    @Req() request: AuthenticatedRequest,
    @Res({ passthrough: true }) response: AuthResponse
  ): Promise<AuthSessionResponse> {
    const result = await this.authService.register(body, {
      ipAddress: request.ip ?? request.socket.remoteAddress ?? null,
      userAgent: request.get?.("user-agent") ?? null
    });

    this.authService.setSessionCookie(response, result.sessionToken);

    return result.response;
  }

  @Public()
  @Post("login")
  async login(
    @Body() body: LoginDto,
    @Req() request: AuthenticatedRequest,
    @Res({ passthrough: true }) response: AuthResponse
  ): Promise<AuthSessionResponse> {
    const result = await this.authService.login(
      body,
      {
        ipAddress: request.ip ?? request.socket.remoteAddress ?? null,
        userAgent: request.get?.("user-agent") ?? null
      },
      request.cookies?.[this.authService.getSessionCookieName()]
    );

    this.authService.setSessionCookie(response, result.sessionToken);

    return result.response;
  }

  @Public()
  @Post("logout")
  @HttpCode(HttpStatus.OK)
  async logout(
    @Req() request: AuthenticatedRequest,
    @Res({ passthrough: true }) response: AuthResponse
  ): Promise<{ success: true }> {
    await this.authService.logout(
      request.cookies?.[this.authService.getSessionCookieName()]
    );
    this.authService.clearSessionCookie(response);

    return { success: true };
  }

  @Get("me")
  getCurrentSession(
    @CurrentUser() user: AuthUser | null
  ): AuthSessionResponse {
    if (!user) {
      throw new InternalServerErrorException(
        "Authenticated user missing from request context."
      );
    }

    return { user };
  }
}
