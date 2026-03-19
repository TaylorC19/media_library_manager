import { Module } from "@nestjs/common";
import { APP_GUARD } from "@nestjs/core";
import { MongooseModule } from "@nestjs/mongoose";
import { AuthController } from "./auth.controller";
import { AuthService } from "./auth.service";
import { SessionAuthGuard } from "./guards/session-auth.guard";
import { LoginRateLimitService } from "./login-rate-limit.service";
import { SessionsRepository } from "./repositories/sessions.repository";
import { UsersRepository } from "./repositories/users.repository";
import { Session, SessionSchema } from "./schemas/session.schema";
import { User, UserSchema } from "./schemas/user.schema";

@Module({
  controllers: [AuthController],
  exports: [AuthService],
  imports: [
    MongooseModule.forFeature([
      {
        name: User.name,
        schema: UserSchema
      },
      {
        name: Session.name,
        schema: SessionSchema
      }
    ])
  ],
  providers: [
    AuthService,
    LoginRateLimitService,
    UsersRepository,
    SessionsRepository,
    {
      provide: APP_GUARD,
      useClass: SessionAuthGuard
    }
  ]
})
export class AuthModule {}
