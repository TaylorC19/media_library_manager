import { ValidationPipe } from "@nestjs/common";
import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";
import type { AuthenticatedRequest } from "./modules/auth/auth.types";

function parseCookies(cookieHeader?: string): Record<string, string> {
  if (!cookieHeader) {
    return {};
  }

  return cookieHeader
    .split(";")
    .map((part) => part.trim())
    .filter(Boolean)
    .reduce<Record<string, string>>((cookies, cookiePart) => {
      const separatorIndex = cookiePart.indexOf("=");

      if (separatorIndex <= 0) {
        return cookies;
      }

      const key = cookiePart.slice(0, separatorIndex).trim();
      const value = cookiePart.slice(separatorIndex + 1).trim();

      cookies[key] = decodeURIComponent(value);
      return cookies;
    }, {});
}

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const corsOrigin = process.env.CORS_ORIGIN ?? "http://localhost:3000";
  const port = Number.parseInt(process.env.API_PORT ?? "4000", 10);

  app.enableCors({
    origin: corsOrigin,
    credentials: true
  });

  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      forbidUnknownValues: true
    })
  );

  app.use(
    (
      request: AuthenticatedRequest,
      _response: unknown,
      next: () => void
    ) => {
      request.cookies = parseCookies(request.headers.cookie);
      next();
    }
  );

  await app.listen(port);
}

void bootstrap();
